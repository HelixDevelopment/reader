import { useEffect, useState } from 'react'
import type { Api, File } from '../lib/api'
import { ApiError } from '../lib/api'

type View =
  | { kind: 'loading' }
  | { kind: 'error'; message: string }
  | { kind: 'text'; text: string }
  | { kind: 'unsupported'; url: string; name: string }

const isText = (f: File, mime: string) => mime.startsWith('text/') || f.format === 'txt'

/**
 * Minimal reader. The server exposes only the original file bytes
 * (GET /v1/files/:id/content); it extracts no EPUB/PDF text. Plain text is rendered
 * here; every other format is offered as a download and the limitation is stated.
 */
export function Reader({ api, file, title }: { api: Api; file: File; title: string }) {
  const [view, setView] = useState<View>({ kind: 'loading' })
  const [size, setSize] = useState(18)

  useEffect(() => {
    if (file.protected) return
    let live = true
    let made: string | null = null
    api.fileContent(file.id).then(
      async ({ blob, mime }) => {
        if (!live) return
        if (isText(file, mime)) setView({ kind: 'text', text: await blob.text() })
        else { made = URL.createObjectURL(blob); setView({ kind: 'unsupported', url: made, name: `${title}.${file.format}` }) }
      },
      (e: unknown) => { if (live) setView({ kind: 'error', message: e instanceof ApiError ? e.message : 'Unexpected error' }) },
    )
    return () => { live = false; if (made) URL.revokeObjectURL(made) }
  }, [api, file, title])

  if (file.protected) {
    return <p className="panel">This file is protected{file.drm_scheme ? ` (${file.drm_scheme})` : ''}. The server does not fulfil licenses, so this client cannot open it.</p>
  }
  if (view.kind === 'loading') return <p role="status" className="note">Loading file…</p>
  if (view.kind === 'error') return <div role="alert" className="panel error"><p>{view.message}</p></div>
  if (view.kind === 'unsupported') {
    return (
      <div className="panel">
        <p>This client does not render {file.format.toUpperCase()} yet. The server serves the original file only; it does not extract readable text.</p>
        <a className="button" href={view.url} download={view.name}>Download original ({file.format.toUpperCase()})</a>
      </div>
    )
  }
  return (
    <div>
      <div className="toolbar" role="group" aria-label="Text size">
        <button type="button" aria-label="Decrease text size" onClick={() => setSize((s) => Math.max(12, s - 2))}>A−</button>
        <button type="button" aria-label="Increase text size" onClick={() => setSize((s) => Math.min(32, s + 2))}>A+</button>
      </div>
      <article className="reading" aria-label={title} style={{ fontSize: size }}>
        {view.text.split(/\n{2,}/).map((p, i) => <p key={i}>{p}</p>)}
      </article>
    </div>
  )
}
