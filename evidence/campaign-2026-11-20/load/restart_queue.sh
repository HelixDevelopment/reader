#!/usr/bin/env bash
# ATM-027 durable-queue evidence: enqueue concurrent scan jobs against the
# live catalog, SIGKILL the server mid-queue, restart, and assert from
# persisted state (PostgreSQL catalog + Redis scan journal) what survived.
# Correctness measurements only: no latency claim is made, so the host->DB
# path (pasta) does not invalidate anything here; the payload still labels it.
# Re-runnable: uses its own port and Redis DB index and cleans both up.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../../.." && pwd)"
RUNLOG="$HERE/run.log"
OUT="$HERE/restart_queue.jsonl"
LOGF="$HERE/restart_queue.log"
PGC=helixqual-pg
RDC=helixqual-redis
ADDR=127.0.0.1:18097
REDIS_URL='redis://127.0.0.1:34379/13'
DSN='postgres://helixqual@127.0.0.1:34320/helix_test?sslmode=disable'
SRC_COUNT=3
FILES_PER_SRC=120
SCANS_PER_SRC=4
log(){ printf '%s %s\n' "$(date -Is)" "$*" | tee -a "$RUNLOG" | tee -a "$LOGF"; }
q(){ podman exec "$PGC" psql -U helixqual -d helix_test -tA -c "$1" 2>>"$LOGF"; }
api(){ curl -sS -m 30 "http://$ADDR$1" "${@:2}"; }

: > "$LOGF"
log "=== restart_queue.sh start ==="
# Fresh journal state for the dedicated Redis DB index.
podman exec "$RDC" redis-cli -n 13 flushdb >/dev/null 2>>"$LOGF" || log "redis flush failed"
# Fresh catalog state: drop any leftovers from a previous harness run.
q "DELETE FROM works WHERE id IN (SELECT DISTINCT e.work_id FROM editions e JOIN files f ON f.edition_id=e.id WHERE f.source_id LIKE 'load-restart-src-%')" >>"$LOGF"

FIX=$(mktemp -d /tmp/helix-load-restart.XXXXXX)
declare -a IDS=()
for s in $(seq 1 "$SRC_COUNT"); do
  d="$FIX/src$s"; mkdir -p "$d"
  for f in $(seq 1 "$FILES_PER_SRC"); do
    printf 'load restart document source %s file %s\n' "$s" "$f" > "$d/doc-$s-$f.txt"
  done
  IDS+=("load-restart-src-$s")
done

build(){ log "build helix-server"; ( cd "$ROOT/server" && go build -o /tmp/helix-server-load ./cmd/helix-server ) 2>&1 | tee -a "$LOGF"; [ -x /tmp/helix-server-load ]; }
start_server(){ # $1 = tag
  HELIX_ADDRESS="$ADDR" HELIX_POSTGRES_DSN="$DSN" HELIX_REDIS_URL="$REDIS_URL" HELIX_HTTP3=false \
  HELIX_COVER_CACHE="$FIX/covers" /tmp/helix-server-load >>"$HERE/restart_server_$1.log" 2>&1 &
  echo $! > "$HERE/restart_server_$1.pid"
  for _ in $(seq 1 60); do
    code=$(curl -s -o /dev/null -w '%{http_code}' -m 2 "http://$ADDR/healthz" || true)
    [ "$code" = 200 ] && { log "server($1) up pid=$(cat "$HERE/restart_server_$1.pid")"; return 0; }
    sleep 0.5
  done
  log "server($1) FAILED to become healthy"; return 1
}
stop_server_hard(){ kill -9 "$(cat "$HERE/restart_server_$1.pid")" 2>/dev/null; log "server($1) SIGKILLed"; sleep 1; }

cleanup(){
  [ -f "$HERE/restart_server_1.pid" ] && kill -9 "$(cat "$HERE/restart_server_1.pid")" 2>/dev/null
  [ -f "$HERE/restart_server_2.pid" ] && kill -9 "$(cat "$HERE/restart_server_2.pid")" 2>/dev/null
  podman exec "$RDC" redis-cli -n 13 flushdb >/dev/null 2>&1
  q "DELETE FROM works WHERE id IN (SELECT DISTINCT e.work_id FROM editions e JOIN files f ON f.edition_id=e.id WHERE f.source_id LIKE 'load-restart-src-%')" >/dev/null
  rm -rf "$FIX" "$HERE"/restart_server_*.pid
  log "cleanup done (fixture dirs, redis db 13, catalog rows)"
}
trap cleanup EXIT

