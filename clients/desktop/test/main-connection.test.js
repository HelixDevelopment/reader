'use strict'
// Drives the real main.js against a mocked `electron` module (no window is ever
// opened): credentials saved for server A must never be sent to server B.
const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const Module = require('node:module')

async function boot() {
  const userData = fs.mkdtempSync(path.join(os.tmpdir(), 'hrd-main-'))
  process.env.HELIX_DESKTOP_USER_DATA = userData
  for (const k of ['HELIX_SERVER_URL', 'HELIX_USERNAME', 'HELIX_PASSWORD_FILE']) delete process.env[k]
  const handlers = {}; let appHandler; let ready
  const win = { reloads: 0, getParentWindow: () => null, isDestroyed: () => false, reload() { this.reloads++ } }
  const electron = {
    app: { setPath() {}, commandLine: { appendSwitch() {} }, getPath: () => userData, whenReady: () => ({ then: (f) => { ready = f } }), on() {}, quit() {} },
    BrowserWindow: Object.assign(function () { return { webContents: { on() {}, setWindowOpenHandler() {} }, setMenu() {}, loadURL() {}, once() {} } }, { getAllWindows: () => [win] }),
    Menu: { buildFromTemplate: () => ({}) },
    ipcMain: { handle: (n, f) => { handlers[n] = f } },
    protocol: { registerSchemesAsPrivileged() {}, handle: (_s, f) => { appHandler = f } },
    safeStorage: { isEncryptionAvailable: () => false },
    session: { defaultSession: { setPermissionRequestHandler() {}, setPermissionCheckHandler() {}, webRequest: { onBeforeRequest() {} } } },
  }
  const origLoad = Module._load
  Module._load = function (request, ...rest) { return request === 'electron' ? electron : origLoad.call(this, request, ...rest) }
  const mainPath = require.resolve('../main.js')
  delete require.cache[mainPath]
  require(mainPath)
  Module._load = origLoad
  const calls = []
  global.fetch = async (url, init) => { calls.push({ url, headers: init.headers }); return new Response('{}', { status: 200 }) }
  ready()
  const ev = { senderFrame: { url: 'app://helix/__shell/connection.html' } }
  const call = (n, v) => handlers[n](ev, v)
  const api = async () => { await appHandler(new Request('app://helix/v1/works')); return calls[calls.length - 1] }
  return { call, api, handlers, win }
}

test('blank password is kept while the server origin is unchanged', async () => {
  const m = await boot()
  await m.call('helix:save-connection', { serverUrl: 'https://a.example:8443/lib', username: 'u', password: 'secret-A' })
  await m.call('helix:save-connection', { serverUrl: 'https://a.example:8443/other', username: 'u', password: '' })
  const c = await m.api()
  assert.equal(c.url, 'https://a.example:8443/other/v1/works')
  assert.equal(c.headers.authorization, 'Basic ' + Buffer.from('u:secret-A').toString('base64'))
})

test('blank password is NOT carried over to a different server origin', async () => {
  const m = await boot()
  await m.call('helix:save-connection', { serverUrl: 'https://a.example', username: 'u', password: 'secret-A' })
  for (const next of ['http://b.example', 'http://a.example', 'https://a.example:444', 'https://evil.example']) {
    await m.call('helix:save-connection', { serverUrl: next, username: 'u', password: '' })
    const c = await m.api()
    assert.ok(c.url.startsWith(next), c.url)
    assert.ok(!c.headers.authorization || c.headers.authorization === 'Basic ' + Buffer.from('u:').toString('base64'), `credentials leaked to ${next}: ${c.headers.authorization}`)
    const info = await m.call('helix:get-connection')
    assert.equal(info.hasPassword, false, next)
    await m.call('helix:save-connection', { serverUrl: 'https://a.example', username: 'u', password: 'secret-A' })
  }
})

test('an explicitly typed password is used for the new origin', async () => {
  const m = await boot()
  await m.call('helix:save-connection', { serverUrl: 'https://a.example', username: 'u', password: 'secret-A' })
  await m.call('helix:save-connection', { serverUrl: 'http://b.example', username: 'u', password: 'secret-B' })
  assert.equal((await m.api()).headers.authorization, 'Basic ' + Buffer.from('u:secret-B').toString('base64'))
})

test('forget-password clears the stored password, reloads, and is shell-only', async () => {
  const m = await boot()
  await m.call('helix:save-connection', { serverUrl: 'https://a.example', username: 'u', password: 'secret-A' })
  const before = m.win.reloads
  const r = await m.call('helix:forget-password')
  assert.equal(r.ok, true)
  assert.equal((await m.call('helix:get-connection')).hasPassword, false)
  assert.equal((await m.api()).headers.authorization, 'Basic ' + Buffer.from('u:').toString('base64'))
  assert.ok(m.win.reloads > before)
  const persisted = JSON.parse(fs.readFileSync(path.join(process.env.HELIX_DESKTOP_USER_DATA, 'connection.json'), 'utf8'))
  assert.equal(persisted.passwordEnc, undefined)
  assert.throws(() => m.handlers['helix:forget-password']({ senderFrame: { url: 'app://helix/index.html' } }), /forbidden/)
})

test('connection UI and preload expose a forget-password action', () => {
  const html = fs.readFileSync(path.join(__dirname, '../renderer/connection.html'), 'utf8')
  assert.match(html, /id="forget"/)
  assert.match(fs.readFileSync(path.join(__dirname, '../preload.js'), 'utf8'), /helix:forget-password/)
  assert.match(fs.readFileSync(path.join(__dirname, '../renderer/connection.js'), 'utf8'), /forgetPassword/)
})
