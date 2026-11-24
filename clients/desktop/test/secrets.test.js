'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { createStore } = require('../lib/secrets')

const fakeSafe = (ok, backend = 'gnome_libsecret') => ({
  isEncryptionAvailable: () => ok,
  getSelectedStorageBackend: () => backend,
  encryptString: (s) => Buffer.from('ENC:' + [...s].reverse().join('')),
  decryptString: (b) => [...b.toString().slice(4)].reverse().join(''),
})
const tmp = () => fs.mkdtempSync(path.join(os.tmpdir(), 'hrd-sec-'))

test('password stored encrypted, never plain text', () => {
  const dir = tmp()
  const s = createStore({ dir, safeStorage: fakeSafe(true) })
  assert.equal(s.save({ serverUrl: 'http://h:1', username: 'u', password: 'SuperSecret' }).passwordPersisted, true)
  const raw = fs.readFileSync(s.file, 'utf8')
  assert.ok(!raw.includes('SuperSecret'))
  assert.equal((fs.statSync(s.file).mode & 0o777), 0o600)
  const again = createStore({ dir, safeStorage: fakeSafe(true) }).load()
  assert.equal(again.password, 'SuperSecret')
  assert.equal(again.username, 'u')
  fs.rmSync(dir, { recursive: true })
})

test('no encryption => password not persisted (memory only)', () => {
  for (const safe of [fakeSafe(false), fakeSafe(true, 'basic_text'), null]) {
    const dir = tmp()
    const s = createStore({ dir, safeStorage: safe })
    assert.equal(s.save({ serverUrl: 'http://h:1', username: 'u', password: 'SuperSecret' }).passwordPersisted, false)
    assert.ok(!fs.readFileSync(s.file, 'utf8').includes('SuperSecret'))
    assert.equal(s.get().password, 'SuperSecret')
    assert.equal(createStore({ dir, safeStorage: safe }).load().password, '')
    fs.rmSync(dir, { recursive: true })
  }
})
