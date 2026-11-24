'use strict'
// Credential persistence. Passwords are only ever written encrypted with the OS
// keystore via Electron safeStorage. If encryption is unavailable (or only the
// weak "basic_text" Linux backend is available) the password is NOT persisted:
// it lives in memory for the session. Nothing here logs values.
const fs = require('node:fs')
const path = require('node:path')

function createStore({ dir, safeStorage }) {
  const file = path.join(dir, 'connection.json')
  let mem = { serverUrl: '', username: '', password: '' }

  function encryptionOk() {
    try {
      if (!safeStorage || !safeStorage.isEncryptionAvailable()) return false
      if (typeof safeStorage.getSelectedStorageBackend === 'function' && safeStorage.getSelectedStorageBackend() === 'basic_text') return false
      return true
    } catch { return false }
  }

  function load() {
    try {
      const j = JSON.parse(fs.readFileSync(file, 'utf8'))
      mem.serverUrl = typeof j.serverUrl === 'string' ? j.serverUrl : ''
      mem.username = typeof j.username === 'string' ? j.username : ''
      if (typeof j.passwordEnc === 'string' && encryptionOk()) {
        mem.password = safeStorage.decryptString(Buffer.from(j.passwordEnc, 'base64'))
      }
    } catch { /* first run or unreadable: start empty */ }
    return { ...mem }
  }

  function save({ serverUrl, username, password }) {
    mem = { serverUrl, username, password }
    const persisted = encryptionOk()
    const doc = { serverUrl, username }
    if (persisted && password) doc.passwordEnc = safeStorage.encryptString(password).toString('base64')
    fs.mkdirSync(dir, { recursive: true, mode: 0o700 })
    const tmp = file + '.tmp'
    fs.writeFileSync(tmp, JSON.stringify(doc), { mode: 0o600 })
    fs.renameSync(tmp, file)
    return { passwordPersisted: persisted && !!password }
  }

  return { load, save, get: () => ({ ...mem }), encryptionOk, file }
}

module.exports = { createStore }
