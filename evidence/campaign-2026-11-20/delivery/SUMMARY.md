# Content delivery and KOSync load harnesses (ATM-027, Phase 1)

Run 2026-11-21, `measurement_path=in_container_clean`.

Harness: `server/internal/httpapi/load_delivery_test.go`
(`TestLiveLoadContentDelivery`, `TestLiveLoadKOSync`).
Needle gate: `scripts/needle_delivery.sh` → `needles/needle_gate_results.json`.
Raw payloads: `content_delivery_ladder.jsonl`, `kosync_ladder.jsonl`.

## Topology

The test process, the Gin router and the TCP listener run inside `helixqual-pg`;
PostgreSQL is reached on the container loopback. Redis is reached over a private
podman network (`helixqual-shared`, `helixqual-redis:6379`) added for this run by
recreating the two disposable qualification containers on a named network; their
data lives in host bind mounts under `~/.local/share/helix-qualification/` and the
catalog row counts were recorded before and after (works 1 → 1, files 1 → 1).

Consequence, stated so the number is not over-read: byte-throughput figures are
measured across an in-container loopback, so the ceiling observed here is the
memory/kernel path and the server's own serving code, **not** a physical NIC or a
reverse proxy. Nothing here is a claim about WAN delivery.

## What was measured

Fixture: 9 objects, 3 per size class (256 KiB, 4 MiB, 32 MiB), content from
`crypto/rand`, each recorded in the catalog and served through `/v1/files/:id/content`.
The ladder runs per size class so a level's number cannot be decided by which object
it happened to reach first (the first version of this harness mixed sizes and produced
a 1-client baseline that only ever touched the smallest file).

| object | clients | aggregate | req/s | p50 | p95 | p99 | peak in flight |
|---|---|---|---|---|---|---|---|
| 256 KiB | 1 | 0.915 Gbit/s | 873 | 1.06 ms | 1.26 ms | 1.31 ms | 1 |
| 256 KiB | 32 | 1.775 Gbit/s | 1693 | 11.12 ms | 37.28 ms | 49.03 ms | 32 |
| 4 MiB | 1 | 3.306 Gbit/s | 287 | 1.20 ms | 7.80 ms | 8.38 ms | 1 |
| 4 MiB | 32 | 7.655 Gbit/s | 664 | 8.30 ms | 85.71 ms | 90.89 ms | 32 |
| 32 MiB | 1 | 4.719 Gbit/s | 52.5 | 1.39 ms | 34.01 ms | 45.80 ms | 1 |
| 32 MiB | 32 | 6.955 Gbit/s | 77.4 | 19.01 ms | 670.31 ms | 683.65 ms | 32 |

Integrity: zero length mismatches, zero digest mismatches, zero recompression
(`range-gzip` jobs send `Accept-Encoding: gzip, br` and must come back byte-identical
to the unencoded window), zero transport errors, across all 240 requests. Scaling to
32 clients is 1.94× / 2.32× / 1.47× for the three classes; the 32 MiB class is already
bandwidth-bound at 1 client (4.7 Gbit/s), so its flat curve is the loopback ceiling,
not a server bottleneck.

## KOSync finding — the surface is bcrypt-bound, and the store is not

`bcrypt_p50_ms` = 55.1 ms on a 16-core container at `DefaultCost`. Every authenticated
request hashes the supplied key, so the theoretical request ceiling is
`cores × 1000 / bcrypt_ms` = **290 req/s**, which is a property of the protocol
(KOReader sends the password digest as `x-auth-key` on every call; there is no session
token to amortise against), not of the store.

Measured: PUT 14.4 → 115 → 165 req/s and GET 16.6 → 79 → 121 req/s at 1/8/32 clients,
with p50 rising 60 ms → 241 ms. That is 57 % / 42 % of the hash-bound ceiling, and the
latency growth shape is CPU queueing, not I/O: the Redis side shows no distress at all.

Contention, store level: 128 goroutines released from one barrier writing the same
document through the shipping Redis store, peak 127 in flight, **zero errors**, and the
stored percentage equals the maximum offered. 32 writers through the HTTP surface
converge the same way. No `TxFailedErr` retry-budget exhaustion was observed.

Open (owner decision, not a defect found here): whether 290 req/s at DefaultCost is an
acceptable KOSync target, or whether cost tuning / a bounded derived-comparison cache is
in scope. A cache would trade brute-force resistance for throughput, so it is not
something to add silently.

## Needle gate — 5/5 detected

Each row is a mutation applied to the tree, the harness rebuilt and run, and the
failure message asserted. Restore is from a `cp` snapshot, never from git.

| case | mutation | observed failure |
|---|---|---|
| `range_ignored` | `if false && serveRange` in `router.go` | `range GET … returned 200, not 206` |
| `wrong_window` | range seek off by one byte | `returned the right length but the wrong window` |
| `progress_rewind_serial` | monotonic guard disabled in `kosync/redis.go` | `a lower percentage rewound stored progress to 0.25, want 0.75` |
| `progress_write_lost` | first `WATCH` conflict returns success without writing | `the optimistic transaction lost a write` |
| `serialised_concurrency` | staggered worker starts in the harness | `only N requests were open at the same instant` |

Two of these caught my own assumptions before they caught product behaviour, and that is
worth recording rather than hiding:

- The first `progress_write_lost` attempt (staggered HTTP writers) **passed** the
  harness. It did not detect anything, because bcrypt spreads arrivals by tens of
  milliseconds, so the Redis-level collision rate through HTTP is near zero. The store
  level probe exists because of that miss.
- The store probe's original gate demanded all 128 writers simultaneously in flight and
  fired at 115, which was correct arithmetic and a wrong model: a sub-millisecond local
  write cannot keep 128 goroutines resident on 16 threads. The gate now requires 75 %
  overlap for the store probe and keeps the strict all-at-once bar for the HTTP ladder,
  where a 55 ms request window makes "all at once" mean what it says.

## What this does not close

No ratified SLO exists for delivery throughput or KOSync latency, so this is a measured
envelope with detection power, not a PASS for ATM-027. Remote-source connectors
(SFTP/WebDAV/HTTP-backed reads) still have no load harness.
