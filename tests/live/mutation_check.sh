#!/usr/bin/env bash
# mutation_check.sh - HRD-050 step 5: prove the live-gated tests catch defects the
# in-memory/no-env suite cannot see. Each mutant is applied to a TEMP COPY of core+server
# (the working tree is never modified), then:
#   control : package tests with NO env  -> expected PASS (mutant survives in-memory)
#   live    : package tests with PG/Redis -> expected FAIL on the named live test (mutant killed)
# Same rootless-podman isolation as run_gated_tests.sh (hrdlive-<pid>-*, 127.0.0.1, caps).
# Usage: tests/live/mutation_check.sh [OUTDIR]
set -uo pipefail
ROOT=$(cd "$(dirname "$0")/../.." && pwd); cd "$ROOT"
OUT=${1:-evidence/campaign-2026-11-23/gap-fix/hrd050/mutation}; mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd)
PFX="hrdlive-mut-$$"; CTRS=(); TMPD=$(mktemp -d)
trap 'for c in "${CTRS[@]:-}"; do [ -n "$c" ] && podman rm -f -t 2 "$c" >/dev/null 2>&1; done; rm -rf "$TMPD"' EXIT INT TERM
free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])'; }
PGPORT=$(free_port); RPORT=$(free_port); PGPW=$(head -c 12 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')
podman run -d --name "$PFX-pg" --memory 512m -p "127.0.0.1:$PGPORT:5432" -e POSTGRES_PASSWORD="$PGPW" -e POSTGRES_DB=helix_test \
  "${HELIX_LIVE_PG_IMAGE:-docker.io/pgvector/pgvector:pg16}" -c max_connections=200 -c fsync=off >/dev/null || exit 2; CTRS+=("$PFX-pg")
podman run -d --name "$PFX-redis" --memory 128m -p "127.0.0.1:$RPORT:6379" "${HELIX_LIVE_REDIS_IMAGE:-docker.io/library/redis:7-alpine}" >/dev/null || exit 2; CTRS+=("$PFX-redis")
for _ in $(seq 1 90); do podman exec "$PFX-pg" pg_isready -U postgres -d helix_test >/dev/null 2>&1 && podman exec "$PFX-redis" redis-cli ping 2>/dev/null | grep -q PONG && break; sleep 1; done
LIVEENV=("HELIX_TEST_POSTGRES_DSN=postgres://postgres:$PGPW@127.0.0.1:$PGPORT/helix_test?sslmode=disable" "HELIX_TEST_REDIS_ADDR=127.0.0.1:$RPORT")

# id|file(relative to repo)|old|new|package|killing live test
MUTANTS=(
"M1-pg-upsert-stale-editions|server/internal/catalog/postgres.go|DELETE FROM editions WHERE work_id=\$1 RETURNING id|DELETE FROM editions WHERE work_id=\$1 AND false RETURNING id|./internal/catalog|TestLivePostgresUpsertReplacesEditionSet"
"M2-redis-progress-no-merge|server/internal/syncstore/redis.go|resolved, err = coreSync.ResolveProgress(existing, incoming)|resolved, err = incoming, error(nil); _ = existing|./internal/syncstore|TestLiveRedisProgressConflictAndReconnect"
"M3-pg-identity-lock-removed|server/internal/catalog/postgres.go|SELECT pg_advisory_xact_lock(\$1, hashtext(\$2))|SELECT 1 WHERE \$1::int IS NOT NULL AND \$2::text IS NOT NULL|./internal/catalog|TestCatalogMergePreservesEveryWriterUnderIdentityLocks"
)
RC=0; : > "$OUT/mutation_summary.txt"
for m in "${MUTANTS[@]}"; do
  IFS='|' read -r id file old new pkg killer <<<"$m"
  W="$TMPD/$id"; mkdir -p "$W"; cp -r core server "$W/" 2>/dev/null; rm -rf "$W/server/internal/ingest/lcp"* 2>/dev/null
  # verify the anchor occurs exactly once, then mutate the COPY only
  python3 - "$W/$file" "$old" "$new" <<'PY' || { echo "$id: ANCHOR FAILED" | tee -a "$OUT/mutation_summary.txt"; RC=1; continue; }
import sys
p,o,n=sys.argv[1:4]; s=open(p).read()
if s.count(o)!=1: sys.exit("anchor count %d"%s.count(o))
open(p,'w').write(s.replace(o,n))
PY
  diff -u "$ROOT/$file" "$W/$file" > "$OUT/$id.diff"
  ( cd "$W/server" && go test "$pkg" -count=1 -p 1 -timeout 20m ) > "$OUT/$id.control_noenv.log" 2>&1; ctl=$?
  ( cd "$W/server" && env "${LIVEENV[@]}" go test "$pkg" -count=1 -v -p 1 -timeout 20m -run "^${killer}\$" ) > "$OUT/$id.live.log" 2>&1; live=$?
  ( cd "$ROOT/server" && env "${LIVEENV[@]}" go test "$pkg" -count=1 -v -p 1 -timeout 20m -run "^${killer}\$" ) > "$OUT/$id.live_unmutated.log" 2>&1; base=$?
  kill=$(grep -c -- "--- FAIL: $killer" "$OUT/$id.live.log")
  verdict=$([ $ctl -eq 0 ] && [ $live -ne 0 ] && [ "$kill" -ge 1 ] && [ $base -eq 0 ] && echo KILLED-ONLY-BY-LIVE || echo NOT-PROVEN)
  [ "$verdict" = KILLED-ONLY-BY-LIVE ] || RC=1
  echo "$id control(no-env,mutant)=exit$ctl live(mutant)=exit$live killer-FAIL-lines=$kill live(unmutated)=exit$base => $verdict" | tee -a "$OUT/mutation_summary.txt"
done
git -C "$ROOT" diff --stat -- server/internal/catalog/postgres.go server/internal/syncstore/redis.go | tee -a "$OUT/mutation_summary.txt"
exit $RC
