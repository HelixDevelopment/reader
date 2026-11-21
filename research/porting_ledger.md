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

## Revision-pinned checkout research round (2026-11-20) — per-repository decisions

Evidence layout: full non-shallow clones (fsck clean) under `/tmp/helix_research_clones/` (five primary) and `/tmp/helix_extra_clones/` (supplementary), each checked out detached at the exact `research/analysis_summary.json` pinned revision except where noted; file ledgers recomputed from `git ls-tree -r HEAD` matched the preserved per-repository `tracked_files` counts before this round flipped statuses. Reports live in `research/reports/`. `research/analysis_summary.json` was updated in place (schema 2, field names unchanged): 111 analyses retained, checkout rows 56→97 valid, coverage counts raised accordingly; no other field was altered.

| Repository | Revision | Decision | License basis | Report |
|---|---|---|---|---|
| anx-reader | `0c7fc1ad4551f7dbbeba6a6468f69d830957fa2a` | PORT-WORTHY: webview/foliate-js bridge shapes, TXT split rules, WebDAV sync conflict model | MIT | `reports/anx-reader.md` |
| bookorbit (backend `bookorbit-bookorbit` + client `bookorbit`) | `f2a7a111881899e7c6cb233a0ec2d8a9c25235c6`; client pin `99aa7dce705986eac29a8f73aa65331de5f50680` present in same clone (client row `matches_analysis=false` by design) | RESEARCHED, BLOCKED-BY-LICENSE: learn API-contract and KOReader-plugin-protocol patterns only | **AGPL-3.0-only WITH Section 7 additional terms** (`ADDITIONAL_TERMS.md`: mandatory "Powered by BookOrbit" UI attribution, in-UI corresponding-source offer, modified-version notice, trademark clause, preservation) — materially beyond plain AGPL | `reports/bookorbit.md` |
| calibre-web-automated | `4fc2c4459965efb1b8ff43ea7849c2f47a755d9c` | LEARN ONLY: ingest-worker and appliance-auth patterns | GPL-3.0-or-later (SPDX headers; in-repo Calibre-Web fork) | `reports/calibre-web-automated.md` |
| sumatrapdf | `a30e4d6429419361132c23c797877f242092ea0e` | LEARN ONLY: renderer/damage-tracking and vendored-engine seam designs | (A)GPLv3 — AGPL via MuPDF linkage; BSD islands per AUTHORS | `reports/sumatrapdf.md` |
| thorium-reader | `1f3e03448eedf352238c0fe5fe4aa765b9e9a1f2` | PORT-WORTHY: OPDS progression schema, r2-opds field mapping | BSD-3-Clause (EDRLab) | `reports/thorium-reader.md` |
| thorium-web | `8d49c2b116e93ca50d4ccfdcb3226efde7bc49c8` | LEARN/reference for progression contract on web | BSD-3-Clause | `reports/thorium-web.md` |
| koodo-reader | pinned `fe1abf4b8960f4e48a2a948659c3bb816a6e2606` **no longer reachable upstream**; checkout at `4eadb2ec9fe98abd02e2f31712b95f8c0306e298`, row `matches_analysis=false` | LEARN ONLY | AGPL-3.0 | `reports/koodo-reader.md` |
| foliate | `67b6676d3f936c5edea91d4d903385ef39dd25c0` | LEARN ONLY from this tree (use MIT foliate-js distribution instead) | GPL-3.0 at pin | `reports/foliate.md` |
| bookokrat | `9c02ee3b3a0f86791b3917cb1a93436d92a8603e` | LEARN ONLY: ingest image normalization for e-ink | AGPL-3.0-only | `reports/bookokrat.md` |
| book-story | `464891ec8cb00196d10b6dbaba3b7409c8fd27a8` | LEARN ONLY: native typesetting/pagination design | GPL-3.0 | `reports/book-story.md` |
| stump (`stumpapp-stump`+`stump`) | `d0f51c8e6da74d584a2a4cda8d9db02337de73ca` | PORT-WORTHY: OPDS assembly + scan/resolve pipeline boundaries | MIT | `reports/stump.md` |
| colophon | `3676404ab253c7e0ef78fcda973a27424056114d` | IDEA PORT: plain-HTTP e-reader UX hardening list | MIT | `reports/colophon.md` |
| go-storage | `1d5ce66ccc0132bcb6d8a224e93a0c2d3963a460` | PATTERN/DEPENDENCY-CANDIDATE: capability negotiation, endpoint parsing | Apache-2.0 | `reports/go-storage.md` |
| cops (`seblucas-cops`+`cops`) | `7828cc600e904941583227071b8f4efd3b9385fb` | LEARN ONLY: OPDS 1.2 client-compat quirks | GPL-2.0 | `reports/cops.md` |
| grimmory (`grimmory-tools-grimmory`+`grimmory`) | `021cb138dc7f19583142c55fd5070201c40b403b` | LEARN ONLY: smart filters, RBAC | AGPL-3.0 | `reports/grimmory.md` |
| audiobookshelf (`advplyr-audiobookshelf`+`audiobookshelf`) | `edb1f7efc92ecacdef51aafb4d9d5054ce10604f` | LEARN ONLY: provider-spec extensibility, scan fingerprinting | GPL-3.0 | `reports/audiobookshelf.md` |
| booklore | `01b3407528a6ce00d064dbb9aa3f69b72d82641b` | LEARN ONLY: metadata-provider orchestration | AGPL-3.0 | `reports/booklore.md` |
| openlib | `fd779d7dd8dee1ab07aaf189f406dc00275c6321` | LEARN ONLY: mobile client requirements checklist | GPL-3.0 | `reports/openlib.md` |
| termepub | `70a1886f691bcfd87d05829a2839638b4ee24969` | PORT-WORTHY: Go text pagination helpers | MIT | `reports/termepub.md` |
| komga (`gotson-komga`+`komga`) | `023ec905c10a35bbb8e29095420d266cd3adad60` | PORT-WORTHY (re-verify license on re-pin): read-progress + OPDS pagination semantics | MIT text at pin (historically MPL-2.0 — pin governs) | `reports/komga.md` |
| kavita (`kareadita-kavita`+`kavita`) | `d549d5b1609086c8190b9bee6eb48ebfed71e1df` | LEARN ONLY: smart-filter expression, age-restriction policy model | GPL-3.0 | `reports/kavita.md` |
| readest | `2f260ee10ff3500f77a43cfbb24254b7ac422aa7` | LEARN ONLY: position-merge strategy, dual-engine rendering seam | AGPL-3.0 | `reports/readest.md` |
| calibre-web | `6e840acbda8d62fdd159bf3855c01872d629a914` | LEARN ONLY: Calibre-DB compatibility edges | GPL-3.0 text at pin (recheck on re-pin) | `reports/calibre-web.md` |
| lanraragi | `adfe9e75cbb8dd4db0fef2c2e35698d557c5a59b` | LEARN + selective (plugin/queue decomposition) | MIT text at pin (historically GPL — pin governs) | `reports/lanraragi.md` |
| yomikiru | `3087618628734ef7e986fcd26b331a80dd4f3b57` | LEARN + selective: keybinding schema | MIT | `reports/yomikiru.md` |
| diy-esp32-epub-reader | `2c2f57fdd7e2a788d14a0bcb26b9e845a47aac42` | PROTOCOL PORT: browser-to-e-ink paging contract | MIT | `reports/diy-esp32-epub-reader.md` |
| diy-esp32s3-epub-reader | `3b35576127dd2e3b27a46336e301c7b4793f95b7` | PROTOCOL PORT: MCU variant checklist | MIT | `reports/diy-esp32s3-epub-reader.md` |
| chytanka | `39b23eeb2abc0b9205695fd7b773ae616e2c3db5` | **BLOCKED — no license file at pin** | all rights reserved (default) | `reports/chytanka.md` |
| polka | `01c2712a771fc72b7eced5887eea68e6bb1876a0` | LEARN ONLY | MIT | `reports/polka.md` |
| quill | `790ffd6fb73318a76888d66cb32c88b2477f09dd` | LEARN ONLY: e-ink repaint UX | GPL-3.0 | `reports/quill.md` |
| tidepage | `7e9caeb0ce4c7eb9922c5abf59cade2d86617004` | SELECTIVE PORT: pagination state machine | MIT | `reports/tidepage.md` |
| read-cat | `da008dced50c56a3fd58f3a7240d811f56e88ce4` | LEARN ONLY | GPL-3.0 | `reports/read-cat.md` |
| myne | `7ec1fdb19ebf19de13d7b61bcb52b3ce4878a579` | PORT-WITH-ATTRIBUTION: catalog-provider abstraction | Apache-2.0 | `reports/myne.md` |
| mitsu | `492d0cc71570936e195b8244efcc70102b5e5209` | LEARN ONLY: CI smoke-test pattern | MIT | `reports/mitsu.md` |
| codexa | `083dffe43e2fcd31a1222ea09196909f78e7fc2c` | **BLOCKED — no license file at pin** | all rights reserved (default) | `reports/codexa.md` |
| candela | `26f079bc6d3dda3e30b8a6fc511990084cc0c838` | LEARN + adopt CI action-SHA-pinning process | GPL-3.0 | `reports/candela.md` |
| read-pico-reader | `2f1ef29d446b66ec1d0bf1afa39f1caa8622bbc8` | **PROHIBITED until custom license scope cleared** | proprietary-adjacent policy doc, not an OSI license | `reports/read-pico-reader.md` |
| quire | `4cbb7dd401dae870e71468665b9bd3e66b4a583e` | **BLOCKED — no license file at pin** | all rights reserved (default) | `reports/quire.md` |
| jellybook | `505684a96fd2529cf17697a97824064b74671297` | LEARN ONLY | MIT | `reports/jellybook.md` |
| readany | `fce44689e1377df1ae99a35e8e71d5e8b3a639d1` | LEARN ONLY: format-normalization plan discipline | GPL-3.0 | `reports/readany.md` |

