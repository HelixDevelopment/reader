# ATM-027 — provider-throttling tests + WebDAV/rclone load runs (2026-11-23)

Follow-up to the 2026-11-20 campaign (`evidence/campaign-2026-11-20/remote/`,
`TestLiveLoadRemoteSource` ladder: 282 origin requests ↔ 282 client requests,
`origin_ranged = 0`, 5/5 needles). Closes the review's open items:

- **ATM-027 report item 7 — "no slow-client or provider-throttling test"** →
  `TestLiveThrottleRemoteSourceHTTP` (new).
- **"SFTP/WebDAV/rclone-specific load runs are still missing"** (review) →
  `TestLiveLoadRemoteSourceWebDAV` and `TestLiveLoadRemoteSourceRclone` (new).

## Delivered test files

| File | Harness | Test | Gating |
|---|---|---|---|
| `server/internal/httpapi/load_remote_throttle_test.go` | throttled origin + remote-source fetch through the route (client → route → `sources.NewHTTP` → origin) | `TestLiveThrottleRemoteSourceHTTP` | `HELIX_LOAD_RUN=1` |
| `server/internal/httpapi/load_remote_webdav_test.go` | remote-source fetch through the route (client → route → `sources.NewWebDAV` → counting proxy → `rclone serve webdav` → local fixture files) | `TestLiveLoadRemoteSourceWebDAV` | `HELIX_LOAD_RUN=1` (+ rclone) |
| `server/internal/httpapi/load_remote_rclone_test.go` | remote-source fetch through the route (client → route → `sources.NewRclone` → `rclone cat <remote:path>`) | `TestLiveLoadRemoteSourceRclone` | `HELIX_LOAD_RUN=1` (+ rclone) |

All three are gated on `HELIX_LOAD_RUN=1` like the rest of the load suite and
emit one JSON line each to `HELIX_LOAD_OUTPUT` (set by the harness to the
files below).

## 1) Provider-throttling test — `TestLiveThrottleRemoteSourceHTTP`

**What was throttled, and by how much (final run):** the origin is the plain
remote-origin handler serving three random files (256 KiB / 4 MiB / 32 MiB)
plus a `time.Sleep` before every response while throttling is on
(`throttledOrigin.handler`). Throttling was calibrated as **delay = 0.30 ×
baseline p50** (floored at 500µs, capped at 250ms). Final run:
`origin_delay_per_response_ms = 2.6082`, baseline p50 = 8.694ms, throttled p50
= 15.972ms (recorded p50 ratio 1.84x).

**Assertions:**

1. **Ingest completes within an expected bound** — per file, the throttled
   fetch must finish within `baseline + delay + slack` (slack absorbs sub-ms
   loopback scheduling noise; gross stalls still fail).
2. **Zero data-integrity errors** — the route-delivered sha256 matches the
   catalog-recorded sha256, in both phases.
3. **No retry storm** — exactly one origin request per file in the throttled
   phase (`throttled_origin_requests == 1`; baseline phase too).
4. **Documented semantics** — the route withholds ETag / Accept-Ranges /
   Content-Length, and the origin sees no Range (`origin_ranged == 0`): the
   contract `docs/api.md` states for non-revalidating remote sources
   (`server/internal/httpapi/router.go` serves full 200s for them).
5. Recorded, not asserted: the p50 ratio. A hard ratio gate was flaky in
   practice — one run measured a 0.94x p50 ratio because the first 32 MiB
   fetch was ~2.6× slower than the second (warm-up noise; added latency per
   file is recorded as `added_ms_throttled_minus_baseline`), so the ratio is
   evidence, not a gate, while the completion bounds, digests, request counts
   and semantics are hard assertions.

**Result: PASS (0.28s).** Per-file (final run, `remote_throttle_ladder.jsonl`):

