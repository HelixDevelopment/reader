'use strict'
// Network transport of the desktop shell (HRD cronet streams). Electron IS Chromium: the main-process API
// forwarder must use Chromium's network stack (session.fetch), not Node's global fetch (undici: HTTP/1.1,
// no QUIC), so the shell speaks HTTP/3 + Brotli to the Helix server exactly as Chrome/Cronet would.
//
// Chromium only uses QUIC for an origin whose certificate chains to a PUBLICLY trusted root (Alt-Svc discovery),
// or for origins it is told to force. Self-hosted Helix servers mostly use a private CA, so two OPT-IN
// environment settings exist (both validated; nothing else reaches Chromium's command line):
//   HELIX_DESKTOP_QUIC_ORIGINS      host:port[,host:port...]   -> --origin-to-force-quic-on
//   HELIX_DESKTOP_PIN_SPKI_SHA256   base64 sha256 SPKI[,...]   -> --ignore-certificate-errors-spki-list
// The second one ACCEPTS a certificate whose public key hash matches even if the chain is untrusted: use it only for
// the private server's own key.

const ORIGIN_RE = /^(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*:([0-9]{1,5})$/
const SPKI_RE = /^[A-Za-z0-9+/]{43}=$/

function list(raw, name, re, extra) {
  const parts = raw.split(',').map((p) => p.trim())
  for (const p of parts) {
    const m = re.exec(p)
    if (!m || (extra && !extra(m))) throw new Error(`${name}: invalid entry ${JSON.stringify(p)}`)
  }
  return parts
}

// Returns [[switch] | [switch, value], ...] for app.commandLine.appendSwitch.
function chromiumSwitches(env) {
  const out = [['enable-quic']]
  const origins = env.HELIX_DESKTOP_QUIC_ORIGINS
  if (origins) out.push(['origin-to-force-quic-on', list(origins, 'HELIX_DESKTOP_QUIC_ORIGINS', ORIGIN_RE, (m) => +m[1] >= 1 && +m[1] <= 65535).join(',')])
  const pin = env.HELIX_DESKTOP_PIN_SPKI_SHA256
  if (pin) out.push(['ignore-certificate-errors-spki-list', list(pin, 'HELIX_DESKTOP_PIN_SPKI_SHA256', SPKI_RE).join(',')])
  return out
}

function apiFetchFor(ses) {
  if (!ses || typeof ses.fetch !== 'function') throw new Error('apiFetchFor needs an Electron session with fetch()')
  return (url, init) => ses.fetch(url, init)
}

module.exports = { chromiumSwitches, apiFetchFor }
