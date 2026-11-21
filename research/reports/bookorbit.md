# BookOrbit — per-repository research report

## Pinned revisions and file ledgers

- Repository: https://github.com/bookorbit/bookorbit
- Backend checkout (slug `bookorbit-bookorbit`): revision `f2a7a111881899e7c6cb233a0ec2d8a9c25235c6`, tree `f1074f1ca0257d226f40762c614264f053879798`, branch `origin/main` (default), commit date 2026-10-28 02:49:03 +0000 — `build(deps): update minor and patch dependencies`. Ledger recomputed from this checkout: 5,902 files, 155,279,142 bytes — matches analysis `tracked_files=5902`. Top extensions: ts 4,257 / 31,158,447; vue 901 / 6,735,571; lua 212 / 2,972,030; json 194 / 80,119,463; sql 135 / 292,939; mjs 41 / 366,563.
- Client checkout (slug `bookorbit`): pinned analysis revision `99aa7dce705986eac29a8f73aa65331de5f50680` (commit date 2026-10-21 19:51:36 +0000 — `fix(reader): preserve third-party KOReader menu entries`) is present in the same clone; `git ls-tree -r 99aa7dce` yields exactly 5,549 files, matching the client analysis `tracked_files=5549`. Because the current checkout HEAD is `f2a7a111`, `current_checkout_matches_analysis` is correctly `false` for the client row.
- Checkout evidence: full non-shallow clone, `git fsck` clean.

## Architecture and data flow

pnpm monorepo (`pnpm-workspace.yaml`) with four runtime surfaces sharing `packages/`:

- `server/` — NestJS on Fastify (`@fastify/compress`, `@fastify/cookie`), Drizzle ORM over PostgreSQL (`pg`). ~50 feature modules under `server/src/modules/`: `auth`, `account-activity`, `annotation`, `book`, `book-cover-store`, `book-duplicates`, `book-journal`, `book-metadata-fetch`, `book-metadata-lock`, `book-move`, `book-request`, `browse-counts`, `catalog`, `chapter-editor`, `chapter-health`, `audiobook`, `audiobook-chapters`, `audit`, `achievement`, `app-info`, `app-settings`, `authors`, … Swagger surface generated in `swagger.ts`. SQL migrations in `*_schema.sql` / drizzle `_meta`.
- `client/` — Vue 3 + Vite PWA (`pwa-assets.config.ts`, `vite.config.ts`): `src/{features,components,composables,stores,views,services}`; browser-side reader.
- `koreader-plugin/bookorbit.koplugin/` — Lua plugin for KOReader (`bookorbit_api.lua`, `bookorbit_annotations.lua`, plus a `board_*` UI subsystem) talking HTTP to the server; `spec/` and `tools/` for offline verification.
- `workers/read-aloud/` — worker-based text-to-speech service.
- `packages/{plugin-api,types}` — shared contracts consumed by server, client, and the Lua plugin.

Flow: books land in the Calibre-compatible library (`server/requirements` documents Calibre expectations) → catalog/metadata modules index them → REST + catalog feeds serve the Vue client and the KOReader plugin → annotations/progress/journal persist via Drizzle/Postgres → read-aloud workers synthesize chapter audio.

## Notable implementation techniques

- One API contract package feeding three heterogeneous clients (Vue PWA, KOReader Lua, internal workers) — a clean model for Helix's own Go-API + web + e-ink targets.
- KOReader plugin's `board_*` modules render a grid dashboard inside KOReader; `bookorbit_activity.lua` mirrors server activity locally.
- Chapter-health/chapter-duplicate detection and metadata-lock show an ingest pipeline that repairs rather than merely stores; `kobo-cloudscraper.txt` documents Kobo metadata harvesting.
- Auditing/account-activity/achievement modules are organized as vertical slices, each with its own tests (`main.test.ts`, per-module `.spec`).

## License review — AGPL-3.0-only WITH Section 7 additional terms (not plain AGPL)

- `LICENSE` is GNU AGPL v3 text; `ADDITIONAL_TERMS.md` (effective 2026-09-10, "Copyright (C) 2025-2026 neon") adds non-permissive terms under §7(b)-(e):
  1. Mandatory prominent "Powered by BookOrbit" attribution + full legal notice in every interactive UI of any covered work, accessible without admin rights, never removable or de-emphasized;
  2. The same UI feature must offer Complete Corresponding Source of the exact running version when §13 applies (linking only to the original repo is insufficient for modified versions);
  3. Modified versions must display a "not an official BookOrbit release" statement with modification date;
  4. No trademark grant; "BookOrbit" may not be the primary name of a derivative;
  5. These terms and notices must be preserved in source distributions; compliance is an express condition of permission (§8 applies).
- Helix's server is not an AGPL project. Even for an AGPL server, the §7 attribution/modified-version/source-offer UI duties would have to be reproduced in Helix's UI, and the terms apply per-file ("other contributions remain subject to the license terms under which they were originally submitted" — per-file provenance review would be mandatory before reuse of any snippet).

## Porting decision — attributed

**Decision: LEARN ONLY — architecture and API-shape study; NO code embedding, NO snippet port.** (Reviewer: Helix research agent, from pinned checkout `f2a7a111` / client pin `99aa7dce`.)

Rationale: AGPL-3.0-only network copyleft combined with §7 additional terms makes any embedding of BookOrbit material into Helix a licensing upgrade plus a chain of UI-visible legal obligations that materially change the product. The value here is purely informational: its NestJS module decomposition, the shared `plugin-api` contract pattern across web/KOReader clients, its annotation and chapter-health pipelines, and its KOReader plugin protocol. Reimplementation from ideas is unrestricted; copying expression is prohibited for Helix's current license posture. Recorded as `researched_blocked_by_license` in the porting ledger.
