'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const { chromiumSwitches, apiFetchFor } = require('../lib/transport')

test('QUIC is always enabled and nothing else is by default', () => {
  assert.deepEqual(chromiumSwitches({}), [['enable-quic']])
})

test('private-CA origins force QUIC; pins are sha256 SPKI base64', () => {
  const spki = 'gAWIqN425y+sEoMfUUcopFUECCpXRF+KyYysOcAOEis='
  const sw = chromiumSwitches({ HELIX_DESKTOP_QUIC_ORIGINS: 'helix.lan:8443, 192.168.1.5:443', HELIX_DESKTOP_PIN_SPKI_SHA256: spki })
  assert.deepEqual(sw, [['enable-quic'], ['origin-to-force-quic-on', 'helix.lan:8443,192.168.1.5:443'], ['ignore-certificate-errors-spki-list', spki]])
})

test('malformed origins or pins are rejected, never passed through to Chromium', () => {
  for (const bad of ['helix.lan', 'a:b', 'x y:1', 'http://h:1', 'h:0', 'h:70000', '*:443', 'h:1,evil flag:2', ',,'])
    assert.throws(() => chromiumSwitches({ HELIX_DESKTOP_QUIC_ORIGINS: bad }), /HELIX_DESKTOP_QUIC_ORIGINS/, bad)
  for (const bad of ['notbase64!', 'abc', 'AAAA,BBBB', 'gAWIqN425y+sEoMfUUcopFUECCpXRF+KyYysOcAOEis'])
    assert.throws(() => chromiumSwitches({ HELIX_DESKTOP_PIN_SPKI_SHA256: bad }), /HELIX_DESKTOP_PIN_SPKI_SHA256/, bad)
})

test('apiFetchFor routes through the Chromium session fetch (not Node undici)', async () => {
  const calls = []
  const ses = { fetch: async (u, o) => { calls.push([u, o]); return new Response('ok') } }
  const f = apiFetchFor(ses)
  const r = await f('https://h:1/v1/x', { method: 'GET' })
  assert.equal(await r.text(), 'ok'); assert.equal(calls.length, 1)
  assert.throws(() => apiFetchFor(undefined), /session/)
})