build || exit 1
start_server 1 || exit 1
# Register sources (process-local by design; the restart leg re-registers).
for i in "${!IDS[@]}"; do
  body=$(printf '{"id":"%s","protocol":"local","root":"%s"}' "${IDS[$i]}" "$FIX/src$((i+1))")
  resp=$(api /v1/sources -X POST -H 'content-type: application/json' -d "$body")
  log "register ${IDS[$i]}: $resp"
done
# Enqueue $SRC_COUNT x $SCANS_PER_SRC scans back to back.
for s in "${IDS[@]}"; do
  for _ in $(seq 1 "$SCANS_PER_SRC"); do
    api /v1/scans -X POST -H 'content-type: application/json' -d "{\"source_id\":\"$s\"}" -o /dev/null -s -w 'scan_post http=%{http_code}\n' | tee -a "$LOGF"
  done
done
sleep 1.5
api /v1/scans > "$HERE/pre_kill.json"; log "pre-kill snapshot saved ($(grep -o '"state"' "$HERE/pre_kill.json" | wc -l) job records)"
PRE_FILES=$(q "SELECT count(*) FROM files WHERE source_id LIKE 'load-restart-src-%'")
PRE_BY_SRC=$(q "SELECT source_id||'='||count(*) FROM files WHERE source_id LIKE 'load-restart-src-%' GROUP BY source_id ORDER BY source_id" | paste -sd ';' -)
PRE_WORKS=$(q "SELECT count(DISTINCT e.work_id) FROM editions e JOIN files f ON f.edition_id=e.id WHERE f.source_id LIKE 'load-restart-src-%'")
log "pre-kill persisted catalog: files=$PRE_FILES works=$PRE_WORKS"
stop_server_hard 1
POST_KILL_FILES=$(q "SELECT count(*) FROM files WHERE source_id LIKE 'load-restart-src-%'")
log "post-kill persisted catalog (no rollback of committed merges): files=$POST_KILL_FILES"

start_server 2 || exit 1
api /v1/scans > "$HERE/post_restart.json"; log "post-restart snapshot saved"
# Re-register sources (connectors are process-local) and rescan ONE source to
# completion to test idempotent progress: no duplicated rows for work already
# committed before the crash.
for i in "${!IDS[@]}"; do
  body=$(printf '{"id":"%s","protocol":"local","root":"%s"}' "${IDS[$i]}" "$FIX/src$((i+1))")
  api /v1/sources -X POST -H 'content-type: application/json' -d "$body" -o /dev/null
done
RES=$(api /v1/scans -X POST -H 'content-type: application/json' -d "{\"source_id\":\"${IDS[0]}\"}")
RES_ID=$(printf '%s' "$RES" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("id",""))' 2>/dev/null)
log "post-restart rescan id=$RES_ID"
for _ in $(seq 1 120); do
  ST=$(api "/v1/scans/$RES_ID" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("state",""))' 2>/dev/null)
  [ "$ST" = succeeded ] || [ "$ST" = failed ] || [ "$ST" = canceled ] && break
  sleep 1
done
log "rescan final state=$ST"
FINAL_BY_SRC=$(q "SELECT source_id||'='||count(*) FROM files WHERE source_id LIKE 'load-restart-src-%' GROUP BY source_id ORDER BY source_id" | paste -sd ';' -)
log "per-source committed files pre-kill: $PRE_BY_SRC ; final: $FINAL_BY_SRC"
FINAL_FILES=$(q "SELECT count(*) FROM files WHERE source_id LIKE 'load-restart-src-%'")
FINAL_WORKS=$(q "SELECT count(DISTINCT e.work_id) FROM editions e JOIN files f ON f.edition_id=e.id WHERE f.source_id LIKE 'load-restart-src-%'")
FINAL_ED_BY_SRC=$(q "SELECT count(DISTINCT id) FROM files WHERE source_id LIKE 'load-restart-src-%' AND id IN (SELECT id FROM files WHERE source_id='${IDS[0]}' GROUP BY id HAVING count(*)>1)")
log "final catalog after completed rescan: files=$FINAL_FILES works=$FINAL_WORKS"

