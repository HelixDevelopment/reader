'use strict'
// Helix Reader Linux desktop shell. Loads the built clients/web bundle from a
// privileged app:// scheme (no open TCP port), proxies API reads to the configured
// Helix server from the main process (the server sends no CORS headers), and keeps
// credentials in the main process (safeStorage), never in the renderer.
const { app, BrowserWindow, Menu, ipcMain, protocol, safeStorage, session } = require('electron')
const fs = require('node:fs')
const path = require('node:path')
const { APP_ORIGIN } = require('./lib/csp')
const { isAllowedNavigation, isAllowedRequest, normalizeServerUrl } = require('./lib/allowlist')
const { createHandler } = require('./lib/app-handler')
const { createStore } = require('./lib/secrets')
const { chromiumSwitches, apiFetchFor } = require('./lib/transport')

if (process.env.HELIX_DESKTOP_USER_DATA) app.setPath('userData', process.env.HELIX_DESKTOP_USER_DATA)
// Containers without user namespaces / suid sandbox need this; never set by default.
if (process.env.HELIX_DESKTOP_NO_CHROMIUM_SANDBOX === '1') app.commandLine.appendSwitch('no-sandbox')

// HTTP/3 (QUIC): Chromium's own network stack; opt-in private-CA settings are validated in lib/transport.js.
for (const [name, value] of chromiumSwitches(process.env)) value === undefined ? app.commandLine.appendSwitch(name) : app.commandLine.appendSwitch(name, value)

const WEB_ROOT = process.env.HELIX_DESKTOP_WEB_ROOT || path.join(__dirname, 'web')
const SHELL_ROOT = path.join(__dirname, 'renderer')
const DEVTOOLS = process.env.HELIX_DESKTOP_DEVTOOLS === '1'

protocol.registerSchemesAsPrivileged([{ scheme: 'app', privileges: { standard: true, secure: true, supportFetchAPI: true, stream: true } }])

let store
function originOf(raw) {
  const n = normalizeServerUrl(raw)
  return n ? new URL(n).origin : null
}
function sameOrigin(a, b) { const o = originOf(a); return o !== null && o === originOf(b) }
let conn = { serverUrl: '', username: '', password: '' }

function readSecretFile(p) {
  try { return fs.readFileSync(p, 'utf8').replace(/\r?\n$/, '') } catch { return '' }
}
function loadConnection() {
  const stored = store.load()
  conn = { ...stored }
  if (process.env.HELIX_SERVER_URL) conn.serverUrl = process.env.HELIX_SERVER_URL
  if (process.env.HELIX_USERNAME) conn.username = process.env.HELIX_USERNAME
  if (process.env.HELIX_PASSWORD_FILE) conn.password = readSecretFile(process.env.HELIX_PASSWORD_FILE)
}

const WEB_PREFS = { contextIsolation: true, nodeIntegration: false, sandbox: true, webSecurity: true, allowRunningInsecureContent: false, webviewTag: false, devTools: DEVTOOLS }

function lockDown(contents) {
  contents.on('will-navigate', (e, url) => { if (!isAllowedNavigation(url)) e.preventDefault() })
  contents.on('will-redirect', (e, url) => { if (!isAllowedNavigation(url)) e.preventDefault() })
  contents.on('will-attach-webview', (e) => e.preventDefault())
  contents.setWindowOpenHandler(() => ({ action: 'deny' }))
}

function openConnectionWindow(parent) {
  const w = new BrowserWindow({ parent, modal: !!parent, width: 460, height: 420, title: 'Helix Reader connection', autoHideMenuBar: true, show: false,
    webPreferences: { ...WEB_PREFS, preload: path.join(__dirname, 'preload.js') } })
  lockDown(w.webContents)
  w.once('ready-to-show', () => w.show())
  w.loadURL(APP_ORIGIN + '/__shell/connection.html')
  return w
}

