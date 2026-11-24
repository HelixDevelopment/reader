'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const { buildTarget, requestHeaders, responseHeaders, isApiPath } = require('../lib/proxy')

const base = 'http://127.0.0.1:8090'

test('API paths', () => {
  for (const p of ['/v1/works', '/v1/files/x/content', '/healthz', '/readyz']) assert.equal(isApiPath(p), true, p)
  for (const p of ['/', '/index.html', '/v1', '/healthzz', '/admin', '/v2/x']) assert.equal(isApiPath(p), false, p)
})

test('buildTarget forwards path+query to the configured base only', () => {
  const r = buildTarget(base, 'app://helix/v1/works?limit=5&q=a', 'GET')
  assert.deepEqual(r, { ok: true, target: 'http://127.0.0.1:8090/v1/works?limit=5&q=a' })
  assert.equal(buildTarget(base + '/pre', 'app://helix/v1/works', 'HEAD').target, 'http://127.0.0.1:8090/pre/v1/works')
})

test('buildTarget refuses everything else', () => {
  assert.equal(buildTarget(null, 'app://helix/v1/works', 'GET').status, 503)
  for (const m of ['POST', 'PUT', 'DELETE', 'PATCH']) assert.equal(buildTarget(base, 'app://helix/v1/works', m).status, 405, m)
  assert.equal(buildTarget(base, 'app://helix/admin', 'GET').status, 404)
  assert.equal(buildTarget(base, 'app://helix/v1/../admin/x', 'GET').ok, false)
  assert.equal(buildTarget(base, 'app://helix/v1/%2e%2e/admin', 'GET').ok, false)
  assert.equal(buildTarget(base, 'app://helix/v1/a%2Fb', 'GET').ok, false)
  assert.equal(buildTarget(base, 'app://helix/v1/a\\b', 'GET').ok, false)
  // host-smuggling forms cannot change the target host: target is always base + pathname
  const r = buildTarget(base, 'app://helix//evil.example/v1/works', 'GET')
  assert.equal(r.ok, false)
})

test('request headers: no cookie, injected creds only when none sent', () => {
  const h = new Headers({ Cookie: 'a=b', Authorization: 'Basic cmVuZGVyZXI6eA==', Range: 'bytes=0-9', 'X-Evil': '1', Accept: 'application/json' })
  const out = requestHeaders(h, { username: 'u', password: 'p' })
  assert.equal(out.cookie, undefined)
  assert.equal(out['x-evil'], undefined)
  assert.equal(out.range, 'bytes=0-9')
  assert.equal(out.authorization, 'Basic cmVuZGVyZXI6eA==')
  const out2 = requestHeaders(new Headers({ Accept: '*/*' }), { username: 'reader', password: 'pw' })
  assert.equal(out2.authorization, 'Basic ' + Buffer.from('reader:pw').toString('base64'))
  assert.equal(requestHeaders(new Headers(), null).authorization, undefined)
  assert.equal(requestHeaders(new Headers(), { username: '', password: 'x' }).authorization, undefined)
})

test('response headers: strip cookies, www-authenticate, upstream CORS, hop-by-hop', () => {
  const h = responseHeaders(new Headers({ 'Set-Cookie': 'a=b', 'WWW-Authenticate': 'Basic realm=x', 'Access-Control-Allow-Origin': '*', 'Content-Type': 'application/json', 'Content-Range': 'bytes 0-9/20', ETag: '"x"' }))
  assert.equal(h.get('set-cookie'), null)
  assert.equal(h.get('www-authenticate'), null)
  assert.equal(h.get('access-control-allow-origin'), null)
  assert.equal(h.get('content-type'), 'application/json')
  assert.equal(h.get('content-range'), 'bytes 0-9/20')
  assert.equal(h.get('etag'), '"x"')
})

test('request headers: host, method override and forwarding headers are never forwarded', () => {
  const h = new Headers({
    Host: 'evil.example', 'X-HTTP-Method-Override': 'DELETE', 'X-Method-Override': 'PUT', 'X-HTTP-Method': 'PATCH',
    'X-Forwarded-For': '6.6.6.6', 'X-Forwarded-Host': 'evil.example', 'X-Forwarded-Proto': 'https', 'X-Forwarded-Port': '1', Forwarded: 'for=6.6.6.6', 'X-Real-IP': '6.6.6.6',
    Accept: 'application/json',
  })
  const out = requestHeaders(h, null)
  assert.deepEqual(Object.keys(out), ['accept'])
})
