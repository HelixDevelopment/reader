'use strict'
// The app:// request handler, free of Electron imports so it can be unit- and
// live-tested under plain Node. main.js wires it to protocol.handle('app', ...).
const fs = require('node:fs')
const { buildCsp } = require('./csp')
const { normalizeServerUrl } = require('./allowlist')
const { buildTarget, requestHeaders, responseHeaders } = require('./proxy')
const { resolveStatic, mimeFor } = require('./static')

function createHandler({ getConn, webRoot, shellRoot, fetchImpl = fetch }) {
  const CSP = buildCsp()
  const text = (status, body) => new Response(body, { status, headers: { 'content-type': 'text/plain; charset=utf-8', 'content-security-policy': CSP } })
  const json = (status, obj) => new Response(JSON.stringify(obj), { status, headers: { 'content-type': 'application/json' } })

  function serveFile(file) {
    if (!file) return text(400, 'bad path')
    try {
      if (!fs.statSync(file).isFile()) return text(404, 'not found')
      return new Response(fs.readFileSync(file), { status: 200, headers: { 'content-type': mimeFor(file), 'content-security-policy': CSP, 'x-content-type-options': 'nosniff' } })
    } catch { return text(404, 'not found') }
  }

  return async function handle(req) {
    const url = new URL(req.url)
    if (url.host !== 'helix') return text(404, 'not found')
    if (url.pathname.startsWith('/__shell/')) return serveFile(resolveStatic(shellRoot, url.pathname.slice('/__shell'.length)))
    const conn = getConn()
    const t = buildTarget(normalizeServerUrl(conn.serverUrl), req.url, req.method)
    if (t.ok) {
      try {
        const up = await fetchImpl(t.target, { method: req.method, headers: requestHeaders(req.headers, conn), redirect: 'manual' })
        return new Response(up.body, { status: up.status, headers: responseHeaders(up.headers) })
      } catch { return json(502, { error: 'Cannot reach the Helix server' }) }
    }
    if (t.status === 503 && /^\/(v1\/|healthz|readyz)/.test(url.pathname)) return json(503, { error: t.reason })
    if (req.method !== 'GET' && req.method !== 'HEAD') return text(405, 'method not allowed')
    return serveFile(resolveStatic(webRoot, url.pathname))
  }
}

module.exports = { createHandler }
