'use strict'
const path = require('node:path')

const MIME = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.mjs': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.json': 'application/json', '.svg': 'image/svg+xml', '.png': 'image/png', '.ico': 'image/x-icon', '.webp': 'image/webp', '.woff2': 'font/woff2', '.woff': 'font/woff', '.map': 'application/json', '.txt': 'text/plain; charset=utf-8', '.webmanifest': 'application/manifest+json' }

// Maps a request pathname to a file inside `root`, or null when it would escape.
function resolveStatic(root, pathname) {
  let p
  try { p = decodeURIComponent(pathname) } catch { return null }
  if (p.includes('\0') || p.includes('\\')) return null
  if (p.split('/').includes('..')) return null
  if (p === '' || p === '/') p = '/index.html'
  const abs = path.resolve(root, '.' + path.posix.normalize('/' + p))
  const rootAbs = path.resolve(root)
  if (abs !== rootAbs && !abs.startsWith(rootAbs + path.sep)) return null
  return abs
}

function mimeFor(file) { return MIME[path.extname(file).toLowerCase()] || 'application/octet-stream' }

module.exports = { resolveStatic, mimeFor }
