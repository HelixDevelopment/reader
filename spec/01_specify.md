# Phase: SPECIFY (SpecKit)
Source of requirements: helix_reader package docs/01–35 (Round 1–6), all ratified.

## What we build (one paragraph)
A 9-platform reading ecosystem with a Go shared core (HDM document model, format parsers,
conversion engine, source abstraction layer, CRDT sync, identification+covers, search,
stats, TTS, AI orchestration), thin native shells (Compose/SwiftUI/ArkUI/QML/Tauri/PWA),
Helix Cloud + WebDAV + self-host sync, and the Helix DS brand design system.

## Acceptance criteria (measurable, per constitution §7.1)
- AC-1: All 52 v1 features + R3/R4 additions (sources, conversion, sharing) traceable to
  tasks with tests. Traceability matrix: spec/05_analyze.md.
- AC-2: Core tests deterministic: two consecutive `go test ./... -count=1` runs identical
  (already enforced in evidence/).
- AC-3: Every screen S01–S24 passes HelixQA visual diff ≤1% vs approved baseline.
- AC-4: Performance budgets (docs/18) measured on reference devices via containers
  submodule emulators, recorded in evidence/.
- AC-5: ISBN/conversion/cache/HLC/source logic covered by unit tests (DONE for core
  skeleton — see evidence/go_test_run2.log).

## Out of scope v1
Audiobook store, DRM store, social network beyond notes sharing (docs/03 Won't list).
