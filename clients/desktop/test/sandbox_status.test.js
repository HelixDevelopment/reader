'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')

test('sandbox state is PASS only when observed ON; OFF is a WARN, or a FAIL when it was expected', async () => {
  const { sandboxStatus } = await import('../e2e/sandbox_status.mjs')
  assert.equal(sandboxStatus(true, false), 'PASS')
  assert.equal(sandboxStatus(true, true), 'PASS')
  assert.equal(sandboxStatus(false, false), 'WARN')
  assert.equal(sandboxStatus(false, true), 'FAIL')
})

test('live.mjs never lets an OFF sandbox count as PASS', () => {
  const src = require('node:fs').readFileSync(require('node:path').join(__dirname, '../e2e/live.mjs'), 'utf8')
  assert.match(src, /sandboxStatus\(sandboxed/)
  assert.doesNotMatch(src, /check\('chromium-sandbox-state'/)
})
