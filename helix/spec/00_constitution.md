# Helix Reader — Project Constitution (SpecKit)
Grounded in `submodules/constitution` — Helix Universal Constitution, revision 76
(verified present at submodules/constitution/Constitution.md, 12,132 lines).
This document is the project's binding process law; the universal constitution is
incorporated by reference. On conflict, the universal constitution wins.

## Adopted mandates (universal → project application)
1. **§1 Test coverage mandatory for every change** → no Go/package merge without tests;
   false-positive immunity (§1.1): tests must be able to fail (needle tests in CI).
2. **§7.1 NO BLUFF — positive-evidence-only validation** → every "done" claim requires a
   recorded artifact: test run log, screenshot, or measured number. `EVIDENCE.md` is
   append-only. This round's own example: ISBN + cache tests FAILED on first run and were
   fixed — the failure and fix are both recorded.
3. **§11.4.2 Recorded evidence** → every verification writes timestamped output to
   `evidence/`.
4. **§11.4.8 Deep-web-research-before-implementation** → no feature implemented before its
   research note exists (docs/ in helix_reader package + spec/research/).
5. **§11.4.6 No-guessing** → unverifiable claims are marked UNVERIFIED or removed.
6. **§11.4.31 Submodule manifest** → `helix-deps.yaml` at root; nested chains forbidden.
7. **§11.4.278 HelixQA mandatory UI testing** → all screens S01–S24 must pass HelixQA
   visual regression + flow tests before release claims.
8. **§9 Data safety** → journal-first annotation writes; no destructive op without backup.
9. **§2/§3/§4 Git mechanics** → submodule commits first, then parent; tags mirrored.
10. **§5/§6 Changelog + nano-detail documentation** → CHANGELOG.md per release; docs/
    updated in the same commit as code.

## Project-local rules
- R1: Shared core in **Go** (per operator decision 2026-10-14), packaged as c-shared/
  c-archive for Android (gomobile), iOS, HarmonyOS (N-API), Aurora (Qt C++), desktop, and
  compiled to WASM for web. Rust rendering hot-paths may be introduced behind the same FFI
  if profiling demands (decision gate: measured page-turn p95 > 16ms).
- R2: Nothing ships to users without passing the phase flow: constitution → specify →
  clarify → plan → tasks → analyze → execute → review → LIVE deterministic validation.
- R3: `submodules/qa` (HelixQA) owns UI/UX/visual verification; `submodules/containers`
  owns all device/emulator/simulator execution environments.
- R4: CodeGraph index must be fresh (`.codegraph/`) before any analyze/review phase.

## Mandatory credential and secret handling

> "Credentials or any secret and sensitive data MUST NOT leak! This MUST BE added as the mandatory constraint into all Submodules and main project's Constitution, CLAUDE.MD and AGENTS.MD."

Credentials, tokens, passwords, private keys, session material, connection-string passwords, and other sensitive values MUST NOT be committed, printed, logged, or included in test evidence or archives. Use environment-specific secret injection and redacted placeholders; sanitize errors and diagnostics at trust boundaries.
