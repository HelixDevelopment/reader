# Thorium Web — per-repository research report

- Repo: https://github.com/edrlab/thorium-web · Revision: `8d49c2b116e93ca50d4ccfdcb3226efde7bc49c8` (matches analysis pin; detached checkout of default branch `main`) · Commit date: 2026-10-07 18:00:07 +0200 — "Prepare new package version". Full non-shallow clone, fsck clean.
- Ledger: 678 tracked files (analysis `tracked_files=678` ✓), 6,862,005 bytes. Top: ts 252, tsx 161, svg 71, md 64, json 43.

**Architecture / data flow.** Next.js (App Router, deployed via OpenNext) web reader sharing the Readium TypeScript toolkits with Thorium Reader: OPDS 1/2 browsing, publication fetch/stream, ReadiumCSS paginated navigator in-browser, progression links compatible with the desktop app's `opds-progression` contract.

**Notable techniques.** Server-less deployment target (edge/OpenNext) with client-side LCP handling; same progression/position model as Thorium desktop — proving the contract is platform-neutral, exactly the property Helix wants for its resolver.

**License.** BSD-3-Clause, Copyright (c) 2024 EDRLab. Permissive, notice-preservation only.

**Porting decision — LEARN + reference implementation (attributed: Helix research agent, checkout `8d49c2b116`).** BSD-3 permits studying and selectively reusing the progression-position handling as the web-side counterpart to Thorium desktop; primary value is confirming the OPDS progression schema Helix implements against. No wholesale UI reuse (Next.js/React stack differs from Helix's).
