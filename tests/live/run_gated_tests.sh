#!/usr/bin/env bash
# run_gated_tests.sh - HRD-050 / GAP-015: run the WHOLE server+core suite with every
# PG/Redis/HELIX_*-gated live test actually executing, and FAIL on any SKIP that is
# not listed (reason + HRD registry item) in tests/live/allowed_skips.txt.
#
# Usage:  tests/live/run_gated_tests.sh [--no-env] [--out DIR] [--allow FILE] [--check LOG...]
#   --no-env     RED baseline: no containers/env, race pass only (gate must FAIL, ~47 skips)
#   --out DIR    evidence dir (default evidence/campaign-2026-11-23/gap-fix/hrd050)
#   --allow FILE allow-list (default tests/live/allowed_skips.txt)
#   --check LOG  skip-gate only, on existing -v log(s) (repeatable); no tests are run
#
# Phases (all with -count=1 -v -p 1):
#   A  whole server+core suite, -race, PG (pgvector) + Redis live         -> run_A_race.log
#   B1 TestLiveScaleCatalog seeds the 50k-work fixture (HELIX_SCALE_KEEP) -> run_B_load.log
#   B2 every test still skipped after A, HELIX_LOAD_RUN=1, NO -race       (perf assertions
#      are meaningless under the race detector's 5-10x slowdown)
#   B3 TestCatalogMultiProcessMerges as 4 worker OS processes + verify
#   B4 TestLCPLiveServerQualification against a freshly built helix-server
#   B5 TestDumpLCPFeeds with HELIX_OPDS_DUMP=<tmp>
# Gate: a test is "skipped" iff it SKIPs somewhere and PASSes nowhere. Every such test must
# be in the allow-list with a reason and an HRD-NNN item.
# Isolation: rootless podman only; throwaway containers hrdlive-<pid>-*, 127.0.0.1 random
# ports, memory caps (PG 512m, Redis 128m); trap removes them. Never touches the display,
# host 5432, or any container it did not create. Tests that write tracked/untracked files
# under evidence/ (hardcoded relative paths) are reverted/moved to $OUT/side_effects.
# Exit: 0 green; 1 test failure; 3 skip-gate violation; 2 setup error.
set -uo pipefail
export LC_ALL=C
ROOT=$(cd "$(dirname "$0")/../.." && pwd); cd "$ROOT"
OUT=evidence/campaign-2026-11-23/gap-fix/hrd050; ALLOW=tests/live/allowed_skips.txt
NOENV=0; CHECKS=()
while [ $# -gt 0 ]; do case $1 in
  --no-env) NOENV=1;; --out) OUT=$2; shift;; --allow) ALLOW=$2; shift;; --check) CHECKS+=("$2"); shift;;
  *) echo "unknown arg $1" >&2; exit 2;; esac; shift; done
mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd)

names_of() { grep -hE "^\s*--- $1: " "$2" 2>/dev/null | sed -E "s/^\s*--- $1: ([^ ]+).*/\1/" | sort -u; }

# skip_gate LOG... : exit 0 iff every net-skipped test is allow-listed with reason+item
skip_gate() {
  local bad=0 n r i
  cat /dev/null > "$OUT/.pass.txt"; cat /dev/null > "$OUT/.skip.txt"
  for l in "$@"; do names_of PASS "$l" >> "$OUT/.pass.txt"; names_of SKIP "$l" >> "$OUT/.skip.txt"; done
  sort -u "$OUT/.pass.txt" -o "$OUT/.pass.txt"; sort -u "$OUT/.skip.txt" -o "$OUT/.skip.txt"
  comm -23 "$OUT/.skip.txt" "$OUT/.pass.txt" > "$OUT/.netskip.txt"
  echo "skip census: $(wc -l < "$OUT/.skip.txt") distinct tests skipped somewhere, $(wc -l < "$OUT/.netskip.txt") never passed"
  while IFS='|' read -r n r i; do
    [ -z "${n// }" ] && continue; case $n in \#*) continue;; esac
    if [ -z "${r// }" ] || ! printf '%s' "$i" | grep -qE '^ *HRD-[0-9]+'; then
      echo "ALLOWLIST-MALFORMED: '$n' needs a reason and an HRD-NNN registry item"; bad=1; fi
  done < "$ALLOW"
  while read -r n; do
    [ -z "$n" ] && continue
    if ! grep -vE '^\s*#' "$ALLOW" | cut -d'|' -f1 | grep -qxF "$n"; then
      echo "UNLISTED-SKIP: $n"; bad=1; fi
  done < "$OUT/.netskip.txt"
  return $bad
}

