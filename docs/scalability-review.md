# Backend and API scalability review

## Verdict

**Enterprise-scale qualification is not complete.** Functional, race, live-service, transport, and recovery tests establish correctness in the recorded environments; they do not establish high-cardinality catalog performance, horizontal scalability, durable job processing, or production service-level objectives. ATM-027 remains open.

This review covers the current source tree. Limits below are code-derived observations, not benchmark measurements. No unsupported capacity number is claimed.

## Service-by-service findings

| Surface | Existing controls | Scalability risks and open work | Current evidence boundary |
|---|---|---|---|
| Gin HTTP API | Request-size limits on JSON mutation endpoints; bounded page size; security headers; content negotiation; request cancellation | No measured requests-per-second, latency percentile, memory, or saturation envelope; no distributed rate limiter or per-tenant fairness layer | Unit, race, live API, and transport correctness only |
| HTTP content delivery | Live revalidation; ranges; single-descriptor local reads; streaming; unknown-size fallback; connection/header timeouts in sources | Strong scan-time hashes still require ingest work; remote content can consume upstream bandwidth; no production CDN/cache qualification | Live content, restart, transport, and raw keepalive tests |
| Scan jobs | Two active scans, 64 pending jobs, 256 retained jobs, cancellation, panic containment, graceful wait group | Jobs and history are process-local; restart loses queue/history; duplicate prevention is per process; no distributed scheduling or durable retry policy | Deterministic job tests and live scans, not multi-node behavior |
| Ingest pipeline | Bounded parser inputs, content identity, source reconciliation, deterministic metadata precedence | Whole-library scans can repeatedly parse/hash large files; no measured incremental-scan throughput or backpressure across remote sources | Unit, integration, identity regression, and live scans |
| Memory catalog | Mutex-protected repository with shared validation | Single-process and volatile; unsuitable as a durable or horizontally shared production catalog | Used explicitly for evaluation/tests, not represented as durable |
| PostgreSQL catalog | Pool caps (20 open/5 idle), 30-minute connection lifetime, transactional migrations with advisory lock, indexed foreign keys and updated-at ordering | A global advisory transaction lock serializes merges and reconciliation; upsert deletes/reinserts editions; search uses offset pagination and `%text%` predicates; each result work causes child-query fan-out; no large-catalog query-plan or lock-wait evidence | Real PostgreSQL correctness tests; no enterprise load result |
| PostgreSQL AI memory | Separate pool caps (10 open/3 idle), dimension checks, bounded search limit, HNSW cosine index, expiry index | Pool sizes are fixed rather than deployment-configurable; no vector-cardinality/recall/latency benchmark; model and embedding dimensions constrain migration | Real pgvector tests with small fixtures |
| Redis KOSync | Optimistic WATCH/MULTI/EXEC, monotonic progress, bounded payloads | No Redis failover, cluster, persistence, or contention-throughput qualification; availability inherits Redis deployment | Real Redis integration tests |
| Internal sync store | PostgreSQL-backed progress logic and tests | No multi-region conflict, retention, or high-contention benchmark; no claimed exactly-once cross-system semantics | Correctness tests only |
| OPDS and OpenSearch | Pagination bounds, acquisition maps, atomic replacement, deterministic public keys | Feed generation inherits catalog query fan-out; large feeds and third-party client behavior are unmeasured; OPDS-PSE remains unimplemented | API tests, not large-feed or external-client certification |
| Remote source connectors | Context cancellation, connection/header timeouts, redirect limits, SSRF dial validation, bounded XML/JSON reads | Long streams deliberately have no total client timeout; deployments need cancellation and bandwidth policy; provider-specific rate limits and retries are not globally normalized | Connector tests; many real providers remain unrun |
| Identification and covers | Provider response-size limits and client timeouts; deterministic evidence precedence | External API quotas, cache hit ratio, thundering-herd behavior, and provider outage budgets lack production measurements | Unit/provider-contract behavior; no production quota result |
| HTTP/1.1, HTTP/2, HTTP/3 | TLS floors, Alt-Svc, Brotli/gzip negotiation, safe identity fallback | Protocol correctness does not measure slow-client exhaustion, QUIC connection capacity, or load-balancer compatibility | Five-check live transport verifier, repeated after compression fix |
| Web prototype | Production build, responsive viewport checks, explicit sample-data disclosure | No backend-connected client workload, virtualized large-catalog measurement, offline storage pressure, or low-end-device profile | 64 browser checks; no native or low-end hardware claim |
| Deployment | Multi-stage non-root image recipe, compose health checks, loopback-published dependencies, secret-injected environment | No actual container cold start/restart, orchestration, horizontal replica, migration rollback, backup scheduling, or autoscaling qualification | Configuration exists; runtime deployment evidence does not |

## Highest-priority engineering actions

1. Add repeatable large-catalog fixtures and record PostgreSQL query plans, lock waits, latency percentiles, memory, and throughput before changing identity-locking semantics.
2. Replace whole-work delete/reinsert and per-work child loading with measured set-based writes and batched reads while preserving provenance and atomic identity behavior.
3. Add keyset pagination for stable large-result traversal; keep offset compatibility only where required.
4. Introduce durable scan scheduling before promising replica-safe or restart-safe background work.
5. Make pool and queue limits deployment-configurable with validated bounds and safe defaults; coordinate them with PostgreSQL/Redis capacity rather than increasing them blindly.
6. Run multi-client content, OPDS, KOSync, vector-search, and remote-source load tests, including slow clients, cancellation, provider throttling, and service restart.
7. Test actual containers and orchestration, then define resource requests, limits, disruption behavior, backup objectives, and rollback procedures from observed data.

## Required release gate

ATM-027 can close only after representative workloads and target service-level objectives are defined, the above hotspots are addressed or explicitly bounded, and repeated machine-readable load results pass in the intended deployment topology. Current functional PASS artifacts must not be cited as enterprise-scale proof.
