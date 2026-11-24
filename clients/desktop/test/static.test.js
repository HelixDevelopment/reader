'use strict'
const test = require('node:test')
const assert = require('node:assert/strict')
const path = require('node:path')
const { resolveStatic, mimeFor } = require('../lib/static')
const root = path.resolve('/srv/app/dist')

test('static resolves inside root and refuses traversal', () => {
  assert.equal(resolveStatic(root, '/'), path.join(root, 'index.html'))
  assert.equal(resolveStatic(root, '/assets/a.js'), path.join(root, 'assets/a.js'))
  for (const bad of ['/../secret', '/..%2fsecret', '/%2e%2e/secret', '/a/../../secret', '/a\\..\\b', '/x%00.js', '/%E0%A4%A']) assert.equal(resolveStatic(root, bad), null, bad)
})

test('mime', () => {
  assert.match(mimeFor('a.js'), /javascript/)
  assert.match(mimeFor('a.css'), /css/)
  assert.equal(mimeFor('a.zzz'), 'application/octet-stream')
})
