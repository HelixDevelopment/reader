# Remote-source delivery harness — Phase 1 (ATM-027)

Run 2026-11-21, `TestLiveLoadRemoteSource` in `server/internal/httpapi/load_remote_test.go`,
executed as a compiled binary inside `helixqual-pg`. Payloads:
`remote_source_ladder.jsonl` (this directory), needles: `needles/needle_gate_results.json`.

## What this harness is for

`needle_delivery.sh` already gates the *local* path, where `sources.Local` revalidates and
the route may serve ranges. The remote connectors (http, webdav, opds, rclone) implement no
revalidation, so `docs/api.md` promises the opposite: a full chunked 200 with **no** ETag,
**no** `Accept-Ranges`, and **no** `Content-Length` taken from a scan-time size. Nothing
measured that promise until now, and nothing measured what it costs a reader.

The origin is an in-process `httptest.Server` with real range support that counts requests
and bytes leaving the building. The route's own response says nothing about either, so the
counters carry the claims.

## Measured

3 objects of cryptographically random bytes (256 KiB / 4 MiB / 32 MiB), levels 1/8/32,
282 origin requests for 282 client requests, **`origin_ranged = 0`, `origin_206 = 0`**.

| size | clients | req/s | MiB/s | p50 ms | p95 ms | peak in flight |
|---|---|---|---|---|---|---|
| 256 KiB | 1 / 8 / 32 | 1148 / 3572 / 3447 | 287 / 893 / 862 | 0.69 / 0.91 / 1.53 | 0.77 / 2.12 / 11.02 | 1 / 8 / 32 |
| 4 MiB | 1 / 8 / 32 | 120 / 265 / 273 | 481 / 1061 / 1092 | 5.76 / 13.52 / 60.24 | 7.92 / 25.59 / 103.50 | 1 / 8 / 32 |
| 32 MiB | 1 / 8 / 32 | 15.3 / 32.2 / 34.7 | 490 / 1032 / 1112 | 43.68 / 128.43 / 310.99 | 57.50 / 231.34 / 750.60 | 1 / 8 / 32 |

Zero integrity mismatches, zero errors, `Completed == Requested` at every level, and
`peak_inflight == clients` at every level. The 256 KiB class is request-rate-limited; the
4 MiB and 32 MiB classes reach ~1.1 GiB/s and are transfer-limited. In-process loopback:
this bounds the serving code, not any NIC.

**The cost the harness exists to make visible.** A resuming client asks for a 64 KiB window.
The origin must resend the whole object, so the amplification against the requested window is
**4× / 64× / 512×** by size class — logged as `range_amplification_vs_requested_window`.
Against the same 32 MiB class on the local path the route serves a 206 of the window:
p50 19.01 ms at 32 clients versus 310.99 ms here. That ratio is directional, not a
controlled A/B — the local run is `in_container_clean` against Postgres and a file-backed
source, this one is `in_process_loopback` against a memory catalog and an httptest origin.
What is not directional is `origin_ranged = 0`: the connector never sends a `Range` header,
so every byte of a changed book crosses the wire on every resume attempt.

## Needle gate — 5/5 fired (`needles/needle_log.txt`)

Each case must fail with the *intended* message; a mutation that passes is a hole.

| case | mutation | caught by |
|---|---|---|
| `remote_etag_lie` | route emits the scan-time ETag when `!current` | `advertised strong validator` |
| `remote_ranges_lie` | route emits `Accept-Ranges: bytes` when `!current` | `while the route cannot honour a range` |
| `remote_range_unvalidated` | range decision stops requiring live confirmation | `returned 206; this source must get a full 200` |
| `remote_double_fetch` | `HTTP.Open` silently fetches twice | `12 client requests caused 24 origin requests; the connector is not one-for-one` |
| `remote_bytes_corrupt` | one byte flipped in the connector body | `delivered sha256 …, catalog recorded …` |

`remote_range_unvalidated` initially did **not** fire on its intended assertion: the first
mutation set `sizeKnown` too, so the run died on the Content-Length check during the full GET
and never reached a range request. It passed for the wrong reason's neighbour, which is not a
pass, so it was rewritten to force only the range decision. The corrected form produces a
genuine 206 and is caught at the range assertion. Restores came from `cp` snapshots in
`/tmp/needle-remote-backup-*`, never from git; the tree was byte-identical after every run.

## What stays open in ATM-027

Unchanged by this harness: there is still no ratified SLO for delivery throughput or KOSync
latency, no recall/search-width objective for AI memory, and no webdav/rclone-specific load
run (the http connector stands in for the shared no-revalidation contract, not for their
transport behaviour). Those are owner decisions and remote-environment dependencies, not code
gaps — see `../delivery/SUMMARY.md` for the local-path and KOSync halves.
