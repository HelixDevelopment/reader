'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const { buildCsp, DIRECTIVES } = require('../lib/csp')

test('CSP is strict', () => {
  const csp = buildCsp()
  assert.match(csp, /default-src 'none'/)
  assert.match(csp, /script-src 'self'(;|$)/)
  assert.doesNotMatch(csp, /unsafe-eval/)
  assert.doesNotMatch(csp.match(/script-src[^;]*/)[0], /unsafe-inline/)
  assert.match(csp, /connect-src 'self'(;|$)/)
  assert.match(csp, /object-src 'none'/)
  assert.match(csp, /frame-ancestors 'none'/)
  assert.match(csp, /base-uri 'none'/)
  assert.match(csp, /form-action 'none'/)
  for (const [k, v] of Object.entries(DIRECTIVES)) assert.ok(!v.includes('*') && !v.some((x) => /^https?:/.test(x)), k)
})
