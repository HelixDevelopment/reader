# Phase: TASKS — epics → stories → subtasks (detail level: implementable)

## EPIC E1 — Core model & formats (M1) — STATUS: DONE (tests green x2)
- S1.1 HDM types ✅ T1.1.1 Publication/Spine/Block/Inline ✅ T1.1.2 WordCount ✅
- S1.2 Format registry + conversion matrix ✅ T1.2.1 matrix data ✅ T1.2.2 verdict fn ✅
- S1.3 ISBN/identifiers ✅ T1.3.1 ISBN-10 ✅ T1.3.2 ISBN-13 ✅ T1.3.3 page extraction ✅
- S1.4 HLC ✅ T1.4.1 mint ✅ T1.4.2 recv-merge ✅ T1.4.3 skew immunity ✅
- S1.5 Cache LRU ✅ T1.5.1 evict ✅ T1.5.2 protected-never ✅ T1.5.3 oversize reject ✅
- S1.6 Sources capability table ✅ T1.6.1 caps ✅ T1.6.2 insecure flag ✅

## EPIC E2 — Parsers (M2)
- S2.1 EPUB: T OCF zip, T OPF parse, T spine→HDM, T FXL detect, T media overlays
- S2.2 FB2: T XML→HDM, T binary images, T zip wrapper
- S2.3 TXT: T encoding detect (UTF/GBK/Big5/SJIS/125x), T chapter split regex packs
- S2.4 Golden corpus: T 20 fixture books, T snapshot tests (golden HDM JSON diff)
- S2.5 PDF: T MuPDF FFI binding decision (OWED-1), T text+quad extraction

## EPIC E3 — Sources & cache (M3)
- S3.1 SAL interface impl per protocol (18 backends, one story each: T dial/auth,
  T list/poll diff, T range read, T error taxonomy, T integration test in containers)
- S3.2 Streaming reader w/ read-ahead (T window mgr, T resume checkpoints)
- S3.3 Cache lifecycle (T state machine, T eviction hookup to S1.5, T pin API)
- S3.4 Source sync scheduler (T per-protocol diff, T battery-aware coalescing)

## EPIC E4 — Sync (M4)
- S4.1 CRDT doc types (T LWW register, T add-wins set, T op log, T snapshot compact)
- S4.2 Transports: Helix Cloud ws, WebDAV CAS, local-only
- S4.3 Position sync furthest/current + cross-format anchors
- S4.4 Chaos suite: 10k random partition/merge cycles, zero-loss invariant proof

## EPIC E5 — Identification/covers/migration/conversion (M5)
- S5.1 Identifier extraction (wire isbn pkg) T title-page OCR+NER T fingerprint search
- S5.2 Cover cascade (OpenLibrary by ISBN → providers → upscale → generated)
- S5.3 Migrators: Moon+ .mrexporter, Kindle MyClippings, Calibre metadata.db
- S5.4 Conversion engine (HDM writers: EPUB/FB2/TXT/HTMLZ/PDF-composer) + queue UI states

## EPIC E6 — Shells (M6) — per platform: navigation, reader surface, settings, share,
widgets; each shell story includes HelixQA flow test + screenshot baseline.

## EPIC E7 — AI/TTS/stats (M7) — providers, spoiler fence, SRS, heatmaps, coach.

Each subtask carries: acceptance test id, evidence artifact, estimate, owner.
