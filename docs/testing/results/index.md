# Helix test and qualification results

This index is generated from preserved artifacts. A suite PASS is scoped to that suite; it is not a production-readiness or final-release claim. Historical failures remain visible.

**Release qualification: NOT READY.**

## Status rollup

- FAIL: 3
- NOT DELIVERED: 1
- NOT RUN: 6
- PARTIAL: 2
- PASS: 15

| Result | Category | Status | Outcome | Evidence |
|---|---|---|---|---|
| core-unit-integration | Core | **PASS** | Complete core Go test log. | `evidence/core/final-go-test.log` |
| core-race | Core | **PASS** | Core race-detector campaign. | `evidence/core/final-go-race.log` |
| core-vet | Core | **PASS** | Core static vet campaign. | `evidence/core/final-go-vet.log` |
| core-fuzz | Fuzz | **PASS** | Six core format fuzz targets. | `evidence/core/final-go-fuzz-formats.log`<br>`evidence/core/final-go-fuzz-cbz.log`<br>`evidence/core/final-go-fuzz-epub.log`<br>`evidence/core/final-go-fuzz-fb2.log`<br>`evidence/core/final-go-fuzz-mobi.log`<br>`evidence/core/final-go-fuzz-txt.log` |
| server-unit-integration | Server | **PASS** | Server test and three repeat logs. | `evidence/server/final-go-test.log`<br>`evidence/server/final-repeat-1.log`<br>`evidence/server/final-repeat-2.log`<br>`evidence/server/final-repeat-3.log` |
| server-race-vet | Server | **PASS** | Server race and vet campaigns. | `evidence/server/final-go-race.log`<br>`evidence/server/final-go-vet.log` |
| server-range-fuzz | Fuzz | **PASS** | Single-range parser fuzz target. | `evidence/server/final-go-fuzz-single-range.log` |
| live-postgresql-redis | Live services | **PASS** | Real PostgreSQL, pgvector, Redis KOSync, and internal progress tests. | `evidence/server/final-go-live-services.log` |
| web-static-and-build | Web | **PASS** | Dependency dry run, lint, and production build. | `evidence/ui/final-npm-ci-dry-run.log`<br>`evidence/ui/final-npm-lint.log`<br>`evidence/ui/final-npm-build.log` |
| web-four-viewports | Web | **PASS** | 64 browser checks across four viewports; sample-only interface, not native-client evidence. | `evidence/ui/final-run.log` |
| go-mutation | Mutation | **PASS** | 21 pass / 0 fail. Twenty-one targeted mutants; original-tree hash check is included in the artifact. | `evidence/go-mutation/results.json` |
| workable-items | Governance | **PASS** | 10 pass / 0 fail. Real SQLite lifecycle, stress, crash recovery, corruption recovery, and four tracker mutations. | `evidence/workable-items/results.json` |
| live-server-post-fix | Live API | **PASS** | Three ordinary 37/37 runs and one daemon/PostgreSQL/Redis restart 38/38 run passed after the keepalive fix. | `evidence/live-server/post-fix-repeat-01/results.json`<br>`evidence/live-server/post-fix-repeat-02/results.json`<br>`evidence/live-server/post-fix-repeat-03/results.json`<br>`evidence/live-server/post-fix-daemon-restart/results.json` |
| live-server-earlier-failure | Live API | **FAIL** | Preserved pre-fix daemon-restart failure: connection reset after the identity attachment response. | `evidence/live-server-daemon-restart/results.json` |
| transport-post-fix | Transport | **PASS** | Three consecutive instrumented 5/5 HTTP/1.1, HTTP/2, HTTP/3, TLS, and Alt-Svc reruns passed. | `evidence/live-transport/post-compression-repeat-02/results.json`<br>`evidence/live-transport/post-compression-repeat-03/results.json`<br>`evidence/live-transport/post-compression-repeat-04/results.json` |
| transport-readiness-failure | Transport | **FAIL** | One post-fix readiness attempt failed without diagnostics; direct reproduction passed and the verifier was instrumented before three passing reruns. The initial failure is not explained away. | `evidence/live-transport/post-compression-failure-01/results.json` |
| backup-restore-post-fix | Recovery | **PASS** | 13 pass / 0 fail. Live backup/restore rerun after the compression change. | `evidence/backup-restore/post-compression/results.json` |
| helixqa-build | HelixQA | **FAIL** | Actual CLI build failed because declared own-org dependency checkouts are incomplete. No HelixQA test pass is claimed. | `evidence/helixqa-build-test.log`<br>`submodules/qa/helix-deps.yaml` |
| repository-research | Research | **PARTIAL** | 106/111 static analyses are indexed; five targets remain unavailable and manual architecture/license review is incomplete. | `research/analysis_summary.json` |
| enterprise-load | Scalability | **NOT RUN** | No representative enterprise load, query-plan, saturation, or multi-node qualification has run. | `docs/scalability-review.md` |
| native-clients | Clients | **NOT DELIVERED** | No native Windows, macOS, Linux, Android, or iOS client artifacts or real-device validation exist. | `docs/architecture.md` |
| device-demos | Clients | **PARTIAL** | Four browser viewports are verified; native-device and low-end-hardware interactive demonstrations are not delivered. | `evidence/ui/final-run.log` |
| container-runtime | Deployment | **NOT RUN** | Container recipes exist, but Docker/Podman cold start, restart, upgrade, and rollback have not run. | `deploy/docker-compose.yml`<br>`server/Dockerfile` |
| firebase-distribution | Deployment | **NOT RUN** | No project-bound Firebase distribution run or CLI evidence exists. | No artifact; non-run statement |
| external-interoperability | Interoperability | **NOT RUN** | No physical KOReader, external OPDS client, or deployment-specific rclone provider validation is recorded. | No artifact; non-run statement |
| helixtrack-remote-sync | Governance | **NOT RUN** | Only an offline outbox exists. Exact remote API compatibility, credentials, and push/pull receipts are unverified. | `docs/tracking/helixtrack-outbox.json`<br>`docs/workable-items.md` |
| final-archives | Release | **NOT RUN** | Fresh final ZIP and tar.gz archives are withheld while requested scope and qualification remain open. | No artifact; non-run statement |

## Interpretation

- `FAIL` entries are preserved defects or failed qualification attempts, not hidden by later passes.
- `PARTIAL` means some evidence exists but the requested scope is incomplete.
- `NOT RUN` means no qualifying execution occurred.
- `NOT DELIVERED` means the requested implementation or artifact does not exist.
- No result in this index authorizes final archives or a production-ready claim.