if [ ${#CHECKS[@]} -gt 0 ]; then skip_gate "${CHECKS[@]}"; G=$?; rm -f "$OUT"/.pass.txt "$OUT"/.skip.txt "$OUT"/.netskip.txt
  [ $G -eq 0 ] && { echo "GATE GREEN"; exit 0; } || { echo "SKIP-GATE FAILED"; exit 3; }; fi

PFX="hrdlive-$$"; CTRS=(); TMPD=$(mktemp -d)
cleanup() { for ((i=${#CTRS[@]}-1;i>=0;i--)); do podman rm -f -t 2 "${CTRS[$i]}" >/dev/null 2>&1; done; rm -rf "$TMPD"; true; }
trap cleanup EXIT INT TERM
free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])'; }
hdr() { echo "# HRD-050 $1  $(date -u +%FT%TZ)  head=$(git rev-parse --short HEAD)  mode=$([ $NOENV = 1 ] && echo no-env || echo live)"; echo "# cmd: $0 $ORIG"; echo "# go: $(go version)"; }
ORIG="$*"
EVBEFORE="$TMPD/ev_before.txt"; git status --porcelain --untracked-files=all evidence > "$EVBEFORE"

if [ $NOENV = 1 ]; then SUF=noenv; else SUF=live; fi
LA="$OUT/run_A_race_$SUF.log"; LB="$OUT/run_B_load.log"
hdr "phase A (race, whole suite)" > "$LA"

if [ $NOENV = 0 ]; then
  PGPORT=$(free_port); RPORT=$(free_port); PGPW=$(head -c 12 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')
  podman run -d --name "$PFX-pg" --memory 512m -p "127.0.0.1:$PGPORT:5432" -p "127.0.0.1:$RPORT:6379" \
    -e POSTGRES_PASSWORD="$PGPW" -e POSTGRES_DB=helix_test "${HELIX_LIVE_PG_IMAGE:-docker.io/pgvector/pgvector:pg16}" \
    -c max_connections=200 -c fsync=off >/dev/null || { echo "pg start failed" >&2; exit 2; }
  CTRS+=("$PFX-pg")
  podman run -d --name "$PFX-redis" --memory 128m --network "container:$PFX-pg" \
    "${HELIX_LIVE_REDIS_IMAGE:-docker.io/library/redis:7-alpine}" >/dev/null || { echo "redis start failed" >&2; exit 2; }
  CTRS+=("$PFX-redis")
  # Test-runner container: shares the PG/Redis netns (127.0.0.1 = clean measurement path) but has its
  # own memory budget, so a 32-client x 32 MiB delivery ladder cannot be OOM-killed inside the 512m PG cap.
  podman run -d --name "$PFX-run" --memory 2g --network "container:$PFX-pg" "${HELIX_LIVE_RUNNER_IMAGE:-docker.io/library/alpine:3.22}" sleep 7200 >/dev/null || { echo "runner start failed" >&2; exit 2; }
  CTRS+=("$PFX-run")
  for _ in $(seq 1 90); do podman exec "$PFX-pg" pg_isready -U postgres -d helix_test >/dev/null 2>&1 \
     && podman exec "$PFX-redis" redis-cli ping 2>/dev/null | grep -q PONG && break; sleep 1; done
  podman exec "$PFX-pg" pg_isready -U postgres -d helix_test >/dev/null || { echo "pg not ready" >&2; exit 2; }
  export HELIX_TEST_POSTGRES_DSN="postgres://postgres:$PGPW@127.0.0.1:$PGPORT/helix_test?sslmode=disable"
  export HELIX_TEST_REDIS_ADDR="127.0.0.1:$RPORT"
  echo "# env: PG=127.0.0.1:$PGPORT (pgvector) redis=127.0.0.1:$RPORT (credentials not logged)" >> "$LA"
fi

RC=0
for mod in server core; do
  echo "=== A module $mod: go test ./... -race -count=1 -v -p 1 ===" >> "$LA"
  ( cd "$mod" && go test ./... -race -count=1 -v -p 1 -timeout 40m ) >> "$LA" 2>&1 || RC=1
done
LOGS=("$LA")

if [ $NOENV = 0 ]; then
  hdr "phase B (load harnesses, no -race)" > "$LB"
  export HELIX_LOAD_RUN=1 HELIX_LOAD_OUTPUT="$OUT/load_output.json"
  echo "=== B1 scale fixture ===" >> "$LB"
  ( cd server && HELIX_SCALE_KEEP=1 go test ./internal/catalog -count=1 -v -timeout 40m -run '^TestLiveScaleCatalog$' ) >> "$LB" 2>&1 || RC=1
  REGEX=$(comm -23 <(names_of SKIP "$LA") <(names_of PASS "$LA") | grep -v '/' | grep -vE '^(TestLiveScaleCatalog|TestCatalogMultiProcessMerges|TestLCPLiveServerQualification|TestDumpLCPFeeds)$' | paste -sd'|')
  echo "=== B2 previously-skipped: $REGEX ===" >> "$LB"
  # Host-side set: needs host binaries (rclone) or measures no PG latency.
  HOSTRE='TestLiveLoadRemoteSource|TestLiveLoadRemoteSourceRclone|TestLiveLoadRemoteSourceWebDAV|TestLiveThrottleRemoteSourceHTTP|TestLiveSlowClientCatalogSurface|TestLiveSlowClientJobSurface'
  INRE=$(tr '|' '\n' <<<"$REGEX" | grep -vxE "$HOSTRE" | grep -v '/' | paste -sd'|')
  HRE=$(tr '|' '\n' <<<"$REGEX" | grep -xE "$HOSTRE" | paste -sd'|')
  SUBRE=$(tr '|' '\n' <<<"$REGEX" | grep '/' | cut -d/ -f1 | sort -u | paste -sd'|')
  echo "=== B2h host-side (measurement_path=host_forwarded): $HRE ===" >> "$LB"
  if [ -n "$HRE" ]; then
    ( cd server && HELIX_MEASUREMENT_PATH=host_forwarded go test ./... -count=1 -v -p 1 -timeout 60m -run "^($HRE)\$" ) >> "$LB" 2>&1 || RC=1
  fi
  # Authoritative PG-latency path: test binaries run INSIDE the database container against
  # 127.0.0.1:5432 (redis shares that netns), exactly as the 2026-11-20 campaign did.
  echo "=== B2c in-container (measurement_path=in_container_clean): $INRE ===" >> "$LB"
  for pk in catalog httpapi aimemory; do
    ( cd server && CGO_ENABLED=0 go test -c -o "$TMPD/$pk.test" "./internal/$pk" ) >> "$LB" 2>&1 || RC=1
    podman exec "$PFX-run" mkdir -p "/work/server/internal/$pk" "/work/evidence" >/dev/null
    podman cp "$TMPD/$pk.test" "$PFX-run:/work/server/internal/$pk/$pk.test" >> "$LB" 2>&1
    [ -d "server/internal/$pk/testdata" ] && podman cp "server/internal/$pk/testdata" "$PFX-run:/work/server/internal/$pk/testdata" >> "$LB" 2>&1
    podman exec -w "/work/server/internal/$pk" \
      -e HELIX_TEST_POSTGRES_DSN="postgres://postgres:$PGPW@127.0.0.1:5432/helix_test?sslmode=disable" \
      -e HELIX_TEST_REDIS_ADDR=127.0.0.1:6379 -e HELIX_LOAD_RUN=1 -e HELIX_MEASUREMENT_PATH=in_container_clean \
      -e HELIX_LOAD_OUTPUT=/tmp/load_output.json \
      "$PFX-run" "./$pk.test" -test.v -test.count=1 -test.timeout 60m -test.run "^($INRE)\$" >> "$LB" 2>&1 || RC=1
  done
  podman cp "$PFX-run:/tmp/load_output.json" "$OUT/load_output_in_container.json" >/dev/null 2>&1
  podman cp "$PFX-run:/work/evidence" "$OUT/in_container_evidence" >/dev/null 2>&1
  echo "=== B3 multi-process merges (4 workers + verify) ===" >> "$LB"
  ( cd server && go test -c -o "$TMPD/catalog.test" ./internal/catalog ) >> "$LB" 2>&1 || RC=1
  RUNTAG="hrd050$$"; WP=()
  for i in 0 1 2 3; do
    ( cd server/internal/catalog && HELIX_MP_ROLE=worker HELIX_MP_INDEX=$i HELIX_MP_PROCESSES=4 HELIX_MP_OPS=60 HELIX_MP_RUN=$RUNTAG \
        "$TMPD/catalog.test" -test.v -test.count=1 -test.run '^TestCatalogMultiProcessMerges$' ) >> "$LB" 2>&1 &
    WP+=($!)
  done
  for p in "${WP[@]}"; do wait "$p" || RC=1; done
  ( cd server/internal/catalog && HELIX_MP_ROLE=verify HELIX_MP_INDEX=0 HELIX_MP_PROCESSES=4 HELIX_MP_OPS=60 HELIX_MP_RUN=$RUNTAG \
      "$TMPD/catalog.test" -test.v -test.count=1 -test.run '^TestCatalogMultiProcessMerges$' ) >> "$LB" 2>&1 || RC=1
  echo "=== B4 LCP live qualification ===" >> "$LB"
  mkdir -p "$OUT/lcp"
  ( cd server && go build -o "$TMPD/helix-server" ./cmd/helix-server && \
    HELIX_LCP_LIVE_BIN="$TMPD/helix-server" HELIX_LCP_LIVE_EVIDENCE="$OUT/lcp" \
    go test ./internal/httpapi -count=1 -v -run '^TestLCPLiveServerQualification$' ) >> "$LB" 2>&1 || RC=1
  echo "=== B5 OPDS dump ===" >> "$LB"; mkdir -p "$TMPD/dump"
  ( cd server && HELIX_OPDS_DUMP="$TMPD/dump" go test ./internal/opds -count=1 -v -run '^TestDumpLCPFeeds$' ) >> "$LB" 2>&1 || RC=1
  LOGS+=("$LB")
fi

# evidence side effects written by tests into the tracked evidence tree
mkdir -p "$OUT/side_effects"
git status --porcelain --untracked-files=all evidence | grep -vxFf "$EVBEFORE" | while read -r st path; do
  case "$path" in "${OUT#$ROOT/}"/*) continue;; esac
  [ -f "$path" ] || continue
  mkdir -p "$OUT/side_effects/$(dirname "$path")"; cp "$path" "$OUT/side_effects/$path"
  if [ "$st" = "??" ]; then rm -f "$path"; else git checkout -- "$path"; fi
done

{ for l in "${LOGS[@]}"; do echo "$(basename "$l"): PASS=$(grep -cE '^\s*--- PASS' "$l") FAIL=$(grep -cE '^\s*--- FAIL' "$l") SKIP=$(grep -cE '^\s*--- SKIP' "$l") DATA_RACE=$(grep -c 'WARNING: DATA RACE' "$l")"; done; } | tee "$OUT/summary_$SUF.txt"
grep -hE '^\s*--- FAIL' "${LOGS[@]}" | sed 's/ (.*//' | sort -u > "$OUT/failures_$SUF.txt"
skip_gate "${LOGS[@]}"; G=$?
cp "$OUT/.netskip.txt" "$OUT/netskips_$SUF.txt"; cp "$OUT/.skip.txt" "$OUT/skips_anywhere_$SUF.txt"
rm -f "$OUT"/.pass.txt "$OUT"/.skip.txt "$OUT"/.netskip.txt
tools/isolation/seal_evidence.sh "$OUT" >/dev/null
[ $G -ne 0 ] && { echo "SKIP-GATE FAILED"; exit 3; }
[ $RC -ne 0 ] && { echo "TEST FAILURES (see $OUT/failures_$SUF.txt)"; exit 1; }
echo "GATE GREEN"; exit 0