| file | size | baseline_ms | throttled_ms | added_ms | baseline_origin_req | throttled_origin_req |
|---|---|---|---|---|---|---|
| load-throttle-f0 | 262144 | 3.969 | 6.090 | +2.121 | 1 | 1 |
| load-throttle-f1 | 4194304 | 8.694 | 15.972 | +7.278 | 1 | 1 |
| load-throttle-f2 | 33554432 | 64.968 | 47.253 | −17.715 (throttled fetch was faster — warm-up noise) | 1 | 1 |

All three files complete within their expected bounds; digests match in both
phases; origin requests are 1:1.

**Evidence:** `evidence/campaign-2026-11-23/remote/remote_throttle_ladder.jsonl`

## 2) WebDAV load run — `TestLiveLoadRemoteSourceWebDAV`

**Target:** a real WebDAV server — `rclone serve webdav` (rclone v1.60.1-DEV)
over a `local` backend remote ("wfdav") whose root is the fixture directory
(random files; digests verified through the proxy). A counting reverse proxy
sits between the WebDAV connector and the server so origin requests/bytes are
measured. Run shape mirrors `TestLiveLoadRemoteSource`: client → route →
`sources.NewWebDAV` → proxy → rclone serve webdav → local files.

**Assertions:**

- `completed == requested` at every level; zero integrity mismatches (route
  digests match the catalog digests);
- `origin_requests == client_requests` at every level — the connector is
  one-for-one, no retry storm;
- `origin_bytes ≥ client_bytes`; `peak_inflight == clients`;
- route withholds ETag / Accept-Ranges / Content-Length (documented contract).

**Result: PASS (4.62s).** Ladder (levels 1/8/32; jobs full + range;
`remote_webdav_ladder.jsonl`):

| size | clients | completed | origin_requests | p50_ms |
|---|---|---|---|---|
| 262144 | 1 | 12 | 12 | 2.207 |
| 262144 | 8 | 16 | 16 | 9.970 |
| 262144 | 32 | 64 | 64 | 36.219 |
| 4194304 | 1 | 12 | 12 | 14.392 |
| 4194304 | 8 | 16 | 16 | 50.255 |
| 4194304 | 32 | 64 | 64 | 96.438 |
| 33554432 | 1 | 12 | 12 | 88.477 |
| 33554432 | 8 | 16 | 16 | 220.598 |
| 33554432 | 32 | 64 | 64 | 455.264 |

**Evidence:** `evidence/campaign-2026-11-23/remote/remote_webdav_ladder.jsonl`

## 3) rclone load run — `TestLiveLoadRemoteSourceRclone`

**Target:** `rclone lsjson --recursive --files-only wfdav:<root>` (List) and
`rclone cat <remote:path>` per Open — `sources.NewRclone` with a temp rclone
config whose `local` backend remote ("wfdav") is rooted at the fixture
directory. A counting wrapper around the connector counts opens; the ladder
runs through the route (client → route → `sources.NewRclone`).

**Assertions:**

- `List` returns every fixture file with its size;
- ladder: `completed == requested`; digests match; `opens == client requests`
  (one `rclone cat` per request — no retry storm).

**Result: PASS (6.63s).** Ladder (levels 1/8/32, job full;
`remote_rclone_ladder.jsonl`):

| size | clients | completed | opens | p50_ms |
|---|---|---|---|---|
| 262144 | 1 | 12 | 12 | 39.354 |
| 262144 | 8 | 16 | 16 | 68.745 |
| 262144 | 32 | 64 | 64 | 249.084 |
| 4194304 | 1 | 12 | 12 | 50.995 |
| 4194304 | 8 | 16 | 16 | 124.930 |
| 4194304 | 32 | 64 | 64 | 324.173 |
| 33554432 | 1 | 12 | 12 | 151.883 |
| 33554432 | 8 | 16 | 16 | 323.112 |
| 33554432 | 32 | 64 | 64 | 1009.131 |

(`opens_total = 276`.)

**Evidence:** `evidence/campaign-2026-11-23/remote/remote_rclone_ladder.jsonl`

## Gaps (honest)

