# Upstream Porting and Interoperability Ledger

This ledger distinguishes protocol learning, original implementation, and copied code. No entry implies upstream endorsement. License identification is subject to the per-repository notice review; incompatible code must not be copied.

## KOReader progress synchronization — implemented original protocol surface

**Upstream evidence**

- `fellipec/kosync-go`, revision `f74cc7409ff6035212f09be877963c1949145fdf`, tree `d48ba3e561aec4a805142a0055ddf4819afca71d`.
  - `handlers.go`: account creation/auth routes, `x-auth-user` / `x-auth-key`, bounded requests, bcrypt verifier handling, progress fields, server timestamps.
  - `store.go`: progress wire shape, atomic JSON persistence, user → document → progress identity.
  - `README.md`: registration lockout, TLS requirement, reverse-proxy deployment, security warning about the client password digest.
  - Upstream declares GPL-3.0. No upstream Go source was copied.
- `koreader/koreader-sync-server`, revision `46ef6b84a3930b3f0265efac74899f2256c158f1`, tree `57d0e43d371675044c73387cc15d70f02adca239`.
  - `config/routes.lua`, `config/application.lua`, `config/nginx.conf`, and `db/redis.lua` identify the original service/deployment shape.
  - Upstream notice is AGPL; no upstream Lua or configuration code was copied.

**Strengths observed**

- Small, well-defined interoperability surface.
- Bcrypt protects the client-supplied password digest at rest.
- Registration can be closed after provisioning.
- Redis maps naturally to isolated user/document keys.

**Weaknesses and danger zones considered**

- The client password digest is replayable and therefore requires TLS.
- Registration and authentication need deployment-level rate limiting; Helix does not yet claim a complete anti-abuse system.
- Server timestamps are not a vector-clock conflict model. Helix keeps KOSync compatibility separate from its structured internal progress resolver.
- Automated protocol tests do not prove compatibility with every KOReader version or device.

**Helix implementation**

- `server/internal/kosync/store.go` — protocol model and storage seam.
- `server/internal/kosync/memory.go` — deterministic development/test store, explicitly not the production server wiring.
- `server/internal/kosync/redis.go` — durable account/progress keys and atomic registration.
- `server/internal/kosync/service.go` — validation, bcrypt, dummy comparison, server timestamps.
- `server/internal/httpapi/kosync.go` — bounded Gin handlers and public error mapping.
- `server/internal/httpapi/router.go` — `/users/create`, `/users/auth`, `/syncs/progress`, `/syncs/progress/{document}`.
- `server/cmd/helix-server/main.go` — Redis-owned production wiring and closed registration default.
- Tests: `server/internal/kosync/*_test.go`, KOReader tests in `server/internal/httpapi/router_test.go`.
- Documentation: `docs/api.md`, `docs/architecture.md`, `docs/user_guide.md`, `docs/faq.md`.

**Evidence status**

Focused offline Go tests passed. Real KOReader device/firmware interoperability is not yet tested and remains non-PASS.

## OPDS Page Streaming Extension — researched, not implemented

**Upstream evidence**

- `anansi-project/opds-pse`, revision `9a082e4e254048a584df55255114dca765c246ba`, tree `7f6d4c5bf1b94d7e97d614867fb055c85125e169`.
- `v1.2.md`: namespace `http://vaemendis.net/opds-pse/ns`, stream link relation, zero-based page requests, required page count, optional width, last-read attributes, and image media-type constraints.
- Specification document is CC BY-SA; no specification text is copied into Helix code.

**Helix gap and implementation constraints**

Helix does not emit OPDS-PSE links or extract pages. A correct implementation requires a verified page count, safe archive/PDF page access, decoded-image resource limits, deterministic page ordering, optional width handling, and actual runtime tests. Adding only a link relation would be a bluff and is explicitly rejected.

## Additional identified candidates — not ported

The following remain gaps rather than hidden completed features:

- Kobo-specific synchronization and annotation translation;
- durable annotations/highlights with export;
- watch folders and durable scheduled scans;
- smart shelves and reading profiles;
- OIDC, multi-user authorization, and per-library permissions;
- OPDS-PSE page streaming;
- external renderer/converter and LCP license/status integration.

Each requires its own revision-pinned source review, license decision, implementation links, and runtime evidence before its status can change.
