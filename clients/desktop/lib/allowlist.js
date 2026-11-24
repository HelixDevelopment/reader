'use strict'
const { APP_ORIGIN } = require('./csp')

// Navigation / request policy. The renderer may only ever be at the app origin.
// Anything else (http(s), file:, javascript:, data: documents, other app hosts)
// is refused. The configured Helix server is reachable only through the
// main-process proxy (see proxy.js), never by renderer navigation.
function isAllowedNavigation(rawUrl) {
  let u
  try { u = new URL(rawUrl) } catch { return false }
  return u.protocol === 'app:' && u.host === 'helix'
}

// Requests the renderer's session may issue: app:// assets/API, plus blob:/data:
// (in-memory images and downloads created by the page itself). devtools:// only
// when explicitly enabled for development.
function isAllowedRequest(rawUrl, { devtools = false } = {}) {
  let u
  try { u = new URL(rawUrl) } catch { return false }
  if (u.protocol === 'app:') return u.host === 'helix'
  if (u.protocol === 'blob:' || u.protocol === 'data:') return true
  if (devtools && u.protocol === 'devtools:') return true
  return false
}

// The server base URL must be an absolute http(s) URL with a host, no
// credentials embedded, no query/fragment. Returns a normalized origin+path
// prefix without trailing slash, or null.
function normalizeServerUrl(raw) {
  if (typeof raw !== 'string') return null
  let u
  try { u = new URL(raw.trim()) } catch { return null }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') return null
  if (!u.hostname || u.username || u.password || u.search || u.hash) return null
  return u.origin + u.pathname.replace(/\/+$/, '')
}

module.exports = { isAllowedNavigation, isAllowedRequest, normalizeServerUrl }
