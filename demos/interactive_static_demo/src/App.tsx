import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import {
  ArrowLeft, ArrowRight, BookOpen, Bookmark, Check, ChevronDown, CircleAlert,
  Cloud, Database, FileText, FolderSync, Highlighter, Library,
  Menu, Minus, Moon, MoreHorizontal, Plus, Search, Settings2, ShieldCheck,
  Smartphone, Sun, X
} from 'lucide-react'
import './App.css'

type Page = 'library' | 'review' | 'sources' | 'sync' | 'settings'
type Mode = 'catalog' | 'detail' | 'reader'
type Theme = 'paper' | 'sepia' | 'night'
type Book = {
  id: number; title: string; author: string; format: string; progress: number
  chapter: string; cover: string; year: string; pages: number; isbn: string
  files: { name: string; size: string; kind: string }[]
}
type ReviewItem = { id: number; title: string; field: string; local: string; proposed: string; source: string; status: 'pending' | 'accepted' | 'rejected' }
type Source = { id: number; name: string; kind: string; detail: string; status: string }

const books: Book[] = [
  { id: 1, title: 'The Glass Meridian', author: 'Elena Voss', format: 'EPUB', progress: 42, chapter: 'Chapter 7 · The Reading', cover: 'meridian', year: '2026', pages: 384, isbn: '978-1-64729-018-4', files: [{ name: 'the-glass-meridian.epub', size: '2.4 MB', kind: 'EPUB 3' }, { name: 'the-glass-meridian.mobi', size: '3.1 MB', kind: 'MOBI' }] },
  { id: 2, title: 'A Field Guide to Silence', author: 'Mira Sen', format: 'PDF', progress: 18, chapter: 'II · Rooms that listen', cover: 'silence', year: '2024', pages: 216, isbn: '978-1-64729-207-2', files: [{ name: 'field-guide-silence.pdf', size: '18.6 MB', kind: 'PDF' }] },
  { id: 3, title: 'Orbital Gardens', author: 'Theo Okafor', format: 'EPUB', progress: 71, chapter: 'Nine · Night cycle', cover: 'orbital', year: '2025', pages: 452, isbn: '978-1-64729-331-4', files: [{ name: 'orbital-gardens.epub', size: '5.8 MB', kind: 'EPUB 3' }] },
  { id: 4, title: 'The Cartographer’s Daughter', author: 'Ines Bell', format: 'FB2', progress: 6, chapter: 'One · The blank coast', cover: 'cartographer', year: '2023', pages: 328, isbn: '978-1-64729-441-0', files: [{ name: 'cartographer.fb2', size: '1.2 MB', kind: 'FictionBook 2' }] },
  { id: 5, title: 'Small Machines', author: 'Ren Ito', format: 'CBZ', progress: 89, chapter: 'Issue 4 of 6', cover: 'machines', year: '2026', pages: 148, isbn: 'No ISBN assigned', files: [{ name: 'small-machines-04.cbz', size: '64.2 MB', kind: 'Comic book ZIP' }] },
  { id: 6, title: 'Letters from the Night Train', author: 'Ada Moreau', format: 'MOBI', progress: 0, chapter: 'Not started', cover: 'letters', year: '2022', pages: 288, isbn: '978-1-64729-550-9', files: [{ name: 'night-train.mobi', size: '2.8 MB', kind: 'MOBI' }] },
]

const initialReviews: ReviewItem[] = [
  { id: 1, title: 'The Glass Meridian', field: 'Author', local: 'E. Voss', proposed: 'Elena Voss', source: 'Embedded OPF + Open Library', status: 'pending' },
  { id: 2, title: 'A Field Guide to Silence', field: 'Series', local: '—', proposed: 'Attentive environments', source: 'AI proposal · confidence 0.61', status: 'pending' },
  { id: 3, title: 'Letters from the Night Train', field: 'Publication year', local: '2022', proposed: '2023', source: 'Filename inference', status: 'pending' },
]

const initialSources: Source[] = [
  { id: 1, name: 'Home NAS', kind: 'SMB', detail: '/library/books · read-only', status: '1,284 files indexed' },
  { id: 2, name: 'Standard Ebooks', kind: 'OPDS 2.0', detail: 'Public catalog · no credentials', status: '68 acquisitions available' },
  { id: 3, name: 'Research archive', kind: 'WebDAV', detail: '/papers · range reads enabled', status: '312 files indexed' },
]

