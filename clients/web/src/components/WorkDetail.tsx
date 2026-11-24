import { useEffect, useState } from 'react'
import type { Api, File, Work } from '../lib/api'
import { ApiError } from '../lib/api'
import { formatSize, primaryEdition } from '../lib/display'
import { Cover } from './Cover'
import { Reader } from './Reader'

export function WorkDetail({ api, workId, onBack }: { api: Api; workId: string; onBack: () => void }) {
  const [state, setState] = useState<{ kind: 'loading' } | { kind: 'error'; message: string } | { kind: 'ok'; work: Work }>({ kind: 'loading' })
  const [reading, setReading] = useState<File | null>(null)

  useEffect(() => {
    let live = true
    api.getWork(workId).then(
      (work) => { if (live) setState({ kind: 'ok', work }) },
      (e: unknown) => { if (live) setState({ kind: 'error', message: e instanceof ApiError ? e.message : 'Unexpected error' }) },
    )
    return () => { live = false }
  }, [api, workId])

  return (
    <section>
      <button type="button" className="back" onClick={reading ? () => setReading(null) : onBack}>← {reading ? 'Back to details' : 'Back to library'}</button>
      {state.kind === 'loading' && <p role="status" className="note">Loading…</p>}
      {state.kind === 'error' && <div role="alert" className="panel error"><p>{state.message}</p></div>}
      {state.kind === 'ok' && (() => {
        const w = state.work
        const e = primaryEdition(w)
        if (reading) return <><h1>{w.canonical_title}</h1><Reader key={reading.id} api={api} file={reading} title={w.canonical_title} /></>
        return (
          <div className="detail">
            <Cover api={api} editionId={e?.id} hasCover={!!e?.cover_href} title={w.canonical_title} />
            <div>
              <h1>{w.canonical_title}</h1>
              {e && <p className="muted">{e.authors.join(', ') || 'Unknown author'}{e.published ? ` · ${e.published}` : ''}{e.language ? ` · ${e.language}` : ''}</p>}
              {w.series && <p className="muted">Series: {w.series}{w.series_index ? ` #${w.series_index}` : ''}</p>}
              {e?.description && <p>{e.description}</p>}
              <h2>Files</h2>
              {!e || e.files.length === 0 ? <p className="muted">No files are attached to this edition.</p> : (
                <ul className="files">
                  {e.files.map((f) => (
                    <li key={f.id}>
                      <span>{f.format.toUpperCase()} · {formatSize(f.size)}{f.protected ? ' · protected' : ''}</span>
                      <button type="button" onClick={() => setReading(f)}>Read {f.format.toUpperCase()}</button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        )
      })()}
    </section>
  )
}
