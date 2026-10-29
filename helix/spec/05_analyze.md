# Phase: ANALYZE — traceability & gap closure
## Traceability (sample, full matrix generated in CI)
LIB-001..008→E6 shell stories · READ-001→E2 · READ-002..007→E6+core · ANN-001..009→E4/E5 ·
SYNC-001..005→E4 · CTL-001..005→E6 · SRC (docs25)→E3 · CONV (docs31)→E5.4+formats pkg ·
SHARE (docs28)→E6 · IDENT (docs26)→E5.1/5.2
## Gaps found in analyze
- OWED-1: PDF engine licensing (MuPDF AGPL!) — resolution: Artifex commercial license OR
  pdfium (BSD, Chromium) — DECISION: pdfium. Tasks updated E2.5.
- OWED-2: containers matrix schema must be adopted from submodules/containers, not
  invented — action: read containers docs before writing .helix/matrix.yaml.
- OWED-3: Go EPUB a11y (media overlays) — SMIL parser needed; added E2.1 task.
- Improvement: property-based test for HLC merge commutativity added to E4.4.
- Improvement: conversion determinism golden tests (same input → byte-identical EPUB)
  added E5.4, satisfying constitution false-positive immunity.