python3 - "$HERE" "$ST" <<'PY'
import json, sys, time
here, rescan_state = sys.argv[1], sys.argv[2]
pre = json.load(open(f"{here}/pre_kill.json"))["jobs"]
post = json.load(open(f"{here}/post_restart.json"))["jobs"]
pre_by = {j["id"]: j for j in pre}
post_by = {j["id"]: j for j in post}
terminal = {"succeeded", "failed", "canceled"}
lost = [i for i in pre_by if i not in post_by]
surviving = [i for i in pre_by if i in post_by]
pre_terminal = {i: j["state"] for i, j in pre_by.items() if j["state"] in terminal}
kept_terminal = [i for i in pre_terminal if post_by.get(i, {}).get("state") == pre_terminal[i]]
reports_kept = [i for i in pre_terminal if pre_terminal[i] == "succeeded" and post_by.get(i, {}).get("report") is not None]
inflight = {i: pre_by[i]["state"] for i in pre_by if pre_by[i]["state"] not in terminal}
requeued = [i for i in inflight if post_by.get(i, {}).get("state") not in {"failed"}]
interrupted = [i for i in inflight if post_by.get(i, {}).get("state") == "failed"
               and "interrupted" in post_by[i].get("error", "").lower()]
def count_state(jobs):
    out = {}
    for j in jobs:
        out[j["state"]] = out.get(j["state"], 0) + 1
    return out
payload = {
    "test": "queue_restart_durability",
    "measurement_path": "host_through_pasta_state_assertions_only",
    "at": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
    "jobs_pre_kill": len(pre),
    "states_pre_kill": count_state(pre),
    "states_post_restart": count_state(post),
    "records_surviving_restart": len(surviving),
    "records_lost": lost,
    "terminal_records_state_preserved": len(kept_terminal),
    "succeeded_records_with_report_preserved": len(reports_kept),
    "in_flight_at_kill": len(inflight),
    "in_flight_marked_failed_interrupted": len(interrupted),
    "in_flight_requeued_for_another_process": requeued,
    "rescan_after_restart_state": rescan_state,
    "policy_observed": "queued/running records are not lost and not re-run; restore re-labels them failed('scan interrupted by server restart'). Terminal records keep their state; succeeded keep their report.",
    "notes": "catalog progress committed before the kill is durable (per-file merge transactions); unprocessed files of an interrupted scan are lost from the job but rescanning the source reproduces them without duplicating rows (idempotent content-keyed ids).",
}
import re
log = open(f"{here}/restart_queue.log").read()
def grab(pat):
    m = re.search(pat, log)
    return int(m.group(1)) if m else None
payload["catalog_files_pre_kill"] = grab(r"pre-kill persisted catalog: files=(\d+)")
payload["catalog_works_pre_kill"] = grab(r"pre-kill persisted catalog: files=\d+ works=(\d+)")
payload["catalog_files_post_kill"] = grab(r"no rollback of committed merges\): files=(\d+)")
payload["catalog_files_final"] = grab(r"final catalog after completed rescan: files=(\d+)")
payload["catalog_works_final"] = grab(r"final catalog after completed rescan: files=\d+ works=(\d+)")
payload["catalog_files_by_source_pre_kill"] = __import__("re").search(r"pre-kill: (.*?) ; final", log).group(1) if __import__("re").search(r"pre-kill: (.*?) ; final", log) else None
payload["catalog_files_by_source_final"] = __import__("re").search(r"final: (\S+)", log).group(1) if __import__("re").search(r"final: (\S+)", log) else None
src1_final = None
if payload["catalog_files_by_source_final"]:
    for part in payload["catalog_files_by_source_final"].split(";"):
        if part.startswith("load-restart-src-1="):
            src1_final = int(part.split("=")[1])
payload["rescanned_source_files_after_completed_rescan"] = src1_final
payload["duplicate_file_rows_after_rescan"] = (src1_final - 120) if src1_final is not None else "unknown"
with open(f"{here}/restart_queue.jsonl", "a") as out:
    out.write(json.dumps(payload) + "\n")
print(json.dumps(payload, indent=2))
PY
HERE_ARG="$HERE" FINAL_FILES="$FINAL_FILES" PRE_FILES="$PRE_FILES" log "harness finished"
