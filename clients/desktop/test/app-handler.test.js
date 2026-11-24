'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { createHandler } = require('../lib/app-handler')

function fixture(conn, fetchImpl) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'hrd-h-'))
  fs.mkdirSync(path.join(root, 'web/assets'), { recursive: true }); fs.mkdirSync(path.join(root, 'shell'))
  fs.writeFileSync(path.join(root, 'web/index.html'), '<html>idx</html>')
  fs.writeFileSync(path.join(root, 'web/assets/a.js'), 'x')
  fs.writeFileSync(path.join(root, 'secret.txt'), 'SECRET')
  fs.writeFileSync(path.join(root, 'shell/connection.html'), 'conn')
  return { root, h: createHandler({ getConn: () => conn, webRoot: path.join(root, 'web'), shellRoot: path.join(root, 'shell'), fetchImpl }) }
}
const req = (u, init) => new Request(u, init)

test('serves index with CSP, refuses traversal and other hosts', async () => {
  const { h } = fixture({ serverUrl: '' }, async () => { throw new Error('no fetch expected') })
  const r = await h(req('app://helix/index.html'))
  assert.equal(r.status, 200); assert.equal(await r.text(), '<html>idx</html>')
  assert.match(r.headers.get('content-security-policy'), /default-src 'none'/)
  assert.equal((await h(req('app://helix/'))).status, 200)
  assert.equal((await h(req('app://helix/..%2fsecret.txt'))).status, 400)
  assert.equal((await h(req('app://helix/__shell/..%2f..%2fsecret.txt'))).status, 400)
  assert.equal((await h(req('app://evil/index.html'))).status, 404)
  assert.equal((await h(req('app://helix/missing.js'))).status, 404)
  assert.equal(await (await h(req('app://helix/__shell/connection.html'))).text(), 'conn')
})

test('API: forwarded to the configured base with injected creds, cookies stripped; 503 when unconfigured', async () => {
  let seen
  const fetchImpl = async (url, init) => { seen = { url, init }; return new Response('{"works":[]}', { status: 200, headers: { 'content-type': 'application/json', 'set-cookie': 'a=b', 'www-authenticate': 'Basic' } }) }
  const { h } = fixture({ serverUrl: 'http://127.0.0.1:9', username: 'reader', password: 'pw' }, fetchImpl)
  const r = await h(req('app://helix/v1/works?limit=2', { headers: { cookie: 'x=y' } }))
  assert.equal(r.status, 200); assert.equal(r.headers.get('set-cookie'), null); assert.equal(r.headers.get('www-authenticate'), null)
  assert.equal(seen.url, 'http://127.0.0.1:9/v1/works?limit=2')
  assert.equal(seen.init.redirect, 'manual')
  assert.equal(seen.init.headers.cookie, undefined)
  assert.equal(seen.init.headers.authorization, 'Basic ' + Buffer.from('reader:pw').toString('base64'))
  const un = fixture({ serverUrl: '' }, fetchImpl).h
  assert.equal((await un(req('app://helix/v1/works'))).status, 503)
  assert.equal((await h(req('app://helix/v1/works', { method: 'POST', body: 'x' }))).status, 405)
  const down = fixture({ serverUrl: 'http://127.0.0.1:9' }, async () => { throw new Error('ECONNREFUSED') }).h
  assert.equal((await down(req('app://helix/v1/works'))).status, 502)
})