const chapters = [
  { title: 'Chapter 7 · The Reading', body: [
    'The archive breathed around her—a helical structure of ten thousand shelves spiraling into the dark, each spine a doorway someone had once stepped through and never quite returned from.',
    'Mara pressed her palm to the glass. Somewhere below, the curators were still reading aloud to the books, the way their order had done for six hundred years, because a story unread—they said—begins to dream.',
    'She thought of Senna’s warning at the meridian gate: glass remembers what light forgets. The ninth shelf down had been waiting for her since before she was born.',
  ] },
  { title: 'Chapter 8 · An Index of Doors', body: [
    'By morning, the ninth shelf had become a corridor. Its books were catalogued in no language she knew, yet every label rearranged itself when she looked away.',
    'Mara opened the smallest volume. Inside was a map of the archive as it might have been, with whole wings penciled in the margins and a single route marked in yellow.',
    'She copied the route into her notebook. Some doors, the curators had taught her, are only visible to those who leave a record of having found them.',
  ] },
  { title: 'Chapter 9 · The Unread', body: [
    'The first whisper came from beneath the floor. It was not a voice but the pressure of one, a sentence waiting for the shape of a listener.',
    'Mara stopped walking. Around her, thousands of unopened books shifted in their sleep, and the archive’s great spiral turned one degree toward the sun.',
    'For the first time she understood that the curators were not protecting the world from the books. They were protecting the books from being forgotten.',
  ] },
]

function Cover({ book, large = false }: { book: Book; large?: boolean }) {
  return <div className={`cover cover-${book.cover} ${large ? 'cover-large' : ''}`} aria-label={`${book.title} cover`}>
    <span className="cover-rule" />
    <span className="cover-title">{book.title}</span>
    <span className="cover-author">{book.author}</span>
    <span className="cover-mark">H</span>
  </div>
}

function IconButton({ label, children, onClick, active = false }: { label: string; children: ReactNode; onClick: () => void; active?: boolean }) {
  return <button type="button" className={`icon-button ${active ? 'active' : ''}`} aria-label={label} title={label} onClick={onClick}>{children}</button>
}

