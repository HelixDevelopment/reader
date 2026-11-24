import { useCallback, useEffect, useState } from 'react'
import type { Api, Page } from '../lib/api'
import { ApiError } from '../lib/api'
import { authorsOf, primaryEdition } from '../lib/display'
import { Cover } from './Cover'

const PAGE = 24

export function Library({ api, onOpen }: { api: Api; onOpen: (workId: string) => void }) {
  const [text, setText] = useState('')
  const [q, setQ] = useState('')
  const [offset, setOffset] = useState(0)
  const [attempt, setAttempt] = useState(0)
  // A result is only shown while its key matches the current request, so a new
  // search or page reads as loading without resetting state inside an effect.
  const key = `${q}|${offset}|${attempt}`
  const [result, setResult] = useState<{ key: string; error?: string; page?: Page }>({ key: '' })
  const state: { kind: 'loading' } | { kind: 'error'; message: string } | { kind: 'ok'; page: Page } =
    result.key !== key ? { kind: 'loading' } : result.error !== undefined ? { kind: 'error', message: result.error } : { kind: 'ok', page: result.page! }

  useEffect(() => {
    const t = setTimeout(() => { setQ(text); setOffset(0) }, 250)
    return () => clearTimeout(t)
  }, [text])

  const retry = useCallback(() => setAttempt((n) => n + 1), [])

  useEffect(() => {
    let live = true
    api.listWorks({ q, offset, limit: PAGE }).then(
      (page) => { if (live) setResult({ key, page }) },
      (e: unknown) => { if (live) setResult({ key, error: e instanceof ApiError ? e.message : 'Unexpected error' }) },
    )
    return () => { live = false }
  }, [api, q, offset, key])

  return (
    <section aria-labelledby="library-h">
      <div className="toolbar">
        <h1 id="library-h">Library</h1>
        <label className="search">
          <span className="sr-only">Search library</span>
          <input type="search" role="searchbox" aria-label="Search library" placeholder="Search titles" value={text} onChange={(e) => setText(e.target.value)} />
        </label>
      </div>
      {state.kind === 'loading' && <p role="status" className="note">Loading library…</p>}
      {state.kind === 'error' && (
        <div role="alert" className="panel error">
          <p>{state.message}</p>
          <button type="button" onClick={retry}>Retry</button>
        </div>
      )}
      {state.kind === 'ok' && state.page.works.length === 0 && (
        <p className="panel">{q ? `No works match “${q}”.` : 'No works in this library yet. Register a source and run a scan on the server.'}</p>
      )}
      {state.kind === 'ok' && state.page.works.length > 0 && (
        <>
          <ul className="grid" aria-label="Works">
            {state.page.works.map((w) => (
              <li key={w.id} className="card">
                <button type="button" className="card-btn" aria-label={`Open ${w.canonical_title}`} onClick={() => onOpen(w.id)}>
                  <Cover api={api} editionId={primaryEdition(w)?.id} hasCover={!!primaryEdition(w)?.cover_href} title={w.canonical_title} />
                  <span className="card-text">
                    <h2>{w.canonical_title}</h2>
                    <span className="muted">{authorsOf(w)}</span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
          <nav className="pager" aria-label="Pagination">
            <button type="button" disabled={offset === 0} onClick={() => setOffset(Math.max(0, offset - PAGE))}>Previous</button>
            <span className="muted">{state.page.offset + 1}–{state.page.offset + state.page.works.length} of {state.page.total}</span>
            <button type="button" disabled={!state.page.has_next} onClick={() => setOffset(offset + PAGE)}>Next</button>
          </nav>
        </>
      )}
    </section>
  )
}
