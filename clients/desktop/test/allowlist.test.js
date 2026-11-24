'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const { isAllowedNavigation, isAllowedRequest, normalizeServerUrl } = require('../lib/allowlist')

test('navigation: only the app origin', () => {
  assert.equal(isAllowedNavigation('app://helix/index.html'), true)
  assert.equal(isAllowedNavigation('app://helix/#/settings'), true)
  for (const bad of ['https://evil.example/', 'http://127.0.0.1:8080/', 'file:///etc/passwd', 'javascript:alert(1)', 'data:text/html,x', 'app://evil/index.html', 'app://helix.evil/', 'not a url', '', 'blob:app://helix/x']) {
    assert.equal(isAllowedNavigation(bad), false, bad)
  }
})

test('requests: app assets, blob and data only; devtools only when enabled', () => {
  assert.equal(isAllowedRequest('app://helix/assets/x.js'), true)
  assert.equal(isAllowedRequest('blob:app://helix/1234'), true)
  assert.equal(isAllowedRequest('data:image/png;base64,AA=='), true)
  assert.equal(isAllowedRequest('https://example.com/x.png'), false)
  assert.equal(isAllowedRequest('http://127.0.0.1:1/'), false)
  assert.equal(isAllowedRequest('ws://127.0.0.1:1/'), false)
  assert.equal(isAllowedRequest('devtools://devtools/x'), false)
  assert.equal(isAllowedRequest('devtools://devtools/x', { devtools: true }), true)
  assert.equal(isAllowedRequest('app://other/x'), false)
})

test('server url normalisation', () => {
  assert.equal(normalizeServerUrl('http://127.0.0.1:8090/'), 'http://127.0.0.1:8090')
  assert.equal(normalizeServerUrl(' https://h.example/base// '), 'https://h.example/base')
  for (const bad of ['', 'ftp://h/', 'file:///x', 'http://u:p@h/', 'http://h/?q=1', 'http://h/#f', 'javascript:1', null, undefined, 5]) {
    assert.equal(normalizeServerUrl(bad), null, String(bad))
  }
})
