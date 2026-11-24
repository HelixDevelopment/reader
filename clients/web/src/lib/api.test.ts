import { ApiError, createClient } from './api'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

describe('api client', () => {
  it('requests /v1/works with query parameters and Basic auth', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => json({ works: [], total: 0, offset: 0, limit: 20, has_next: false }))
    const api = createClient({ baseUrl: 'http://srv:8080/', username: 'u', password: 'p', fetchImpl })
    const page = await api.listWorks({ q: 'glass meridian', offset: 20, limit: 10 })
    expect(page.total).toBe(0)
    const [input, init] = fetchImpl.mock.calls[0]
    const url = String(input)
    expect(url).toBe('http://srv:8080/v1/works?q=glass+meridian&offset=20&limit=10')
    expect((init!.headers as Record<string, string>).Authorization).toBe('Basic ' + btoa('u:p'))
    expect(init!.credentials).toBe('omit')
  })

  it('uses same-origin paths and no auth header when unconfigured', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => json({ id: 'w1', canonical_title: 'T', editions: [] }))
    const api = createClient({ baseUrl: '', fetchImpl })
    await api.getWork('a/b c')
    const [input, init] = fetchImpl.mock.calls[0]
    const url = String(input)
    expect(url).toBe('/v1/works/a%2Fb%20c')
    expect((init!.headers as Record<string, string>).Authorization).toBeUndefined()
  })

  it('turns server error JSON into ApiError with status and request id', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => json({ error: 'work not found', request_id: 'r-1' }, 404))
    const api = createClient({ baseUrl: '', fetchImpl })
    await expect(api.getWork('x')).rejects.toMatchObject({ status: 404, message: 'work not found', requestId: 'r-1' })
  })

  it('reports 401 as an authentication error even with an empty body', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => new Response(null, { status: 401 }))
    const api = createClient({ baseUrl: '', fetchImpl })
    const err = await api.listWorks({}).catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(401)
    expect(err.message).toMatch(/credentials/i)
  })

  it('wraps network failures as ApiError status 0', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => { throw new TypeError('Failed to fetch') })
    const api = createClient({ baseUrl: 'http://down', fetchImpl })
    await expect(api.listWorks({})).rejects.toMatchObject({ status: 0 })
  })

  it('fetches cover and file bytes as blobs with their MIME type', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => new Response('hello', { status: 200, headers: { 'Content-Type': 'text/plain; charset=utf-8' } }))
    const api = createClient({ baseUrl: '', fetchImpl })
    const file = await api.fileContent('f1')
    expect(file.mime).toBe('text/plain')
    expect(await file.blob.text()).toBe('hello')
    expect(String(fetchImpl.mock.calls[0][0])).toBe('/v1/files/f1/content')
    await api.cover('e1')
    expect(String(fetchImpl.mock.calls[1][0])).toBe('/v1/covers/e1')
  })

  it('normalizes JSON null arrays (Go nil slices) to empty arrays', async () => {
    const wire = { id: 'w', canonical_title: 'T', subjects: null, editions: [{ id: 'e', title: 'T', authors: null, identifiers: null, files: null }] }
    const fetchImpl = vi.fn<typeof fetch>(async () => json(wire))
    const api = createClient({ baseUrl: '', fetchImpl })
    const w = await api.getWork('w')
    expect(w.editions[0].authors).toEqual([])
    expect(w.editions[0].files).toEqual([])
    expect(w.editions[0].identifiers).toEqual([])
    const page = await createClient({ baseUrl: '', fetchImpl: vi.fn<typeof fetch>(async () => json({ works: null, total: 0, offset: 0, limit: 20, has_next: false })) }).listWorks({})
    expect(page.works).toEqual([])
  })

  it('encodes non-Latin-1 credentials as UTF-8 base64 instead of throwing', async () => {
    const fetchImpl = vi.fn<typeof fetch>(async () => json({ works: [], total: 0, offset: 0, limit: 20, has_next: false }))
    const api = createClient({ baseUrl: '', username: 'Милош', password: 'лозинка', fetchImpl })
    await api.listWorks({})
    const h = fetchImpl.mock.calls[0][1]!.headers as Record<string, string>
    expect(h.Authorization).toBe('Basic ' + '0JzQuNC70L7RiDrQu9C+0LfQuNC90LrQsA==' /* = Buffer.from('Милош:лозинка','utf8').toString('base64') */)
  })

  it('rejects a username containing a colon (not representable in Basic auth)', () => {
    expect(() => createClient({ baseUrl: '', username: 'a:b', password: 'p' })).toThrow(/colon/i)
  })
})
