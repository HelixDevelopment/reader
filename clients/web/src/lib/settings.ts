// Connection settings. The base URL is a convenience kept in localStorage; the
// password is session-only. Storage can throw (private windows), so every access
// is guarded and the app still works with in-memory defaults.
export interface Settings { baseUrl: string; username: string; password: string }
const KEY = 'helix.settings.v1'

export function loadSettings(): Settings {
  const out: Settings = { baseUrl: '', username: '', password: '' }
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) ?? '{}')
    if (typeof saved.baseUrl === 'string') out.baseUrl = saved.baseUrl
    if (typeof saved.username === 'string') out.username = saved.username
  } catch { /* ignore */ }
  try { out.password = sessionStorage.getItem(KEY + '.pw') ?? '' } catch { /* ignore */ }
  return out
}

export function saveSettings(s: Settings): void {
  try { localStorage.setItem(KEY, JSON.stringify({ baseUrl: s.baseUrl, username: s.username })) } catch { /* ignore */ }
  try { sessionStorage.setItem(KEY + '.pw', s.password) } catch { /* ignore */ }
}

/** Accepts "", or an absolute http(s) URL. Returns an error message or null. */
export function validateBaseUrl(v: string): string | null {
  if (v.trim() === '') return null
  try {
    const u = new URL(v.trim())
    return u.protocol === 'http:' || u.protocol === 'https:' ? null : 'Use an http:// or https:// address.'
  } catch {
    return 'Enter a full address such as http://192.168.1.10:8080, or leave empty for the same origin.'
  }
}
