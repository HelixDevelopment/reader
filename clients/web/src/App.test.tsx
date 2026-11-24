import { render, screen } from '@testing-library/react'
import App from './App'

const json = (b: unknown) => new Response(JSON.stringify(b), { status: 200, headers: { 'Content-Type': 'application/json' } })

describe('App', () => {
  beforeEach(() => {
    localStorage.clear(); sessionStorage.clear(); window.location.hash = ''
    vi.stubGlobal('fetch', vi.fn(async () => json({ works: [], total: 0, offset: 0, limit: 20, has_next: false })))
  })
  afterEach(() => vi.unstubAllGlobals())

  it('renders with a stored Cyrillic username and sends UTF-8 Basic auth', async () => {
    localStorage.setItem('helix.settings.v1', JSON.stringify({ baseUrl: '', username: 'Милош' }))
    sessionStorage.setItem('helix.settings.v1.pw', 'лозинка')
    render(<App />)
    expect(screen.getByRole('link', { name: 'Helix Reader' })).toBeInTheDocument()
    await screen.findByText(/no works in this library/i)
    const init = (fetch as unknown as ReturnType<typeof vi.fn>).mock.calls[0][1] as { headers: Record<string, string> }
    expect(init.headers.Authorization).toBe('Basic ' + '0JzQuNC70L7RiDrQu9C+0LfQuNC90LrQsA==' /* = Buffer.from('Милош:лозинка','utf8').toString('base64') */)
  })

  it('shows Settings with an error instead of a blank app when the client cannot be built', () => {
    localStorage.setItem('helix.settings.v1', JSON.stringify({ baseUrl: '', username: 'a:b' }))
    render(<App />)
    expect(screen.getByRole('link', { name: 'Helix Reader' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /server settings/i })).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent(/colon/i)
  })

  it('skip link moves focus to main without changing the route', async () => {
    render(<App />)
    await screen.findByText(/no works in this library/i)
    const link = screen.getByRole('link', { name: /skip to content/i })
    link.click()
    expect(document.activeElement).toBe(document.getElementById('main'))
    expect(window.location.hash).not.toBe('#main')
  })
})
