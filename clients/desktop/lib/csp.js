'use strict'
// Content-Security-Policy applied to every app:// response. Strict: only the
// bundled assets, no network from the renderer except the app origin (the main
// process proxies API calls), no frames, no plugins, no form posts.
const APP_ORIGIN = 'app://helix'

const DIRECTIVES = Object.freeze({
  'default-src': ["'none'"],
  'script-src': ["'self'"],
  'style-src': ["'self'", "'unsafe-inline'"], // Reader.tsx sets one inline style attribute
  'img-src': ["'self'", 'blob:', 'data:'],
  'font-src': ["'self'"],
  'connect-src': ["'self'"],
  'media-src': ["'self'", 'blob:'],
  'object-src': ["'none'"],
  'base-uri': ["'none'"],
  'form-action': ["'none'"],
  'frame-ancestors': ["'none'"],
  'frame-src': ["'none'"],
  'worker-src': ["'none'"],
})

function buildCsp() {
  return Object.entries(DIRECTIVES).map(([k, v]) => `${k} ${v.join(' ')}`).join('; ')
}

module.exports = { APP_ORIGIN, DIRECTIVES, buildCsp }
