import { loadSettings, saveSettings, validateBaseUrl } from './settings'

describe('settings', () => {
  beforeEach(() => { localStorage.clear(); sessionStorage.clear() })

  it('round-trips, keeping the password out of localStorage', () => {
    saveSettings({ baseUrl: 'http://h:1', username: 'u', password: 'secret' })
    expect(loadSettings()).toEqual({ baseUrl: 'http://h:1', username: 'u', password: 'secret' })
    expect(localStorage.getItem('helix.settings.v1')).not.toContain('secret')
  })

  it('validates base URLs', () => {
    expect(validateBaseUrl('')).toBeNull()
    expect(validateBaseUrl('http://x:8080')).toBeNull()
    expect(validateBaseUrl('ftp://x')).toMatch(/http/)
    expect(validateBaseUrl('not a url')).toMatch(/full address/)
  })
})
