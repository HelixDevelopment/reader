# Phase: PLAN
## Milestones
M1 Core foundations (this round): module, HDM, formats matrix, ISBN, HLC, cache, sources
   caps + deterministic tests ✅ (evidence/).
M2 Parsers EPUB/FB2/TXT → HDM; renderer contract tests; golden-book corpus.
M3 Sources SAL backends: local, OPDS, WebDAV, SFTP, SMB, HTTP, S3 (+cloud drives);
   streaming cache; sync destinations decoupled.
M4 Sync engine (CRDT over HLC) + cloud API + self-host; chaos tests.
M5 Identification + covers pipeline; migration importers; conversion engine + queue.
M6 Shells: Android first, then iOS, desktop, web; HarmonyOS/Aurora via containers matrix.
M7 AI layer, TTS, stats; HelixQA full visual pass; release candidate.
## Environments (containers submodule)
Android API 30/34 emulators, iOS sim, HarmonyOS emulator, Aurora SDK VM, Linux/Win/mac
runners, e-ink simulator profile. All UI flows executed by HelixQA with Vision eyes.
## CodeGraph
`codegraph init` at repo root after each milestone; analyze/review phases require fresh
index (R4). Install: npm i -g @colbymchenry/codegraph (validated upstream install path).
