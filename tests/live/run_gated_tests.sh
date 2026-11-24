#!/usr/bin/env bash
# run_gated_tests.sh - HRD-050 / GAP-015: run the WHOLE server+core suite with every
# PG/Redis/HELIX_*-gated live test actually executing, and FAIL on any SKIP that is
# not listed (reason + HRD registry item) in tests/live/allowed_skips.txt.
#
# Usage:  tests/live/run_gated_tests.sh [--no-env] [--out DIR] [--allow FILE] [--check LOG...]
#   --no-env     RED baseline: no containers/env, race pass only (gate must FAIL, ~47 skips)
#   --out DIR    evidence dir (default: a FRESH untracked mktemp dir under $TMPDIR, printed at start; never the tracked,
#                sealed evidence dir. A live/--no-env run refuses a non-empty --out, exit 2)
#   --print-out  print the resolved evidence dir and exit (default dir is removed again)
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
# under evidence/ (hardcoded relative paths) are only DETECTED and REPORTED ($OUT/side_effects_report_*.txt); the
# runner never reverts, deletes, moves or copies them (operator decision, round 6).
# Exit: 0 green; 1 test failure; 3 skip-gate violation; 2 setup error.
set -uo pipefail
set -E
export LC_ALL=C
# Fail closed on a directly-invoked missing command (exit 127). LIMIT: it does NOT fire for a missing command inside
# $(...) or in a `cmd || x` / `if cmd` context, so tools are also checked up front (preflight) and values checked before use.
# shellcheck disable=SC2154  # rc is set inside the trap string
trap 'rc=$?; if [ $rc -eq 127 ]; then echo "FATAL: command not found (line $LINENO), aborting" >&2; exit 2; fi' ERR
# Required tool features (checked, not assumed): without them every verdict could be wrong.
printf 'a\n' | grep -xF a >/dev/null && printf 'a\n' | grep -E '^a$' >/dev/null && echo | awk 'BEGIN{exit 0}' && command -v sort >/dev/null && command -v comm >/dev/null \
  || { echo "FATAL: required grep -x -F / grep -E / awk / sort / comm features unavailable" >&2; exit 2; }
