# ATM-024 — Containerized build / deploy / restart / upgrade / rollback qualification

**Date:** 2026-11-20T17:35–17:50Z · **Host:** Ubuntu, podman 5.7.0 (rootless), podman-compose 1.5.0, **no docker daemon**
**Prior state (docs/scalability-review.md:27):** deployment configuration existed; runtime deployment evidence did not. This section closes that gap with executed evidence.

## Outcome

All seven lifecycle stages PASS on podman/podman-compose. One real build-blocking defect was found and it was **not** in deploy/: the first `podman build` failed because `server/internal/httpapi/jobs.go:74` used `slog` without importing `log/slog` (tree-level compile break under concurrent modification; the owning agent's import fix landed minutes later and the rebuild succeeded).

| Stage | Result | Proof (this directory) |
|---|---|---|
| Build server image v1 | PASS (attempt 2) | `build.log` — image `b567c4fb27c7`, label `revision=v1-coldstart` |
| Cold start → healthy | PASS | `cold-start.log` — fresh volumes, `status=healthy` in ~8s, restart_count=0 |
| DEFECT-001 on empty schema (in-container) | PASS | `cold-start.log` — `helix_schema_migrations` = exactly `version=1 rows=1`, 6 tables, no duplicate-key; the image carries the fixed migration |
| API endpoints (catalog CRUD, OPDS, health) | PASS | `cold-start.log` — /healthz, /readyz 200; source create 201; scan succeeded; works read via API+DB; OPDS 1.2 atom + 2.0 opds+json 200; source delete 204; unauthenticated 401s |
| Container restart preserves data | PASS | `restart.log` — same work id after `podman restart`, no rescan, no re-migration |
| Upgrade v1 → v2 | PASS | `upgrade.log` — running image `c1ddd5654707` (label `v2-upgrade`), healthy, data intact |
| Rollback v2 → v1 | PASS | `rollback.log` — running image back to `b567c4fb27c7`, healthy, data intact; teardown: 0 residual containers/volumes/networks, `helixqual-pg`/`helixqual-redis` untouched |

## Isolation actually used

- Project `helixqual-deploy`; containers `helixqual-deploy_{postgres,redis,helix}_1` (podman-compose underscore naming; docker-compose would hyphenate).
- Ports `127.0.0.1:39080/tcp+udp` only (host 8080 occupied; 5432/6379/34320/34379 avoided).
- Dedicated project network + fresh named volumes; removed on teardown. Qualification images removed afterwards.

## podman-compose vs docker-compose divergences observed (recorded, not assumed)

1. `up -d` recreated **all three** services on the v1→v2 upgrade and again on rollback; docker-compose recreates only the changed service. No data loss because state lives in named volumes, but expect database/connection churn on single-service upgrades under podman-compose.
2. Top-level `name:` beat `-p helixqual-deploy` in podman-compose 1.5.0; the compose file now interpolates `${COMPOSE_PROJECT_NAME:-helix}` so both tools honor an isolated project prefix.
3. `podman-compose config` prints fully resolved secret values to stdout — treat its output as sensitive.
4. `podman-compose down --rmi local` did **not** remove the localhost-tagged built images; explicit `podman rmi` was needed.
5. `depends_on: condition: service_healthy` behaved correctly in practice (helix never crash-looped waiting for Postgres; restart_count stayed 0 across cold start).
6. No docker-compose run was possible on this host (docker not installed); parity claims beyond the file semantics themselves remain untested.

## Deploy-tree changes made for this qualification

- `deploy/docker-compose.yml`: parameterized project name, explicit `image:` on the helix service (`${HELIX_IMAGE:-helix-helix:latest}`) to make upgrade/rollback tag-driven, `HELIX_PUBLISH_PORT` for the loopback publish (default unchanged at 8080), and a `/healthz` healthcheck for the helix service.
- `server/Dockerfile`: `ARG HELIX_BUILD_REVISION` + `org.opencontainers.image.revision` label so every container is traceable to its build revision.
- `.gitignore`: re-include `evidence/**/*.log` (campaign artifacts were being swallowed by the global `*.log` ignore).

## Sanitization

Credentials existed only in a gitignored scratch env file (never printed). All artifacts were machine-checked: no secret value, no DSN userinfo, no token appears in any file in this directory. Scratch env file deleted after the run.

---

# Second pass (2026-11-20T19:28–19:34Z) — distribution and compose execution

The first pass left two acceptance words unearned: **distribution** had never been exercised, and **compose execution** had only been asserted at file semantics. Both are now executed evidence in `distribution.log`, produced by `run-distribution-and-compose.sh` followed by `run-digest-and-auth-probes.sh`.

| Check | Result | Proof |
|---|---|---|
| Build the shipped recipe | PASS | `podman build` 42 s, config id `ea83760a44f3`, `user=helix`, entrypoint `/usr/local/bin/helix-server` |
| Registry push → cache-free pull | PASS | pushed to `registry:2` on 127.0.0.1:15000; `/v2/_catalog` lists `helix-reader`, `/tags/list` lists `v1`; **both local tags force-deleted (0 matching local images)**; pull then reproduces `Docker-Content-Digest: sha256:ccfab78d…6d90` and the same config id as the build |
| Run the pulled artifact | PASS | `/healthz` 200; `/v1/works` cursor body; `/opds/1.2` atom navigation; `/opds/2.0` `application/opds+json`; `id -un` = `helix` |
| podman-compose execution | PASS | all three services healthy; `/readyz` `{"status":"ready"}`; `psql` **inside the stack** shows `helix_schema_migrations=1` and the `works` table, so the app consumed `HELIX_POSTGRES_DSN`; `redis-cli ping` PONG; 3 project volumes |
| Restart / upgrade / rollback under compose | PASS | `restart` kept readiness and the migration row; `HELIX_IMAGE=:v2` then `:v1` both served 200 |
| Credential gating needles | PASS | 401 (none) / 401 (wrong) / 200 (right) on four read+OPDS routes; 401 / 401 / 200 on `GET /v1/scans`; `DELETE /v1/works/<unknown>` 401 → 404 with token; `POST /users/create` 503 with registration disabled |
| Teardown | PASS | 0 residual containers / images / volumes for `helixqual-dist`; `helixqual-pg` and `helixqual-redis` still running |
| **docker-compose CLI parity** | **NOT RUN** | docker is not installed on this host. No claim is made about docker-compose runtime behavior. |

## What the first probes got wrong, and why it is recorded rather than deleted

1. `podman push --remove-signature` does not exist in podman 5.7, so the first push never happened and the pull failed with `manifest unknown`. The log of that run was truncated before the authoritative run; the flag is gone from the script.
2. podman-compose names containers `project_service_N`; the first script exec'd `project-service-N` and reported a bogus "migrations" line that was actually a shell error string. `compose_migrations=1` in the final log is `psql -Atc` output from inside the stack's Postgres.
3. The first script probed `/opds/1.2/navigation`, which is not a route (`/opds/1.2` is), and expected `/v1/works` to reject an *admin bearer* token. The read group is guarded by `Basic(HELIX_OPDS_USERNAME, …)`; with that unset, `middleware.go:61-64` calls `c.Next()`, so 200 was correct behavior, not a leak. The rejection needle was re-cut with OPDS credentials set (row above).
4. Equality is asserted against the **registry's** manifest digest and the config id, not `podman inspect --format '{{.Digest}}'`, which reports a different local representation.

## Residual against acceptance

`acceptance` names cold start, health, restart, upgrade, rollback and distribution in actual container runtimes — all six are now executed under podman. Two things stay open by observation, not by omission: docker-compose CLI runtime parity requires a Docker-equipped host, and push/pull was exercised against a local `registry:2`, not a hosted registry with authentication, signing or attestation.