## Remaining UNAVAILABLE targets — honest status and evidence (2026-11-20)

14 rows across 14 unique URLs remain `current_checkout_status=unavailable` in `research/analysis_summary.json`. None is unreachable; they are unpinned large repositories deferred by clone-time budget, plus one clone that repeatedly failed. `git ls-remote <url> HEAD` on 2026-11-20 (exact output preserved):

```
https://github.com/0x1abin/crossmux                     4ac84e3d7a2339f15a2285b9e60ae880e588af2f  HEAD
https://github.com/aniyomiorg/aniyomi                   97414446b8a95994c72dd33c41c971a89d4d25b8  HEAD
https://github.com/binarynonsense/comic-book-reader     1dec0803273fcbf050540a076e79ddcc2c61a651  HEAD
https://github.com/crosspoint-reader/crosspoint-reader  08d9d68de5de269dc55e510cc69f55d30e8fd237  HEAD
https://github.com/foobnix/librerareader                10f8803e74d403744008fd99ec727f36bc99af03  HEAD
https://github.com/hankscafe/omnibus                    a1dbfe94c662d94cc92e80d29288c38668f7ebfa  HEAD
https://github.com/hthoene/paperpaw                     2e9f4bebcb96300d37471828f8f3051f0dfa8629  HEAD
https://github.com/kodjodevf/mangayomi                  0ad144b744b4bf8359f22207409f78335cf7e359  HEAD
https://github.com/kovidgoyal/calibre                   dc441a2a132169bf0924468b79aeff22d611bcdb  HEAD
https://github.com/mihonapp/mihon                       f64d82b94218b42410c36e6c85da818574865a28  HEAD
https://github.com/ollm/opencomic                       c7ae2a4d19e31bf76a4df05f80aabd025666da9c  HEAD
https://github.com/ryacub/rayniyomi                     afd3608b7d676d0c122a9c86135f5e0c9ba3f046  HEAD
https://github.com/smoores-dev/storyteller              996aca77356067a228562b843b71d7aa1cd66ecd  HEAD
https://github.com/turgu1/epub-inkplate                 7360a3fa966d2048202a470d9d1a93362b1337d6  HEAD
```

Storyteller is the only one with a *failed* clone attempt to report (repository ≈1.0 GB per GitHub API; three full-clone attempts exceeded the per-clone timeout; no shallow/partial clone was accepted as evidence and none was kept):

```
$ timeout 420 git clone https://github.com/smoores-dev/storyteller smoores-dev-storyteller
FAIL smoores-dev-storyteller rc=124          # (reproduced identically on retry)
  | Cloning into 'smoores-dev-storyteller'...
$ timeout 560 ./clone_one.sh smoores-dev-storyteller https://github.com/smoores-dev/storyteller
FAIL smoores-dev-storyteller rc=124
  | Cloning into 'smoores-dev-storyteller'...
```

One further honesty note: koodo-reader's pinned analysis revision `fe1abf4b…` was rewritten out of upstream history; its row is `valid` at a later HEAD with `current_checkout_matches_analysis=false` and its preserved ledger is flagged non-corroborable in `reports/koodo-reader.md`.
