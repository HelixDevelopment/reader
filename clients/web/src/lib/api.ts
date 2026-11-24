// Typed client for the real Helix server API (server/internal/httpapi/router.go).
// Read routes live under /v1 and use HTTP Basic auth when the server has OPDS
// credentials configured; with none configured they are open.

export interface Identifier { scheme: string; value: string }
export interface File {
  id: string
  source_id?: string
  format: string
  mime_type: string
  size: number
  sha256: string
  protected: boolean
  drm_scheme?: string
}
export interface Edition {
  id: string
  title: string
  subtitle?: string
  authors: string[]
  publisher?: string
  published?: string
  language?: string
  description?: string
  identifiers: Identifier[]
  files: File[]
  cover_href?: string
  identification_state: string
  identification_confidence: number
}
export interface Work {
  id: string
  canonical_title: string
  series?: string
  series_index?: number
  subjects?: string[]
  editions: Edition[]
  created_at: string
  updated_at: string
}
export interface Page {
  works: Work[]
  total: number
  offset: number
  limit: number
  has_next: boolean
}
export interface ListParams { q?: string; author?: string; offset?: number; limit?: number }
export interface FileContent { blob: Blob; mime: string }

export interface Api {
  listWorks(params: ListParams): Promise<Page>
  getWork(id: string): Promise<Work>
  cover(editionId: string): Promise<Blob>
  fileContent(fileId: string): Promise<FileContent>
}

export interface ClientOptions {
  baseUrl: string
  username?: string
  password?: string
  fetchImpl?: typeof fetch
}

export class ApiError extends Error {
  status: number
  requestId?: string
  constructor(status: number, message: string, requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.requestId = requestId
  }
}

// Go marshals nil slices as JSON null (observed from the live server for authors
// of files without embedded metadata); the UI types promise arrays.
/* eslint-disable @typescript-eslint/no-explicit-any */
export function normalizeWork(w: any): Work {
  return { ...w, subjects: w.subjects ?? [], editions: (w.editions ?? []).map((e: any) => ({ ...e, authors: e.authors ?? [], identifiers: e.identifiers ?? [], files: e.files ?? [] })) }
}
/* eslint-enable @typescript-eslint/no-explicit-any */

/** RFC 7617 charset=UTF-8: btoa() alone throws on anything outside Latin-1. */
function utf8Base64(text: string): string {
  let bin = ''
  for (const b of new TextEncoder().encode(text)) bin += String.fromCharCode(b)
  return btoa(bin)
}

export function createClient(opts: ClientOptions): Api {
  const base = opts.baseUrl.replace(/\/+$/, '')
  const doFetch = opts.fetchImpl ?? ((...a: Parameters<typeof fetch>) => fetch(...a))
  const headers: Record<string, string> = {}
  if (opts.username) {
    if (opts.username.includes(':')) throw new Error('The username cannot contain a colon (HTTP Basic authentication cannot represent it).')
    headers.Authorization = 'Basic ' + utf8Base64(`${opts.username}:${opts.password ?? ''}`)
  }

  async function request(path: string): Promise<Response> {
    let res: Response
    try {
      // credentials 'omit': the Helix server answers 401 with WWW-Authenticate, which would
      // otherwise make the browser raise its own modal prompt and stall the request.
      res = await doFetch(base + path, { headers, credentials: 'omit' })
    } catch {
      throw new ApiError(0, `Cannot reach the server${base ? ' at ' + base : ''}`)
    }
    if (res.ok) return res
    if (res.status === 401) throw new ApiError(401, 'The server rejected the credentials (HTTP 401). Check the username and password in Settings.')
    let message = `Server returned HTTP ${res.status}`
    let requestId: string | undefined
    try {
      const body = await res.json()
      if (typeof body?.error === 'string') message = body.error
      if (typeof body?.request_id === 'string') requestId = body.request_id
    } catch { /* body was not JSON; keep the status message */ }
    throw new ApiError(res.status, message, requestId)
  }

  const blobOf = async (path: string): Promise<FileContent> => {
    const res = await request(path)
    return { blob: await res.blob(), mime: (res.headers.get('Content-Type') ?? '').split(';')[0].trim() }
  }

  return {
    async listWorks(p) {
      const qs = new URLSearchParams()
      if (p.q) qs.set('q', p.q)
      if (p.author) qs.set('author', p.author)
      if (p.offset) qs.set('offset', String(p.offset))
      if (p.limit) qs.set('limit', String(p.limit))
      const s = qs.toString()
      const page = await (await request('/v1/works' + (s ? '?' + s : ''))).json()
      return { ...page, works: (page.works ?? []).map(normalizeWork) }
    },
    async getWork(id) {
      return normalizeWork(await (await request('/v1/works/' + encodeURIComponent(id))).json())
    },
    async cover(editionId) {
      return (await blobOf('/v1/covers/' + encodeURIComponent(editionId))).blob
    },
    fileContent(fileId) {
      return blobOf('/v1/files/' + encodeURIComponent(fileId) + '/content')
    },
  }
}
