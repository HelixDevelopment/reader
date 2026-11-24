import { useEffect, useMemo, useState } from 'react'
import { createClient } from './lib/api'
import type { Api } from './lib/api'
import { loadSettings, saveSettings } from './lib/settings'
import type { Settings as S } from './lib/settings'
import { Library } from './components/Library'
import { Settings } from './components/Settings'
import { WorkDetail } from './components/WorkDetail'
import './App.css'

type Theme = 'system' | 'light' | 'dark'

function route(): { name: 'library' } | { name: 'work'; id: string } | { name: 'settings' } {
  const h = window.location.hash
  if (h === '#/settings') return { name: 'settings' }
  const m = /^#\/works\/(.+)$/.exec(h)
  return m ? { name: 'work', id: decodeURIComponent(m[1]) } : { name: 'library' }
}

export default function App() {
  const [settings, setSettings] = useState<S>(loadSettings)
  const [r, setR] = useState(route)
  const [theme, setTheme] = useState<Theme>('system')
  const built = useMemo<{ api: Api | null; error: string | null }>(() => {
    try { return { api: createClient(settings), error: null } } catch (e) { return { api: null, error: e instanceof Error ? e.message : String(e) } }
  }, [settings])
  const api = built.api

  useEffect(() => {
    const on = () => setR(route())
    window.addEventListener('hashchange', on)
    return () => window.removeEventListener('hashchange', on)
  }, [])
  useEffect(() => {
    if (theme === 'system') document.documentElement.removeAttribute('data-theme')
    else document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  const save = (s: S) => { saveSettings(s); setSettings(s); go('/') }
  const go = (hash: string) => { window.location.hash = hash }
  const nextTheme: Record<Theme, Theme> = { system: 'light', light: 'dark', dark: 'system' }

  return (
    <>
      <a className="skip" href="#main" onClick={(e) => { e.preventDefault(); document.getElementById('main')?.focus() }}>Skip to content</a>
      <header className="top">
        <a className="brand" href="#/">Helix Reader</a>
        <nav aria-label="Primary" className="row">
          <button type="button" onClick={() => setTheme(nextTheme[theme])} aria-label={`Theme: ${theme}. Switch theme`}>Theme: {theme}</button>
          <button type="button" onClick={() => go('/settings')}>Settings</button>
        </nav>
      </header>
      <main id="main" tabIndex={-1}>
        {!api && r.name !== 'settings' && <Settings value={settings} error={built.error} onSave={save} onCancel={() => go('/')} />}
        {api && r.name === 'library' && <Library api={api} onOpen={(id) => go('/works/' + encodeURIComponent(id))} />}
        {api && r.name === 'work' && <WorkDetail key={r.id} api={api} workId={r.id} onBack={() => go('/')} />}
        {r.name === 'settings' && <Settings value={settings} error={built.error} onSave={save} onCancel={() => go('/')} />}
      </main>
    </>
  )
}
