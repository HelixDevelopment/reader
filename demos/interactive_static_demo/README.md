# helixreader — interactive product prototype

A static, front-end-only prototype of the Helix reader product UI (library,
metadata review, sources, sync model, and reading settings), built with React,
TypeScript, Vite, and Tailwind CSS.

## Important: sample-only prototype

- **No backend is connected.** There is no server, database, network sync, or
  persistence of any kind. All state lives in component memory and is reset on
  page reload.
- **All content is fictional sample data.** Book titles, authors, ISBNs,
  sources, devices, and sync states are invented placeholders.
- **The reader shows the same shared sample chapter text for every book.**
  It is original placeholder prose written for this prototype, not real book
  content.
- The synchronization screen is a **UI demo of the conflict-resolution model
  only** — no data is ever transmitted or merged.

## Requirements

- Node.js 20+

## Run locally

```bash
npm install
npm run dev
```

Then open the URL printed by Vite (default `http://localhost:5173`).

## Build

```bash
npm run build    # type-checks with tsc, then bundles to dist/
npm run preview  # serves the built bundle locally (default port 4173)
```

## Lint

```bash
npm run lint
```

## Project structure

- `src/App.tsx` — the entire prototype UI and in-memory state
- `src/App.css` — prototype styles
- `src/index.css` — global styles
- `src/main.tsx` — entry point
- `src/components/ui/` — shadcn/ui component library
- `src/hooks/` — custom hooks
- `src/lib/` — utilities
- `public/fonts/` — bundled fonts (DejaVu Sans Mono, Noto Sans, Noto Serif
  Display; see the included license files)
