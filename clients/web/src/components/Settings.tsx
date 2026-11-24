import { useState } from 'react'
import type { FormEvent } from 'react'
import { validateBaseUrl } from '../lib/settings'
import type { Settings as S } from '../lib/settings'

export function Settings({ value, error: initialError, onSave, onCancel }: { value: S; error?: string | null; onSave: (s: S) => void; onCancel: () => void }) {
  const [draft, setDraft] = useState(value)
  const error = validateBaseUrl(draft.baseUrl)
  const submit = (e: FormEvent) => { e.preventDefault(); if (!error) onSave({ ...draft, baseUrl: draft.baseUrl.trim() }) }
  return (
    <form className="panel settings" onSubmit={submit} aria-labelledby="settings-h">
      <h1 id="settings-h">Server settings</h1>
      {initialError && <p role="alert" className="error-text">{initialError}</p>}
      <label>Server address
        <input value={draft.baseUrl} placeholder="same origin" aria-invalid={!!error} aria-describedby="base-help" onChange={(e) => setDraft({ ...draft, baseUrl: e.target.value })} />
      </label>
      <p id="base-help" className={error ? 'error-text' : 'muted'}>{error ?? 'Leave empty when this page is served by (or proxied to) the Helix server.'}</p>
      <label>Username (optional)
        <input value={draft.username} autoComplete="username" onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
      </label>
      <label>Password (kept for this session only)
        <input type="password" value={draft.password} autoComplete="current-password" onChange={(e) => setDraft({ ...draft, password: e.target.value })} />
      </label>
      <div className="row">
        <button type="submit" disabled={!!error}>Save</button>
        <button type="button" onClick={onCancel}>Cancel</button>
      </div>
    </form>
  )
}