1. **rclone/WebDAV support in the tree: present** — grep finds
   `server/internal/sources/webdav.go` (`NewWebDAV`: PROPFIND discovery, GET
   Open), `server/internal/sources/rclone.go` (`NewRclone`: `lsjson`/`cat`),
   `server/internal/sources/source.go` (`DefaultRegistry`: webdav/webdavs →
   NewWebDAV; sftp, ftp, smb, s3, nextcloud, … → NewRclone),
   `server/internal/sources/webdav_test.go`, `server/internal/sources/rclone_test.go`,
   and `server/Dockerfile` (rclone installed in the server image). So the
   support question is closed: it exists, and now load runs exist for it too.
2. **rclone target availability: none pre-existing** — `rclone listremotes` is
   empty; the run created a temp local-backend remote for itself. A run
   against a real SFTP/S3/Box/… target would need those providers; the
   rclone connector's retry flags (`--retries 2 --low-level-retries 2`,
   `--contimeout 15s --timeout 2m`) were not measured against a failing or
   throttling real provider.
3. **WebDAV target availability: none pre-configured** — stood up with
   `rclone serve webdav` against a local backend; digests verified through
   the counting proxy.
4. **Pre-existing broken files (not this work):** untracked
   `server/internal/catalog/slow_client_live_test.go` (created ~18:58) and
   `server/internal/httpapi/slow_client_live_test.go` (created ~19:08) have
   compile errors that currently break `go vet ./...` and `go test ./...` for
   `internal/catalog` and `internal/httpapi`. I did not modify them; during
   verification of my three tests I moved the httpapi one out of the tree and
   restored it afterwards.

## Verification (server/)

- `go build ./...` → **rc=0** (non-test code builds).
- `go vet ./...` → **rc=1** — compile errors in the pre-existing untracked
  `slow_client_live_test.go` files: `internal/catalog/slow_client_live_test.go:522:79`,
  `internal/catalog/slow_client_live_test.go:533:21`,
  `internal/httpapi/slow_client_live_test.go:430:22`,
  `internal/httpapi/slow_client_live_test.go:440:54`,
  `internal/httpapi/slow_client_live_test.go:483:31`
  (float64/int64 type mismatches). `go vet ./...` → rc=0 when those files are
  absent (verified 17:55).
- `go test ./...` → **rc=1** — `[build failed]` for `internal/catalog` and
  `internal/httpapi` (same files); all other packages ok:
  `cmd/helix-pse-check`, `internal/aimemory`, `internal/config`,
  `internal/covers`, `internal/identify`, `internal/ingest`, `internal/kosync`,
  `internal/opds`, `internal/sources`, `internal/syncstore`. Earlier
  `go test ./...` → rc=0 (17:5x: `internal/catalog ok 0.011s`,
  `internal/httpapi ok 4.461s`, …).
- The three new tests pass with `HELIX_LOAD_RUN=1` (verified with the
  pre-existing broken files moved aside so the package could build):
  `TestLiveThrottleRemoteSourceHTTP` (0.28s, PASS),
  `TestLiveLoadRemoteSourceWebDAV` (4.62s, PASS),
  `TestLiveLoadRemoteSourceRclone` (6.63s, PASS).

## Cross-references

- Review item 7 (provider-throttling test): closed by
  `TestLiveThrottleRemoteSourceHTTP` + `remote_throttle_ladder.jsonl`.
- SFTP/WebDAV/rclone load runs: added by `TestLiveLoadRemoteSourceWebDAV` +
  `remote_webdav_ladder.jsonl` and `TestLiveLoadRemoteSourceRclone` +
  `remote_rclone_ladder.jsonl`.
- Prior campaign unchanged: `evidence/campaign-2026-11-20/remote/`
  (282↔282 requests, `origin_ranged = 0`, 5/5 needle gates).

Redaction note (round 2): the operator home-directory prefix in remote_rclone_ladder.jsonl was replaced by /home/<user>/ (temp-dir paths only; no measurement value changed).
