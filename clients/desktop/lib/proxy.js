'use strict'
// API proxy: the Helix server sends no CORS headers, so the renderer never talks
// to it. The app:// protocol handler forwards API requests from the main process.
// Only read-only methods and the server's known API prefixes pass; the renderer's
// Cookie header is never forwarded; credentials from the main-process store are
// injected only when the renderer did not send its own Authorization.

const API_PREFIXES = ['/v1/', '/healthz', '/readyz']
const FORWARD_REQUEST_HEADERS = ['accept', 'accept-language', 'range', 'if-none-match', 'if-modified-since', 'authorization']
const DROP_RESPONSE_HEADERS = new Set(['set-cookie', 'set-cookie2', 'www-authenticate', 'proxy-authenticate', 'connection', 'keep-alive', 'transfer-encoding', 'content-encoding', 'content-length', 'access-control-allow-origin'])

function isApiPath(pathname) {
  return API_PREFIXES.some((p) => (p.endsWith('/') ? pathname.startsWith(p) : pathname === p))
}

// Returns {ok:true, target} or {ok:false, status, reason}. `serverBase` is the
// normalized base from allowlist.normalizeServerUrl.
function buildTarget(serverBase, requestUrl, method) {
  if (!serverBase) return { ok: false, status: 503, reason: 'No Helix server is configured' }
  if (method !== 'GET' && method !== 'HEAD') return { ok: false, status: 405, reason: 'Method not allowed' }
  let u
  try { u = new URL(requestUrl) } catch { return { ok: false, status: 400, reason: 'Bad URL' } }
  // Reject raw dot segments / encoded slashes / backslashes that could escape the prefix
  if (/(^|\/)\.\.?(\/|$)/.test(u.pathname) || /%2e|%2f|%5c|\\/i.test(u.pathname)) return { ok: false, status: 400, reason: 'Bad path' }
  if (!isApiPath(u.pathname)) return { ok: false, status: 404, reason: 'Not an API path' }
  return { ok: true, target: serverBase + u.pathname + u.search }
}

function requestHeaders(inHeaders, creds) {
  const out = {}
  for (const [k, v] of inHeaders) {
    if (FORWARD_REQUEST_HEADERS.includes(k.toLowerCase())) out[k.toLowerCase()] = v
  }
  if (!out.authorization && creds && creds.username) {
    out.authorization = 'Basic ' + Buffer.from(`${creds.username}:${creds.password || ''}`).toString('base64')
  }
  return out
}

function responseHeaders(inHeaders) {
  const out = new Headers()
  for (const [k, v] of inHeaders) if (!DROP_RESPONSE_HEADERS.has(k.toLowerCase())) out.append(k, v)
  return out
}

module.exports = { isApiPath, buildTarget, requestHeaders, responseHeaders, API_PREFIXES }
