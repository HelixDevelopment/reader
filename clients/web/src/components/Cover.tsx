import { useEffect, useState } from 'react'
import type { Api } from '../lib/api'

/** Covers need the Authorization header, which <img src> cannot send, so they are fetched as blobs. */
export function Cover({ api, editionId, hasCover, title }: { api: Api; editionId?: string; hasCover: boolean; title: string }) {
  const [url, setUrl] = useState<string | null>(null)
  useEffect(() => {
    if (!editionId || !hasCover) return
    let live = true
    let made: string | null = null
    api.cover(editionId).then(
      (blob) => { if (live) { made = URL.createObjectURL(blob); setUrl(made) } },
      () => { /* no cover is a normal state; the placeholder stays */ },
    )
    return () => { live = false; if (made) URL.revokeObjectURL(made) }
  }, [api, editionId, hasCover])
  return url && hasCover
    ? <img className="cover" src={url} alt={`Cover of ${title}`} />
    : <div className="cover cover-fallback" role="img" aria-label={`No cover for ${title}`}><span aria-hidden="true">{title.slice(0, 1).toUpperCase()}</span></div>
}