CWD0=$PWD
abs() { case $1 in /*) printf '%s' "$1";; *) printf '%s' "$CWD0/$1";; esac; }
ROOT=$(cd "$(dirname "$0")/../.." && pwd) || exit 2; cd "$ROOT" || exit 2
OUT=""; PRINTOUT=0; ALLOW=tests/live/allowed_skips.txt
SEDIR=""; SEMODE=""; SEA=""; SEB=""; SEC=""; SESKIP=""; NOENV=0; CHECKS=(); PB=""; SEALDIR=""; PREFLIGHT=0; SELFTEST=0
while [ $# -gt 0 ]; do case $1 in
  --no-env) NOENV=1;; --out) OUT=$(abs "$2"); shift;; --allow) ALLOW=$(abs "$2"); shift;; --check) CHECKS+=("$(abs "$2")"); shift;;
  --print-phase-b) PB=$(abs "$2"); shift;; --print-out) PRINTOUT=1;;
  --seal) SEALDIR=$(abs "$2"); shift;; --preflight) PREFLIGHT=1;; --selftest-missing-cmd) SELFTEST=1;; --selftest-wait) SELFTEST=2;;
  --selftest-side-effects) SELFTEST=3; SEDIR=$(abs "$2"); SEMODE=$3; SEA=$(abs "$4"); SEB=$(abs "$5"); SEC=$(abs "$6"); SESKIP=$7; shift 6;;
  *) echo "unknown arg $1" >&2; exit 2;; esac; shift; done
# wait_workers PID... : wait for background workers; any non-zero exit sets RC=1 (B3 failure propagation).
wait_workers() { local p; for p in "$@"; do wait "$p" || RC=1; done; }
# Side effects of the live tests inside the tracked evidence tree are DETECTED and REPORTED only. This code never
# modifies the working tree (no revert, delete, move or copy). Snapshot = `path<TAB>sha256` of every path git status
# lists under evidence/ (NUL-separated, so quoted/spaced/non-ASCII names are exact; a path with a TAB or a rename
# entry is UNCONFIRMED/unsupported).
se_snapshot() { # OUTFILE
  local e p h
  git status --porcelain -z --untracked-files=all evidence | while IFS= read -r -d '' e; do
    p=${e:3}; if [ -f "$p" ]; then h=$(sha256sum -- "$p" | cut -c1-64); else h=-; fi
    printf '%s\t%s\n' "$p" "$h"
  done | LC_ALL=C sort > "$1"
}
se_report() { # BEFORE AFTER REPORT [OUT_REL_TO_SKIP]  (read-only on the tree; writes only REPORT)
  awk -F'\t' -v skip="${4:-}" 'FILENAME==ARGV[1] { b[$1]=$2; next } { a[$1]=$2 }
    END { for (p in a) { if (skip != "" && index(p, skip "/") == 1) continue
            if (!(p in b)) print "NEW\t" p; else if (a[p] != b[p]) print "MODIFIED\t" p }
          for (p in b) { if (skip != "" && index(p, skip "/") == 1) continue
            if (!(p in a)) print "NO-LONGER-LISTED\t" p } }' "$1" "$2" | LC_ALL=C sort > "$3"
}
if [ $SELFTEST = 3 ]; then cd "$SEDIR" || exit 2; case $SEMODE in snap) se_snapshot "$SEA";; report) se_report "$SEA" "$SEB" "$SEC" "$SESKIP";; *) exit 2;; esac; exit 0; fi
if [ $SELFTEST = 1 ]; then hrd_no_such_command_xyz; exit 0; fi
if [ $SELFTEST = 2 ]; then RC=0; ( exit 0 ) & a=$!; ( exit 3 ) & b=$!; wait_workers $a $b; [ $RC -eq 1 ] && exit 7; exit 8; fi
# need TOOL...: exit 2 naming the first missing tool (checked before anything is started).
need() { local t; for t in "$@"; do command -v "$t" >/dev/null 2>&1 || { echo "SETUP-ERROR: required tool '$t' not found" >&2; exit 2; }; done; }
need git awk sort comm sed sha256sum find xargs seq cp tee basename dirname date mktemp cat wc rm mkdir cut paste tr head
[ $NOENV = 1 ] || { [ -n "$PB" ] || [ ${#CHECKS[@]} -gt 0 ] || [ -n "$SEALDIR" ] || need python3 go podman base64; }
[ $PREFLIGHT = 1 ] && { echo "preflight ok"; exit 0; }
[ $NOENV = 0 ] || [ -n "$PB" ] || [ ${#CHECKS[@]} -gt 0 ] || need go
# Default evidence dir is a fresh untracked mktemp dir (F2): the sealed, tracked evidence dir is never a default target.
OUTDEF=0
if [ -z "$OUT" ]; then OUT=$(mktemp -d "${TMPDIR:-/tmp}/hrdlive-out.XXXXXX") && [ -n "$OUT" ] || { echo "SETUP-ERROR: cannot create default evidence dir" >&2; exit 2; }; OUTDEF=1; fi
RUNMODE=1; { [ -n "$PB" ] || [ ${#CHECKS[@]} -gt 0 ] || [ -n "$SEALDIR" ] || [ $PREFLIGHT = 1 ] || [ $SELFTEST != 0 ] || [ $PRINTOUT = 1 ]; } && RUNMODE=0
if [ $RUNMODE = 1 ] && [ $OUTDEF = 0 ] && [ -d "$OUT" ] && [ -n "$(ls -A "$OUT" 2>/dev/null)" ]; then
  echo "SETUP-ERROR: --out '$OUT' is not empty; a run would overwrite/re-seal existing evidence. Use a fresh dir." >&2; exit 2; fi
mkdir -p "$OUT"; OUT=$(cd "$OUT" && pwd) || exit 2
if [ $PRINTOUT = 1 ]; then echo "$OUT"; [ $OUTDEF = 1 ] && rmdir "$OUT" 2>/dev/null; exit 0; fi
[ $RUNMODE = 1 ] && echo "evidence dir: $OUT"
if [ ! -f "$ALLOW" ] || [ ! -r "$ALLOW" ]; then echo "SETUP-ERROR: allow-list '$ALLOW' missing or unreadable" >&2; exit 2; fi
# Scratch files live in a private mktemp dir, never in the (tracked) evidence dir; removed on exit.
SCR=$(mktemp -d) || exit 2; [ -n "$SCR" ] || exit 2
trap 'rm -rf "${SCR:?}"; [ $OUTDEF = 1 ] && [ $RUNMODE = 0 ] && rmdir "$OUT" 2>/dev/null; true' EXIT

# seal_dir DIR : write DIR/SHA256SUMS (sha256 of every regular, non-hidden file except SHA256SUMS itself,
# relative paths, fixed C-locale order). Self-contained: no external seal tool.
seal_dir() {
  local d=$1
  [ -d "$d" ] || { echo "seal: no such dir $d" >&2; return 2; }
  ( cd "$d" || exit 2
    find . -type f ! -name SHA256SUMS ! -name '.*' ! -path '*/.*' -print | LC_ALL=C sort | sed 's|^\./||' | xargs -d '\n' -r sha256sum -- > .SHA256SUMS.tmp \
      && mv .SHA256SUMS.tmp SHA256SUMS )
}
if [ -n "$SEALDIR" ]; then seal_dir "$SEALDIR"; exit $?; fi

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# Identity of a test is package<TAB>name (a bare name can be masked by a same-named test in another package).
# log_records LOG... : TSV records (see parse_gotest.awk). Returns 2 if a log is missing/unreadable/empty
# or holds no `--- PASS/SKIP/FAIL` line (blind instrument: a zero from it would say nothing).
trim() { local v=$1; v=${v#"${v%%[![:space:]]*}"}; v=${v%"${v##*[![:space:]]}"}; printf '%s' "$v"; }
LOGBAD=""
log_records() {
  local l
  for l in "$@"; do
    if [ ! -f "$l" ] || [ ! -r "$l" ] || [ ! -s "$l" ]; then echo "LOG-UNUSABLE: $l (missing, unreadable or empty)" >&2; LOGBAD=1; continue; fi
    # NOTE: never `grep -q` at the end of a pipe under pipefail: early exit SIGPIPEs the writer (rc 141) on big logs.
    if ! sed 's/\x1b\[[0-9;]*[A-Za-z]//g' "$l" | grep -E '^[[:space:]]*--- (PASS|SKIP|FAIL): ' >/dev/null; then echo "LOG-UNUSABLE: $l (no '--- PASS/SKIP/FAIL' line: not a go test -v log)" >&2; LOGBAD=1; continue; fi
    awk -f "$HERE/parse_gotest.awk" "$l"
  done
}

# skip_gate LOG... : 2 = unusable log; 3 = violation; 0 = green.
# Violations: net-skipped test (skips somewhere in its package, passes nowhere) not allow-listed; any FAIL test;
# build failure / failed package / panic / timeout / package with no results; malformed allow-list.
skip_gate() {
  local bad=0 pk n r rr i kind rec
  : > "$SCR/.rec.tsv"; LOGBAD=""
  for l in "$@"; do log_records "$l" >> "$SCR/.rec.tsv"; done
  [ -n "$LOGBAD" ] && return 2
  awk -F'\t' '$1=="PASS"{print $2"|"$3}' "$SCR/.rec.tsv" | sort -u > "$SCR/.pass.txt"
  awk -F'\t' '$1=="SKIP"{print $2"|"$3}' "$SCR/.rec.tsv" | sort -u > "$SCR/.skip.txt"
  comm -23 "$SCR/.skip.txt" "$SCR/.pass.txt" > "$SCR/.netskip.txt"
  echo "skip census: $(wc -l < "$SCR/.skip.txt") distinct package|tests skipped somewhere, $(wc -l < "$SCR/.netskip.txt") never passed"
  : > "$SCR/.used.txt"; local -A SEEN_ALLOW=()
  while IFS='|' read -r pk n r i || [ -n "$pk$n$r$i" ]; do
    pk=$(trim "$pk"); n=$(trim "$n"); r=$(trim "$r"); i=$(trim "$i")
    if [ -z "$pk" ]; then [ -z "$n$r$i" ] && continue
      echo "ALLOWLIST-MALFORMED: empty package field ('|$n|$r|$i')"; bad=1; continue; fi
    case $pk in \#*) continue;; esac
    rr=${r//$'\xc2\xa0'/}   # NBSP is not [[:space:]] in the C locale; strip it so an NBSP-only reason is blank
    if [ -z "$n" ] || ! [[ $rr =~ [^[:space:]] ]] || ! [[ $i =~ ^HRD-[0-9]+$ ]]; then
      echo "ALLOWLIST-MALFORMED: '$pk|$n' needs Package|Test|reason|HRD-NNN (non-blank reason without '|', item exactly HRD-<digits>)"; bad=1
    else
      if [ -n "${SEEN_ALLOW["$pk|$n"]:-}" ]; then
        if [ "${SEEN_ALLOW["$pk|$n"]}" = "$r|$i" ]; then echo "DUPLICATE-ALLOW (warning): $pk|$n listed twice"
        else echo "CONTRADICTORY-ALLOW: $pk|$n listed with different reason/item"; bad=1; fi
      fi
      SEEN_ALLOW["$pk|$n"]="$r|$i"; echo "$pk|$n" >> "$SCR/.used.txt"
    fi
  done < "$ALLOW"
  while read -r rec; do
    [ -z "$rec" ] && continue
    if ! grep -qxF "$rec" "$SCR/.used.txt"; then echo "UNLISTED-SKIP: $rec"; bad=1; fi
  done < "$SCR/.netskip.txt"
  while IFS=$'\t' read -r kind pk n; do
    case $kind in
      FAIL) echo "FAILED-TEST: $pk|$n"; bad=1;;
      BUILDFAIL) echo "BUILD-FAILED: $pk"; bad=1;;
      PKGFAIL) echo "PACKAGE-FAILED: $pk"; bad=1;;
      NORESULTS) echo "NO-RESULTS: $pk (package FAIL with no test results)"; bad=1;;
      PANIC) echo "PANIC: $pk $n"; bad=1;;
      TIMEOUT) echo "TIMEOUT: $pk"; bad=1;;
      INCOMPLETE) echo "INCOMPLETE: $pk ($n; truncated log, unclosed or mismatched # PKG marker, or records with no closing PASS/FAIL)"; bad=1;;
    esac
  done < <(sort -u "$SCR/.rec.tsv" | awk -F'\t' '$1!="PASS" && $1!="SKIP" && $1!="SEEN"')
  while read -r rec; do [ -n "$rec" ] && ! grep -qxF "$rec" "$SCR/.skip.txt" && echo "UNUSED-ALLOW (warning): $rec never skipped in these logs"; done < <(sort -u "$SCR/.used.txt")
  return $bad
}