export default function App() {
  const [page, setPage] = useState<Page>('library')
  const [mode, setMode] = useState<Mode>('catalog')
  const [selected, setSelected] = useState<Book>(books[0])
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState('All')
  const [theme, setTheme] = useState<Theme>('paper')
  const [fontSize, setFontSize] = useState(20)
  const [lineHeight, setLineHeight] = useState(1.7)
  const [chapter, setChapter] = useState(0)
  const [readerPanel, setReaderPanel] = useState<'appearance' | 'notes' | null>(null)
  const [notes, setNotes] = useState<string[]>(['“Glass remembers what light forgets.”'])
  const [reviews, setReviews] = useState(initialReviews)
  const [sources, setSources] = useState(initialSources)
  const [addSourceOpen, setAddSourceOpen] = useState(false)
  const [sourceName, setSourceName] = useState('')
  const [sourceKind, setSourceKind] = useState('OPDS 2.0')
  const [sourceError, setSourceError] = useState('')
  const [conflictResolved, setConflictResolved] = useState(false)
  const [toast, setToast] = useState('')
  const [mobileMenu, setMobileMenu] = useState(false)
  const modalRef = useRef<HTMLElement>(null)
  const addSourceTriggerRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!addSourceOpen) return
    const modal = modalRef.current
    const trigger = addSourceTriggerRef.current
    if (!modal) return
    const focusableItems = () => Array.from(modal.querySelectorAll<HTMLElement>('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')).filter((el) => !el.hasAttribute('disabled'))
    const trapFocus = (event: KeyboardEvent) => {
      if (event.key !== 'Tab') return
      const items = focusableItems()
      if (!items.length) return
      const first = items[0]
      const last = items[items.length - 1]
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
      else if (!modal.contains(document.activeElement)) { event.preventDefault(); first.focus() }
    }
    document.addEventListener('keydown', trapFocus)
    return () => { document.removeEventListener('keydown', trapFocus); trigger?.focus() }
  }, [addSourceOpen])

  useEffect(() => {
    if (!toast) return
    const id = window.setTimeout(() => setToast(''), 2400)
    return () => window.clearTimeout(id)
  }, [toast])

  useEffect(() => {
    const close = (event: KeyboardEvent) => {
      if (event.key === 'Escape') { setAddSourceOpen(false); setReaderPanel(null); setMobileMenu(false) }
    }
    window.addEventListener('keydown', close)
    return () => window.removeEventListener('keydown', close)
  }, [])

  const filtered = useMemo(() => books.filter((book) => {
    const matchesQuery = `${book.title} ${book.author} ${book.format}`.toLowerCase().includes(query.toLowerCase())
    return matchesQuery && (filter === 'All' || book.format === filter || (filter === 'Reading' && book.progress > 0 && book.progress < 100))
  }), [query, filter])

  const pendingCount = reviews.filter((item) => item.status === 'pending').length
  const navigate = (next: Page) => { setPage(next); setMode('catalog'); setMobileMenu(false) }
  const openBook = (book: Book) => { setSelected(book); setMode('detail'); setChapter(Math.min(2, Math.floor(book.progress / 34))) }
  const readBook = (book = selected) => { setSelected(book); setMode('reader'); setReaderPanel(null) }
  const notify = (message: string) => setToast(message)
  const addNote = () => {
    const passage = chapters[chapter].body[1].split('.')[0] + '.'
    if (!notes.includes(passage)) setNotes([passage, ...notes])
    setReaderPanel('notes')
    notify('Passage saved to this prototype session')
  }
  const submitSource = (event: FormEvent) => {
    event.preventDefault()
    if (sourceName.trim().length < 2) { setSourceError('Enter a descriptive source name.'); return }
    setSources([...sources, { id: Date.now(), name: sourceName.trim(), kind: sourceKind, detail: 'Configuration not connected in prototype', status: 'Not scanned' }])
    setSourceName(''); setSourceError(''); setAddSourceOpen(false); notify('Source added to the sample workspace')
  }

  const navItems: { id: Page; label: string; icon: ReactNode; count?: number }[] = [
    { id: 'library', label: 'Library', icon: <Library size={19} /> },
    { id: 'review', label: 'Metadata review', icon: <ShieldCheck size={19} />, count: pendingCount },
    { id: 'sources', label: 'Sources', icon: <Database size={19} /> },
    { id: 'sync', label: 'Synchronization', icon: <FolderSync size={19} /> },
    { id: 'settings', label: 'Reading settings', icon: <Settings2 size={19} /> },
  ]

  return <div className="app" data-theme={theme}>
    <a className="skip-link" href="#main-content">Skip to content</a>
    <aside className={`sidebar ${mobileMenu ? 'mobile-open' : ''}`}>
      <button className="brand" onClick={() => navigate('library')} aria-label="Helix library home">
        <span className="brand-symbol">h</span><span>helix<span className="brand-light">reader</span></span>
      </button>
      <div className="sidebar-label">Your reading space</div>
      <nav aria-label="Primary navigation">
        {navItems.map((item) => <button key={item.id} className={`nav-item ${page === item.id ? 'selected' : ''}`} onClick={() => navigate(item.id)}>
          {item.icon}<span>{item.label}</span>{item.count !== undefined && item.count > 0 && <span className="nav-count">{item.count}</span>}
        </button>)}
      </nav>
      <div className="sidebar-bottom">
        <div className="storage-line"><Cloud size={16} /><span>Local workspace</span></div>
        <p>Sample data only. No files leave this browser.</p>
      </div>
    </aside>
    {mobileMenu && <button className="sidebar-backdrop" aria-label="Close navigation" onClick={() => setMobileMenu(false)} />}

    <main id="main-content" className="main">
      <header className="topbar">
        <IconButton label="Open navigation" onClick={() => setMobileMenu(true)}><Menu size={21} /></IconButton>
        <div className="prototype-label"><span /> Interactive product prototype</div>
        <div className="topbar-actions">
          <span className="sample-note">No backend connected</span>
          <IconButton label={theme === 'night' ? 'Use paper theme' : 'Use night theme'} onClick={() => setTheme(theme === 'night' ? 'paper' : 'night')}>{theme === 'night' ? <Sun size={19} /> : <Moon size={19} />}</IconButton>
        </div>
      </header>
      <p className="mobile-sample-note">Sample data only · no backend connected</p>

      {page === 'library' && mode === 'catalog' && <section className="page-section library-page">
        <div className="page-heading">
          <div><p className="eyebrow">Personal library / 1,596 indexed files</p><h1>Find your next<br /><em>way back in.</em></h1></div>
          <p className="heading-note">Books stay where they are.<br />Editions, evidence and reading state stay together.</p>
        </div>
        <div className="catalog-tools">
          <label className="search-box"><Search size={20} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search title, author or format" aria-label="Search library" />{query && <button onClick={() => setQuery('')} aria-label="Clear search"><X size={17} /></button>}</label>
          <div className="filter-row" role="group" aria-label="Library filters">
            {['All', 'Reading', 'EPUB', 'PDF', 'FB2', 'MOBI', 'CBZ'].map((item) => <button key={item} className={`filter ${filter === item ? 'active' : ''}`} onClick={() => setFilter(item)}>{item}</button>)}
          </div>
        </div>
        {!query && filter === 'All' && <article className="continue-feature">
          <Cover book={books[0]} large />
          <div className="continue-copy">
            <p className="eyebrow">Continue reading · {books[0].progress}%</p>
            <h2>{books[0].title}</h2><p className="byline">{books[0].author}</p>
            <p className="chapter-name">{books[0].chapter}</p>
            <div className="progress-track"><span style={{ width: `${books[0].progress}%` }} /></div>
            <button className="primary-button" onClick={() => readBook(books[0])}><BookOpen size={18} /> Resume chapter <ArrowRight size={18} /></button>
          </div>
          <div className="feature-aside"><span>18</span><p>minutes left<br />in this chapter</p></div>
        </article>}
        <div className="section-heading"><h2>{query || filter !== 'All' ? `${filtered.length} matching books` : 'Recently added'}</h2><span>{filtered.length} shown</span></div>
        {filtered.length ? <div className="book-grid">{filtered.map((book) => <article className="book-card" key={book.id}>
          <button className="book-open" onClick={() => openBook(book)} aria-label={`Open ${book.title}`}><Cover book={book} /></button>
          <div className="book-meta"><span>{book.format}</span><span>{book.progress ? `${book.progress}% read` : 'Unread'}</span></div>
          <button className="book-title" onClick={() => openBook(book)}>{book.title}</button><p>{book.author}</p>
        </article>)}</div> : <div className="empty-state"><Search size={28} /><h3>No books match this view</h3><p>Clear the search or choose another format.</p><button className="text-button" onClick={() => { setQuery(''); setFilter('All') }}>Reset filters</button></div>}
      </section>}

      {page === 'library' && mode === 'detail' && <section className="page-section detail-page">
        <button className="back-button" onClick={() => setMode('catalog')}><ArrowLeft size={18} /> Library</button>
        <div className="detail-layout">
          <div className="detail-cover"><Cover book={selected} large /><span className="edition-label">Primary edition · {selected.year}</span></div>
          <div className="detail-copy">
            <p className="eyebrow">{selected.format} / verified local metadata</p><h1>{selected.title}</h1><p className="detail-author">{selected.author}</p>
            <p className="detail-description">A reading copy preserved in its original location, with edition-specific identifiers and every local file kept distinct.</p>
            <div className="detail-actions"><button className="primary-button" onClick={() => readBook()}><BookOpen size={18} /> {selected.progress ? 'Continue reading' : 'Start reading'}</button><button className="secondary-button" onClick={() => { navigate('review'); notify('Review queue opened') }}><ShieldCheck size={18} /> Review metadata</button></div>
            <dl className="metadata-grid"><div><dt>Publisher</dt><dd>Northline Press</dd></div><div><dt>Published</dt><dd>{selected.year}</dd></div><div><dt>Length</dt><dd>{selected.pages} pages</dd></div><div><dt>ISBN</dt><dd>{selected.isbn}</dd></div></dl>
          </div>
        </div>
        <div className="files-section"><div className="section-heading"><h2>Files in this edition</h2><span>{selected.files.length} local originals</span></div>
          {selected.files.map((file) => <div className="file-row" key={file.name}><FileText size={21} /><div><strong>{file.name}</strong><span>{file.kind} · {file.size}</span></div><span className="verified"><Check size={15} /> SHA-256 indexed</span><IconButton label={`More actions for ${file.name}`} onClick={() => notify(`${file.name}: original file remains unchanged`)}><MoreHorizontal size={19} /></IconButton></div>)}
        </div>
      </section>}

      {page === 'library' && mode === 'reader' && <section className="reader-shell">
        <header className="reader-topbar"><button className="back-button" onClick={() => setMode('detail')}><ArrowLeft size={18} /><span>{selected.title}</span></button><span className="reader-position">{chapter + 1} / {chapters.length}</span><div><IconButton label="Reader notes" active={readerPanel === 'notes'} onClick={() => setReaderPanel(readerPanel === 'notes' ? null : 'notes')}><Bookmark size={19} /></IconButton><IconButton label="Appearance" active={readerPanel === 'appearance'} onClick={() => setReaderPanel(readerPanel === 'appearance' ? null : 'appearance')}><Settings2 size={19} /></IconButton></div></header>
        <div className="reader-layout">
          <article className="reading-page" style={{ fontSize: `${fontSize}px`, lineHeight }}>
            <p className="eyebrow">{selected.author} · {selected.title}</p><h1>{chapters[chapter].title}</h1>
            <p className="sample-chapter-note">Sample chapter text — the same placeholder passages are shown for every book in this prototype.</p>
            {chapters[chapter].body.map((paragraph, index) => <p key={paragraph}>{index === 1 && chapter === 0 ? <>Mara pressed her palm to the glass. Somewhere below, the curators were still reading aloud to the books, the way their order had done for six hundred years, because a story unread—they said—begins to <mark>dream</mark>.</> : paragraph}</p>)}
            <div className="chapter-end"><span>·</span><button className="text-button" onClick={addNote}><Highlighter size={17} /> Save a passage from this chapter</button><span>·</span></div>
          </article>
          {readerPanel && <aside className="reader-panel" aria-label={readerPanel === 'notes' ? 'Reader notes' : 'Appearance settings'}>
            <div className="panel-heading"><h2>{readerPanel === 'notes' ? 'Passages & notes' : 'Appearance'}</h2><IconButton label="Close panel" onClick={() => setReaderPanel(null)}><X size={18} /></IconButton></div>
            {readerPanel === 'appearance' ? <>
              <label>Text size <output>{fontSize}px</output></label><div className="stepper"><button onClick={() => setFontSize(Math.max(16, fontSize - 1))} disabled={fontSize <= 16} aria-label="Decrease text size"><Minus size={17} /></button><input aria-label="Text size" type="range" min="16" max="28" value={fontSize} onChange={(event) => setFontSize(Number(event.target.value))} /><button onClick={() => setFontSize(Math.min(28, fontSize + 1))} disabled={fontSize >= 28} aria-label="Increase text size"><Plus size={17} /></button></div>
              <label>Line spacing <output>{lineHeight.toFixed(1)}</output></label><input aria-label="Line spacing" type="range" min="1.4" max="2" step="0.1" value={lineHeight} onChange={(event) => setLineHeight(Number(event.target.value))} />
              <label>Reading theme</label><div className="theme-options">{(['paper', 'sepia', 'night'] as Theme[]).map((item) => <button key={item} className={theme === item ? 'active' : ''} onClick={() => setTheme(item)}>{item}</button>)}</div>
            </> : <>{notes.map((note, index) => <div className="note" key={`${note}-${index}`}><Highlighter size={16} /><p>{note}</p><span>Prototype session · {chapters[chapter].title}</span></div>)}<button className="secondary-button full" onClick={addNote}><Plus size={17} /> Save current passage</button></>}
          </aside>}
        </div>
        <footer className="reader-footer"><button onClick={() => setChapter(Math.max(0, chapter - 1))} disabled={chapter === 0}><ArrowLeft size={18} /> Previous</button><div><div className="progress-track"><span style={{ width: `${((chapter + 1) / chapters.length) * 100}%` }} /></div><span>{chapters[chapter].title}</span></div><button onClick={() => setChapter(Math.min(chapters.length - 1, chapter + 1))} disabled={chapter === chapters.length - 1}>Next <ArrowRight size={18} /></button></footer>
      </section>}

      {page === 'review' && <section className="page-section narrow-page">
        <div className="page-heading compact"><div><p className="eyebrow">Evidence before automation</p><h1>Metadata review</h1></div><p className="heading-note">AI and filename proposals never overwrite deterministic evidence.</p></div>
        <div className="review-summary"><ShieldCheck size={24} /><div><strong>{pendingCount} decisions need a human</strong><p>Accept or reject each proposed field independently.</p></div></div>
        <div className="review-list">{reviews.map((item) => <article className={`review-item ${item.status}`} key={item.id}>
          <div className="review-book"><span>{item.title}</span><strong>{item.field}</strong></div><div className="value-comparison"><div><span>Current local value</span><strong>{item.local}</strong></div><ArrowRight size={19} /><div><span>Proposed value</span><strong>{item.proposed}</strong></div></div>
          <div className="review-footer"><span>{item.source}</span>{item.status === 'pending' ? <div><button className="secondary-button" onClick={() => setReviews(reviews.map((r) => r.id === item.id ? { ...r, status: 'rejected' } : r))}><X size={16} /> Reject</button><button className="primary-button small" onClick={() => setReviews(reviews.map((r) => r.id === item.id ? { ...r, status: 'accepted' } : r))}><Check size={16} /> Accept proposal</button></div> : <span className={`decision ${item.status}`}>{item.status === 'accepted' ? <Check size={16} /> : <X size={16} />}{item.status}</span>}</div>
        </article>)}</div>
      </section>}

      {page === 'sources' && <section className="page-section narrow-page">
        <div className="page-heading compact"><div><p className="eyebrow">Index in place</p><h1>Connected sources</h1></div><button className="primary-button" ref={addSourceTriggerRef} onClick={() => setAddSourceOpen(true)}><Plus size={18} /> Add source</button></div>
        <p className="page-intro">Connectors read originals without reorganizing the library. This prototype changes only local interface state.</p>
        <div className="source-list">{sources.map((source) => <article className="source-row" key={source.id}><div className="source-icon"><Database size={22} /></div><div className="source-copy"><span>{source.kind}</span><h2>{source.name}</h2><p>{source.detail}</p></div><div className="source-status"><span>{source.status}</span><button className="secondary-button" onClick={() => { setSources(sources.map((s) => s.id === source.id ? { ...s, status: 'Scan queued in prototype' } : s)); notify(`Scan queued for ${source.name}`) }}><FolderSync size={17} /> Scan</button></div></article>)}</div>
      </section>}

      {page === 'sync' && <section className="page-section narrow-page">
        <div className="page-heading compact"><div><p className="eyebrow">Reading state</p><h1>Synchronization</h1></div><span className="status-badge"><span /> Sync model demo</span></div>
        <div className="sync-devices"><div><Smartphone size={24} /><strong>Phone</strong><span>{conflictResolved ? 'Chapter 8 · 47%' : 'Chapter 7 · 42%'}</span></div><div><BookOpen size={24} /><strong>Desktop</strong><span>Chapter 8 · 47%</span></div><div><Database size={24} /><strong>Server</strong><span>{conflictResolved ? 'Resolved · 47%' : 'Conflict pending'}</span></div></div>
        <article className="conflict-card"><div className="conflict-heading"><CircleAlert size={23} /><div><h2>{conflictResolved ? 'Conflict resolved' : 'One conflict needs a decision'}</h2><p>The Glass Meridian · same content identity</p></div></div>{!conflictResolved ? <><div className="conflict-options"><div><span>Phone · vector clock 12:4</span><strong>42%</strong><p>Chapter 7 · updated 21:14</p></div><div><span>Desktop · vector clock 12:5</span><strong>47%</strong><p>Chapter 8 · updated 21:18</p></div></div><button className="primary-button" onClick={() => { setConflictResolved(true); notify('Furthest position selected') }}><Check size={18} /> Keep furthest position · 47%</button></> : <div className="resolved-message"><Check size={20} /><p>Chapter 8 at 47% is now the shared position in this sample session.</p></div>}</article>
        <div className="sync-rule"><ShieldCheck size={20} /><p><strong>Deterministic rule:</strong> compatible vector clocks merge automatically; divergent positions retain both candidates until a policy or user resolves them.</p></div>
      </section>}

      {page === 'settings' && <section className="page-section narrow-page">
        <div className="page-heading compact"><div><p className="eyebrow">One reading language</p><h1>Reading settings</h1></div><p className="heading-note">Changes apply immediately to this prototype.</p></div>
        <div className="settings-group"><h2>Theme</h2><div className="theme-cards">{(['paper', 'sepia', 'night'] as Theme[]).map((item) => <button key={item} className={`theme-card ${item} ${theme === item ? 'active' : ''}`} onClick={() => setTheme(item)}><span className="theme-sample">Aa</span><strong>{item}</strong>{theme === item && <Check size={17} />}</button>)}</div></div>
        <div className="settings-group"><h2>Typography</h2><label className="setting-label">Default text size <output>{fontSize}px</output></label><input aria-label="Default text size" type="range" min="16" max="28" value={fontSize} onChange={(event) => setFontSize(Number(event.target.value))} /><label className="setting-label">Default line spacing <output>{lineHeight.toFixed(1)}</output></label><input aria-label="Default line spacing" type="range" min="1.4" max="2" step="0.1" value={lineHeight} onChange={(event) => setLineHeight(Number(event.target.value))} /></div>
        <div className="settings-group capability-group"><h2>Capability honesty</h2><p>Detection and byte streaming do not imply rendering, conversion or DRM fulfillment.</p>{[['EPUB', 'Metadata · stream · convert paths', 'Native parser'], ['PDF', 'Detect · byte stream', 'Renderer not connected'], ['LCP packages', 'Detect · protected byte stream', 'License fulfillment not connected']].map(([name, support, limit]) => <div className="capability-row" key={name}><strong>{name}</strong><span>{support}</span><small>{limit}</small></div>)}</div>
      </section>}
    </main>

    <nav className="mobile-nav" aria-label="Mobile navigation">{navItems.slice(0, 4).map((item) => <button key={item.id} className={page === item.id ? 'selected' : ''} onClick={() => navigate(item.id)}>{item.icon}<span>{item.id === 'review' ? 'Review' : item.label}</span>{item.id === 'review' && pendingCount > 0 && <b>{pendingCount}</b>}</button>)}<button className={page === 'settings' ? 'selected' : ''} onClick={() => navigate('settings')}><Settings2 size={19} /><span>Settings</span></button></nav>

    {addSourceOpen && <div className="modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget) setAddSourceOpen(false) }}><section className="modal" role="dialog" aria-modal="true" aria-labelledby="add-source-title" ref={modalRef}><div className="panel-heading"><h2 id="add-source-title">Add a sample source</h2><IconButton label="Close dialog" onClick={() => setAddSourceOpen(false)}><X size={19} /></IconButton></div><form onSubmit={submitSource}><label htmlFor="source-name">Source name</label><input id="source-name" autoFocus value={sourceName} onChange={(event) => setSourceName(event.target.value)} placeholder="e.g. Upstairs NAS" aria-describedby={sourceError ? 'source-error' : undefined} /><label htmlFor="source-kind">Connector</label><div className="select-wrap"><select id="source-kind" value={sourceKind} onChange={(event) => setSourceKind(event.target.value)}>{['OPDS 2.0', 'OPDS 1.2', 'WebDAV', 'SMB', 'SFTP', 'S3', 'HTTP(S)', 'Local folder'].map((kind) => <option key={kind}>{kind}</option>)}</select><ChevronDown size={17} /></div>{sourceError && <p className="form-error" id="source-error"><CircleAlert size={16} />{sourceError}</p>}<p className="form-note">No network connection or credentials are used in this prototype.</p><button className="primary-button full" type="submit"><Plus size={18} /> Add to sample workspace</button></form></section></div>}
    {toast && <div className="toast" role="status"><Check size={17} />{toast}</div>}
  </div>
}