function createMainWindow() {
  const w = new BrowserWindow({ width: 1180, height: 820, minWidth: 360, minHeight: 480, title: 'Helix Reader', show: !process.env.HELIX_DESKTOP_HIDDEN, backgroundColor: '#ffffff', webPreferences: WEB_PREFS })
  lockDown(w.webContents)
  const menu = Menu.buildFromTemplate([
    { label: 'Helix Reader', submenu: [{ label: 'Connection…', accelerator: 'CmdOrCtrl+,', click: () => openConnectionWindow(w) }, { label: 'Reload library', accelerator: 'CmdOrCtrl+R', click: () => w.reload() }, { type: 'separator' }, { role: 'quit' }] },
    { role: 'viewMenu', submenu: [{ role: 'resetZoom' }, { role: 'zoomIn' }, { role: 'zoomOut' }, { role: 'togglefullscreen' }] },
  ])
  w.setMenu(menu)
  w.loadURL(APP_ORIGIN + '/index.html')
  return w
}

ipcMain.handle('helix:get-connection', (e) => {
  if (!e.senderFrame || !e.senderFrame.url.startsWith(APP_ORIGIN + '/__shell/')) throw new Error('forbidden')
  return { serverUrl: conn.serverUrl, username: conn.username, hasPassword: !!conn.password, encryptionAvailable: store.encryptionOk() }
})
ipcMain.handle('helix:save-connection', (e, v) => {
  if (!e.senderFrame || !e.senderFrame.url.startsWith(APP_ORIGIN + '/__shell/')) throw new Error('forbidden')
  const serverUrl = typeof v?.serverUrl === 'string' ? v.serverUrl.trim() : ''
  if (serverUrl !== '' && !normalizeServerUrl(serverUrl)) return { ok: false, error: 'Enter a full http:// or https:// address without credentials, query or fragment.' }
  const username = typeof v?.username === 'string' ? v.username : ''
  // Blank = keep the stored password, but only for the same server origin: otherwise
  // credentials saved for host A would be sent to host B (possibly over plain http).
  const typed = typeof v?.password === 'string' ? v.password : ''
  const password = typed !== '' ? typed : (sameOrigin(conn.serverUrl, serverUrl) ? conn.password : '')
  const r = store.save({ serverUrl, username, password })
  conn = { serverUrl, username, password }
  for (const w of BrowserWindow.getAllWindows()) if (!w.getParentWindow() && !w.isDestroyed()) w.reload()
  return { ok: true, passwordPersisted: r.passwordPersisted }
})

ipcMain.handle('helix:forget-password', (e) => {
  if (!e.senderFrame || !e.senderFrame.url.startsWith(APP_ORIGIN + '/__shell/')) throw new Error('forbidden')
  store.save({ serverUrl: conn.serverUrl, username: conn.username, password: '' })
  conn = { ...conn, password: '' }
  for (const w of BrowserWindow.getAllWindows()) if (!w.getParentWindow() && !w.isDestroyed()) w.reload()
  return { ok: true }
})

app.on('web-contents-created', (_e, contents) => lockDown(contents))

app.whenReady().then(() => {
  store = createStore({ dir: app.getPath('userData'), safeStorage })
  loadConnection()
  // API forwarding goes through a dedicated in-memory Chromium session: Chromium network stack (h2/h3, Brotli), and
  // not the default session whose webRequest filter only admits app:// requests.
  const apiFetch = apiFetchFor(session.fromPartition('helix-api'))
  protocol.handle('app', createHandler({ getConn: () => conn, webRoot: WEB_ROOT, shellRoot: SHELL_ROOT, fetchImpl: apiFetch }))
  const ses = session.defaultSession
  ses.setPermissionRequestHandler((_wc, _perm, cb) => cb(false))
  ses.setPermissionCheckHandler(() => false)
  ses.webRequest.onBeforeRequest((d, cb) => cb({ cancel: !isAllowedRequest(d.url, { devtools: DEVTOOLS }) }))
  createMainWindow()
  app.on('activate', () => { if (BrowserWindow.getAllWindows().length === 0) createMainWindow() })
})

app.on('window-all-closed', () => app.quit())