# phase_b_sets LOG : derive the phase-B test lists from a phase-A log. Sets REGEX (all previously net-skipped
# top-level tests that are not run elsewhere), INRE (in-container set), HRE (host-side set). Returns 2 on an
# unusable log. Tests in package|name form are reduced to bare names for `go test -run`.
HOSTRE='TestLiveLoadRemoteSource|TestLiveLoadRemoteSourceRclone|TestLiveLoadRemoteSourceWebDAV|TestLiveThrottleRemoteSourceHTTP|TestLiveSlowClientCatalogSurface|TestLiveSlowClientJobSurface'
EXCL='TestLiveScaleCatalog|TestCatalogMultiProcessMerges|TestLCPLiveServerQualification|TestDumpLCPFeeds'
phase_b_sets() {
  LOGBAD=""; log_records "$1" > "$SCR/.pb.tsv"
  [ -n "$LOGBAD" ] && { rm -f "$SCR/.pb.tsv"; return 2; }
  REGEX=$(awk -F'\t' -v ex="^($EXCL)$" '
    $1=="PASS"{p[$2"\t"$3]=1} $1=="SKIP"{s[$2"\t"$3]=1}
    END{for(k in s) if(!(k in p)){split(k,a,"\t"); if(a[2]!~/\// && a[2]!~ex) u[a[2]]=1}
        n=0; for(t in u) l[++n]=t; for(i=1;i<=n;i++)for(j=i+1;j<=n;j++)if(l[j]<l[i]){x=l[i];l[i]=l[j];l[j]=x}
        for(i=1;i<=n;i++) printf "%s%s", (i>1?"|":""), l[i]; print ""}' "$SCR/.pb.tsv")
  INRE=$(awk -v h="^($HOSTRE)$" 'BEGIN{RS="|"; ORS=""} {sub(/\n$/,"")} $0!="" && $0!~h {printf "%s%s", (c++?"|":""), $0}' <<<"$REGEX")
  HRE=$(awk -v h="^($HOSTRE)$" 'BEGIN{RS="|"; ORS=""} {sub(/\n$/,"")} $0!="" && $0~h {printf "%s%s", (c++?"|":""), $0}' <<<"$REGEX")
  rm -f "$SCR/.pb.tsv"; return 0
}

if [ -n "$PB" ]; then phase_b_sets "$PB" || { echo "GATE-BLIND: unusable phase-A log" >&2; exit 2; }; echo "REGEX=$REGEX"; echo "INRE=$INRE"; echo "HRE=$HRE"; exit 0; fi

if [ ${#CHECKS[@]} -gt 0 ]; then skip_gate "${CHECKS[@]}"; G=$?
  [ $G -eq 2 ] && { echo "GATE-BLIND: unusable log(s), no verdict possible"; exit 2; }
  [ $G -eq 0 ] && { echo "GATE GREEN"; exit 0; } || { echo "SKIP-GATE FAILED"; exit 3; }; fi

PFX="hrdlive-$$"; CTRS=(); TMPD=$(mktemp -d) && [ -n "$TMPD" ] || { echo "SETUP-ERROR: mktemp failed" >&2; exit 2; }
cleanup() { for ((i=${#CTRS[@]}-1;i>=0;i--)); do podman rm -f -t 2 "${CTRS[$i]}" >/dev/null 2>&1; done; rm -rf "${TMPD:?}" "${SCR:?}"; true; }
trap cleanup EXIT INT TERM
free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])'; }
hdr() { echo "# HRD-050 $1  $(date -u +%FT%TZ)  head=$(git rev-parse --short HEAD)  dirty_files=$(git status --porcelain | wc -l)  dirty_diff_sha=$( (git diff HEAD; git ls-files -o --exclude-standard | sort) | sha256sum | cut -c1-16)  mode=$([ $NOENV = 1 ] && echo no-env || echo live)"; echo "# cmd: $0 $ORIG"; echo "# go: $(go version)"; }
ORIG="$*"
EVBEFORE="$TMPD/ev_before.txt"; se_snapshot "$EVBEFORE"

if [ $NOENV = 1 ]; then SUF=noenv; else SUF=live; fi
LA="$OUT/run_A_race_$SUF.log"; LB="$OUT/run_B_load.log"
hdr "phase A (race, whole suite)" > "$LA"

if [ $NOENV = 0 ]; then
  PGPORT=$(free_port); RPORT=$(free_port); PGPW=$(head -c 12 /dev/urandom | base64 | tr -dc 'A-Za-z0-9')
  [ -n "$PGPORT" ] && [ -n "$RPORT" ] && [ -n "$PGPW" ] || { echo "SETUP-ERROR: could not allocate ports/password" >&2; exit 2; }
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
     && [ "$(podman exec "$PFX-redis" redis-cli ping 2>/dev/null)" = PONG ] && break; sleep 1; done
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
  phase_b_sets "$LA" || { echo "phase-B list construction failed (unusable phase-A log)" >&2; exit 2; }
  echo "=== B2 previously-skipped: $REGEX ===" >> "$LB"
  echo "=== B2h host-side (measurement_path=host_forwarded): $HRE ===" >> "$LB"
  if [ -n "$HRE" ]; then
    ( cd server && HELIX_MEASUREMENT_PATH=host_forwarded go test ./... -count=1 -v -p 1 -timeout 60m -run "^($HRE)\$" ) >> "$LB" 2>&1 || RC=1
  fi
  # Authoritative PG-latency path: test binaries run INSIDE the database container against
  # 127.0.0.1:5432 (redis shares that netns), exactly as the 2026-11-20 campaign did.
  echo "=== B2c in-container (measurement_path=in_container_clean): $INRE ===" >> "$LB"
  # F1: with an empty INRE a binary would print "no tests to run" inside its marker pair and the gate would refuse it as
  # INCOMPLETE (empty-marker-pair). Fail-closed: nothing is run and no marker is written; the tests stay net skips
  # and the skip gate still demands an allow-list entry for each, so nothing is hidden.
  [ -n "$INRE" ] || echo "B2c: INRE empty, no in-container run (tests remain net-skipped and gated)" >> "$LB"
  for pk in catalog httpapi aimemory; do
    [ -n "$INRE" ] || break
    ( cd server && CGO_ENABLED=0 go test -c -o "$TMPD/$pk.test" "./internal/$pk" ) >> "$LB" 2>&1 || RC=1
    podman exec "$PFX-run" mkdir -p "/work/server/internal/$pk" "/work/evidence" >/dev/null
    podman cp "$TMPD/$pk.test" "$PFX-run:/work/server/internal/$pk/$pk.test" >> "$LB" 2>&1
    [ -d "server/internal/$pk/testdata" ] && podman cp "server/internal/$pk/testdata" "$PFX-run:/work/server/internal/$pk/testdata" >> "$LB" 2>&1
    IMP=$(cd server && go list -f '{{.ImportPath}}' "./internal/$pk") && [ -n "$IMP" ] || { echo "go list failed for $pk" >&2; exit 2; }
    echo "# PKG $IMP" >> "$LB"
    podman exec -w "/work/server/internal/$pk" \
      -e HELIX_TEST_POSTGRES_DSN="postgres://postgres:$PGPW@127.0.0.1:5432/helix_test?sslmode=disable" \
      -e HELIX_TEST_REDIS_ADDR=127.0.0.1:6379 -e HELIX_LOAD_RUN=1 -e HELIX_MEASUREMENT_PATH=in_container_clean \
      -e HELIX_LOAD_OUTPUT=/tmp/load_output.json \
      "$PFX-run" "./$pk.test" -test.v -test.count=1 -test.timeout 60m -test.run "^($INRE)\$" >> "$LB" 2>&1 || RC=1
    echo "# PKG-END $IMP" >> "$LB"
  done
  podman cp "$PFX-run:/tmp/load_output.json" "$OUT/load_output_in_container.json" >/dev/null 2>&1
  podman cp "$PFX-run:/work/evidence" "$OUT/in_container_evidence" >/dev/null 2>&1
  echo "=== B3 multi-process merges (4 workers + verify) ===" >> "$LB"
  ( cd server && go test -c -o "$TMPD/catalog.test" ./internal/catalog ) >> "$LB" 2>&1 || RC=1
  IMP=$(cd server && go list -f '{{.ImportPath}}' ./internal/catalog) && [ -n "$IMP" ] || { echo "go list failed for catalog" >&2; exit 2; }
  echo "# PKG $IMP" >> "$LB"
  RUNTAG="hrd050$$"; WP=()
  for i in 0 1 2 3; do
    ( cd server/internal/catalog && HELIX_MP_ROLE=worker HELIX_MP_INDEX=$i HELIX_MP_PROCESSES=4 HELIX_MP_OPS=60 HELIX_MP_RUN=$RUNTAG \
        "$TMPD/catalog.test" -test.v -test.count=1 -test.run '^TestCatalogMultiProcessMerges$' ) > "$TMPD/worker$i.log" 2>&1 &
    WP+=($!)
  done
  wait_workers "${WP[@]}"
  for i in 0 1 2 3; do cat "$TMPD/worker$i.log" >> "$LB"; done   # fixed order, no interleaving
  ( cd server/internal/catalog && HELIX_MP_ROLE=verify HELIX_MP_INDEX=0 HELIX_MP_PROCESSES=4 HELIX_MP_OPS=60 HELIX_MP_RUN=$RUNTAG \
      "$TMPD/catalog.test" -test.v -test.count=1 -test.run '^TestCatalogMultiProcessMerges$' ) >> "$LB" 2>&1 || RC=1
  echo "# PKG-END $IMP" >> "$LB"
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
se_snapshot "$TMPD/ev_after.txt"
se_report "$EVBEFORE" "$TMPD/ev_after.txt" "$OUT/side_effects_report_$SUF.txt" "${OUT#"$ROOT"/}"
if [ -s "$OUT/side_effects_report_$SUF.txt" ]; then echo "SIDE-EFFECTS-DETECTED: $(wc -l < "$OUT/side_effects_report_$SUF.txt") path(s) under evidence/ changed by this run (NOT reverted, see side_effects_report_$SUF.txt)"; fi

{ for l in "${LOGS[@]}"; do echo "$(basename "$l"): PASS=$(grep -cE '^\s*--- PASS' "$l") FAIL=$(grep -cE '^\s*--- FAIL' "$l") SKIP=$(grep -cE '^\s*--- SKIP' "$l") DATA_RACE=$(grep -c 'WARNING: DATA RACE' "$l")"; done; } | tee "$OUT/summary_$SUF.txt"
grep -hE '^\s*--- FAIL' "${LOGS[@]}" | sed 's/ (.*//' | sort -u > "$OUT/failures_$SUF.txt"
skip_gate "${LOGS[@]}"; G=$?
cp "$SCR/.netskip.txt" "$OUT/netskips_$SUF.txt"; cp "$SCR/.skip.txt" "$OUT/skips_anywhere_$SUF.txt"
seal_dir "$OUT" || { echo "sealing evidence failed" >&2; exit 2; }
[ $G -eq 2 ] && { echo "GATE-BLIND: unusable log(s)"; exit 2; }
[ $G -ne 0 ] && { [ $RC -ne 0 ] && echo "ALSO: go test returned failures (see $OUT/failures_$SUF.txt)"; echo "SKIP-GATE FAILED"; exit 3; }
[ $RC -ne 0 ] && { echo "TEST FAILURES (see $OUT/failures_$SUF.txt)"; exit 1; }
echo "GATE GREEN"; exit 0
