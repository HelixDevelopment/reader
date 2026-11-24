#!/usr/bin/env bash
# Live HRD-029 run (round 2) against a self-hosted Helix-Track/Core on loopback.
# usage: run_live.sh <env-file sync user (0600)> <env-file remote editor (0600)> <core sqlite db> <workdir (outside repo)>
# Results go to a FRESH directory runs/<UTC timestamp>/ (built as .run-<ts>.tmp and renamed into place);
# nothing already inside the evidence directory is ever deleted or overwritten; `latest` is swapped atomically.
set -u
umask 077   # assertions.txt, logs and every artifact of the run are created owner-only
ENV_SYNC=$1; ENV_REMOTE=$2; COREDB=$3; WORKROOT=$4
HERE=$(cd "$(dirname "$0")" && pwd); ROOT=$(cd "$HERE/../../.." && pwd)
TS=$(date -u +%Y%m%dT%H%M%SZ); OUT=$HERE/.run-$TS.tmp; WORK=$WORKROOT/$TS
mkdir "$OUT" "$WORK" || exit 9; mkdir -p "$HERE/runs"
SYNC="python3 -I $ROOT/tools/helixtrack-sync/helixtrack_sync.py"
OUTBOX=$ROOT/docs/tracking/helixtrack-outbox.json; DB=$ROOT/docs/workable_items.db
STATE=$WORK/state.json; STATEX=$WORK/state_hrx.json
N=$(python3 -I -c 'import json,sys;print(len(json.load(open(sys.argv[1]))["records"]))' "$OUTBOX"); export N
export ASSERT_LOG=$OUT/assertions.txt; : > "$ASSERT_LOG"
A="python3 -I $HERE/live_assert.py"
common="--outbox $OUTBOX --db $DB --state $STATE --env-file $ENV_SYNC"
step(){ echo "== $1"; }
cntp(){ sqlite3 "$COREDB" "select count(*) from ticket t join project p on p.id=t.project_id where p.identifier='$1' and t.deleted=0"; }
rc(){ echo "$1 exit_code=$2" | tee -a "$OUT/exit_codes.txt"; }
expect_rc(){ if [ "$2" = "$3" ]; then echo "PASS $1 exit_code=$2 expected=$3" | tee -a "$ASSERT_LOG"; else echo "FAIL $1 exit_code=$2 expected=$3" | tee -a "$ASSERT_LOG"; fi; }
expect_eq(){ if [ "$2" = "$3" ]; then echo "PASS $1 actual=$2 expected=$3" | tee -a "$ASSERT_LOG"; else echo "FAIL $1 actual=$2 expected=$3" | tee -a "$ASSERT_LOG"; fi; }
umask 077
sed 's#:18417#:18418#' "$ENV_SYNC" > "$WORK/ht_proxy.env"; chmod 600 "$WORK/ht_proxy.env"
PORTS_OK=1
echo '{}' > "$WORK/rules.json"
python3 -I "$HERE/fault_proxy.py" --listen 18418 --upstream http://127.0.0.1:18417 --rules "$WORK/rules.json" \
  --sink-port 18419 --sink-log "$OUT/16-redirect-sink.hits.log" --log "$OUT/proxy.log" &
PROXY_PID=$!; trap 'kill $PROXY_PID 2>/dev/null' EXIT; sleep 1

step "00 server identity (public actions, no auth)"
set -a; . "$ENV_SYNC"; set +a
for a in version jwtCapable dbCapable; do curl -s -m 10 -X POST $HT_BASE_URL/do -H 'Content-Type: application/json' -d "{\"action\":\"$a\"}"; echo; done > "$OUT/00-server-identity.jsonl"
unset HT_PASSWORD HT_USER HT_BASE_URL
echo "tickets_before=$(cnt=$(sqlite3 "$COREDB" 'select count(*) from ticket where deleted=0'); echo $cnt)" > "$OUT/00-counts.txt"

step "01 dry-run push (no writes, no state file)"
$SYNC push $common --dry-run --record "$OUT/01-dryrun.traffic.jsonl" --report "$OUT/01-dryrun.report.json"; r=$?; rc 01-dryrun $r; expect_rc 01-dryrun $r 0
echo "tickets_after_dryrun=$(sqlite3 "$COREDB" 'select count(*) from ticket where deleted=0')" >> "$OUT/00-counts.txt"
[ -f "$STATE" ] && expect_eq 01-dryrun-state-file-absent present absent || expect_eq 01-dryrun-state-file-absent absent absent
$A 01-dryrun-planned-creates "$OUT/01-dryrun.report.json" 'd["counts"]' '{"create": @N}'

step "02 first push"
$SYNC push $common --record "$OUT/02-push1.traffic.jsonl" --report "$OUT/02-push1.report.json"; r=$?; rc 02-push1 $r; expect_rc 02-push1 $r 0
$A 02-push1-counts "$OUT/02-push1.report.json" 'd["counts"]' '{"create": @N}'
expect_eq 02-core-ticket-count "$(cntp HRD)" $N

step "03 compare registry vs remote field-by-field"
$SYNC compare $common --record "$OUT/03-compare1.traffic.jsonl" --report "$OUT/03-compare1.report.json"; r=$?; rc 03-compare $r; expect_rc 03-compare $r 0
$A 03-compare-matching "$OUT/03-compare1.report.json" '[d["matching"], d["total"]]' '[@N, @N]'

step "04 pull (expect all in-sync)"
$SYNC pull $common --record "$OUT/04-pull1.traffic.jsonl" --report "$OUT/04-pull1.report.json"; r=$?; rc 04-pull $r; expect_rc 04-pull $r 0
$A 04-pull-in-sync "$OUT/04-pull1.report.json" 'd["counts"]' '{"in-sync": @N}'

step "05 re-push (expect all noop, zero writes)"
$SYNC push $common --record "$OUT/05-push2.traffic.jsonl" --report "$OUT/05-push2.report.json"; r=$?; rc 05-repush $r; expect_rc 05-repush $r 0
$A 05-repush-noop "$OUT/05-push2.report.json" 'd["counts"]' '{"noop": @N}'
python3 -I - "$OUT/05-push2.traffic.jsonl" > "$OUT/05-push2.writes.txt" <<'PY'
import json,sys
n=0
for l in open(sys.argv[1]):
    r=json.loads(l)["request"]
    if r.get("action") in ("create","modify") and r.get("data",{}).get("version")!=2147483647: n+=1
print("real_write_requests_in_repush=%d" % n)
PY
cat "$OUT/05-push2.writes.txt"; expect_eq 05-repush-write-requests "$(cut -d= -f2 "$OUT/05-push2.writes.txt")" 0
expect_eq 05-core-ticket-count "$(cntp HRD)" $N

step "06 second user edits HRD-007 on the remote (status+description)"
python3 -I "$HERE/live_actions.py" "$ENV_REMOTE" remote-edit HRD-007 "$OUT/06-remote-edit.traffic.jsonl" | tee "$OUT/06-remote-edit.result.json"

step "07 pull detects the remote edit (with version probes)"
$SYNC pull $common --record "$OUT/07-pull2.traffic.jsonl" --report "$OUT/07-pull2.report.json"; r=$?; rc 07-pull-drift $r; expect_rc 07-pull-drift $r 2
$A 07-remote-edit-detected "$OUT/07-pull2.report.json" '[a["action"] for a in d["actions"] if a["key"]=="HRD-007"]' '["remote-edit"]'
$A 07-remote-version-advanced "$OUT/07-pull2.report.json" '[a["remote_version"] > a["base_version"] for a in d["actions"] if a["key"]=="HRD-007"]' '[true]'

step "07b pull --no-probe issues no probe modify requests"
$SYNC pull $common --no-probe --record "$OUT/07b-pull-noprobe.traffic.jsonl" --report "$OUT/07b-pull-noprobe.report.json"; r=$?; rc 07b-pull-noprobe $r; expect_rc 07b-pull-noprobe $r 2
python3 -I - "$OUT/07b-pull-noprobe.traffic.jsonl" > "$OUT/07b-probe-count.txt" <<'PY'
import json,sys
print("probe_modify_requests=%d" % sum(1 for l in open(sys.argv[1]) if json.loads(l)["request"].get("data",{}).get("version")==2147483647))
PY
cat "$OUT/07b-probe-count.txt"; expect_eq 07b-probe-requests "$(cut -d= -f2 "$OUT/07b-probe-count.txt")" 0

step "08 push does NOT clobber the remote edit (conflict, exit 2)"
$SYNC push $common --record "$OUT/08-push3.traffic.jsonl" --report "$OUT/08-push3.report.json"; r=$?; rc 08-push-conflict $r; expect_rc 08-push-conflict $r 2
$A 08-conflict "$OUT/08-push3.report.json" 'd["counts"]' '{"noop": @N1, "conflict": 1}'

step "09 stale-version write is rejected by Core (optimistic locking)"
python3 -I "$HERE/live_actions.py" "$ENV_REMOTE" stale-version HRD-007 "$OUT/09-stale.traffic.jsonl" | tee "$OUT/09-stale.result.json"
$A 09-stale-rejected "$OUT/09-stale.result.json" 'd["result"]' '"VersionConflict"'

step "10 registry-side change (copy of outbox, HRD-012 status -> In progress) pushes as update"
python3 -I - "$OUTBOX" "$WORK/outbox-changed.json" <<'PY'
import json,sys
d=json.load(open(sys.argv[1]))
for r in d["records"]:
    if r["client_key"]=="HRD-012": r["payload"]["status"]="In progress"
json.dump(d,open(sys.argv[2],"w"))
PY
$SYNC push --outbox "$WORK/outbox-changed.json" --db $DB --state $STATE --env-file $ENV_SYNC --record "$OUT/10-push4.traffic.jsonl" --report "$OUT/10-push4.report.json"; r=$?; rc 10-push-registry-change $r; expect_rc 10-push-registry-change $r 2
$A 10-update-and-conflict "$OUT/10-push4.report.json" 'd["counts"]' '{"noop": @N2, "conflict": 1, "update": 1}'

step "11 operator reviews conflict, pushes with --overwrite-remote (original outbox)"
$SYNC push $common --overwrite-remote --record "$OUT/11-push5.traffic.jsonl" --report "$OUT/11-push5.report.json"; r=$?; rc 11-overwrite $r; expect_rc 11-overwrite $r 0

step "12 pull + compare (expect all in-sync / N of N)"
$SYNC pull $common --record "$OUT/12-pull3.traffic.jsonl" --report "$OUT/12-pull3.report.json"; r=$?; rc 12-pull $r; expect_rc 12-pull $r 0
$SYNC compare $common --record "$OUT/12-compare2.traffic.jsonl" --report "$OUT/12-compare2.report.json"; r=$?; rc 12-compare $r; expect_rc 12-compare $r 0

step "13 remote RETITLE (key removed) -> detected, never re-created, restored only with --overwrite-remote"
python3 -I "$HERE/live_actions.py" "$ENV_REMOTE" remote-retitle HRD-009 "$OUT/13-retitle.traffic.jsonl" | tee "$OUT/13-retitle.result.json"
$SYNC pull $common --record "$OUT/13-pull.traffic.jsonl" --report "$OUT/13-pull.report.json"; r=$?; rc 13-pull-retitle $r; expect_rc 13-pull-retitle $r 2
$A 13-retitle-detected "$OUT/13-pull.report.json" '[[a["action"], "title" in a["changed_fields"]] for a in d["actions"] if a["key"]=="HRD-009"]' '[["remote-edit", true]]'
$SYNC push $common --record "$OUT/13-push.traffic.jsonl" --report "$OUT/13-push.report.json"; r=$?; rc 13-push-retitle $r; expect_rc 13-push-retitle $r 2
$A 13-retitle-conflict-not-create "$OUT/13-push.report.json" 'd["counts"]' '{"noop": @N1, "conflict": 1}'
expect_eq 13-core-ticket-count-unchanged "$(cntp HRD)" $N
$SYNC push $common --overwrite-remote --record "$OUT/13-restore.traffic.jsonl" --report "$OUT/13-restore.report.json"; r=$?; rc 13-restore $r; expect_rc 13-restore $r 0

step "14 remote DELETE -> remote-deleted (exit 2), no silent re-create; --overwrite-remote re-creates explicitly"
python3 -I "$HERE/live_actions.py" "$ENV_REMOTE" remote-delete HRD-015 "$OUT/14-delete.traffic.jsonl" | tee "$OUT/14-delete.result.json"
expect_eq 14-core-live-tickets-after-delete "$(cntp HRD)" $((N-1))
$SYNC pull $common --record "$OUT/14-pull.traffic.jsonl" --report "$OUT/14-pull.report.json"; r=$?; rc 14-pull-deleted $r; expect_rc 14-pull-deleted $r 2
$A 14-pull-remote-deleted "$OUT/14-pull.report.json" '[a["action"] for a in d["actions"] if a["key"]=="HRD-015"]' '["remote-deleted"]'
$SYNC push $common --record "$OUT/14-push.traffic.jsonl" --report "$OUT/14-push.report.json"; r=$?; rc 14-push-deleted $r; expect_rc 14-push-deleted $r 2
$A 14-push-remote-deleted "$OUT/14-push.report.json" 'd["counts"]' '{"noop": @N1, "remote-deleted": 1}'
expect_eq 14-no-silent-recreate "$(cntp HRD)" $((N-1))
$SYNC push $common --overwrite-remote --record "$OUT/14-recreate.traffic.jsonl" --report "$OUT/14-recreate.report.json"; r=$?; rc 14-recreate $r; expect_rc 14-recreate $r 0
$A 14-recreate "$OUT/14-recreate.report.json" 'd["counts"]' '{"noop": @N1, "recreate": 1}'
expect_eq 14-core-live-tickets-after-recreate "$(cntp HRD)" $N
$SYNC pull $common --record "$OUT/14-pull-final.traffic.jsonl" --report "$OUT/14-pull-final.report.json"; r=$?; rc 14-pull-final $r; expect_rc 14-pull-final $r 0

step "15 503 mid-batch + failed status-modify after create -> partial state, then resume (project HRX, via fault proxy)"
cX="--outbox $OUTBOX --db $DB --state $STATEX --env-file $WORK/ht_proxy.env --project-key HRX --project-name HRX-resume"
echo '{"fail_status_modify_nth": 1}' > "$WORK/rules.json"
$SYNC push $cX --record "$OUT/15a.traffic.jsonl" --report "$OUT/15a.report.json"; r=$?; rc 15a-status-modify-503 $r; expect_rc 15a-status-modify-503 $r 3
$A 15a-aborted "$OUT/15a.report.json" '[d["aborted"], d["error"]["type"], d["error"]["key"]]' '[true, "ServerError", "HRD-001"]'
$A 15a-state-has-created-baseline "$STATEX" 'sorted(d["items"])' '["HRD-001"]'
expect_eq 15a-core-tickets-hrx "$(cntp HRX)" 1
echo '{"fail_create_nth": 10}' > "$WORK/rules.json"
$SYNC push $cX --record "$OUT/15b.traffic.jsonl" --report "$OUT/15b.report.json"; r=$?; rc 15b-create-503 $r; expect_rc 15b-create-503 $r 3
$A 15b-aborted "$OUT/15b.report.json" '[d["aborted"], d["error"]["type"]]' '[true, "ServerError"]'
$A 15b-first-item-was-update-not-conflict "$OUT/15b.report.json" '[a["action"] for a in d["actions"] if a["key"]=="HRD-001"]' '["update"]'
expect_eq 15b-core-tickets-hrx-partial "$(cntp HRX)" 10
echo '{}' > "$WORK/rules.json"
$SYNC push $cX --record "$OUT/15c.traffic.jsonl" --report "$OUT/15c.report.json"; r=$?; rc 15c-resume $r; expect_rc 15c-resume $r 0
$A 15c-resume-counts "$OUT/15c.report.json" '[d["counts"].get("create"), d["counts"].get("noop"), d["has_conflicts"]]' '[@N10, 10, false]'
expect_eq 15c-core-tickets-hrx-final "$(cntp HRX)" $N
$SYNC compare $cX --record "$OUT/15c-compare.traffic.jsonl" --report "$OUT/15c-compare.report.json"; r=$?; rc 15c-compare $r; expect_rc 15c-compare $r 0
$A 15c-compare-matching "$OUT/15c-compare.report.json" '[d["matching"], d["total"]]' '[@N, @N]'
expect_eq 15c-distinct-keys-hrx "$(sqlite3 "$COREDB" "select count(distinct substr(t.title,2,7)) from ticket t join project p on p.id=t.project_id where p.identifier='HRX'")" $N

step "16 redirect refusal (authenticated request answered 302 to a sink)"
echo '{"redirect_authed": true}' > "$WORK/rules.json"
$SYNC pull $common --env-file "$WORK/ht_proxy.env" --record "$OUT/16-redirect.traffic.jsonl" --report "$OUT/16-redirect.report.json" 2> "$OUT/16-redirect.stderr.txt"; r=$?; rc 16-redirect $r; expect_rc 16-redirect $r 3
cat "$OUT/16-redirect.stderr.txt"
expect_eq 16-redirect-error-type "$(grep -c 'RedirectRefused' "$OUT/16-redirect.stderr.txt")" 1
expect_eq 16-sink-received-requests "$( [ -f "$OUT/16-redirect-sink.hits.log" ] && grep -c SINK-HIT "$OUT/16-redirect-sink.hits.log" || echo 0 )" 0
grep -c "INJECT-302" "$OUT/proxy.log" | sed 's/^/proxy_302_responses_sent=/' | tee "$OUT/16-proxy-302-count.txt"
echo '{}' > "$WORK/rules.json"

step "17 scheme guard: uppercase HTTP:// to a non-loopback host is refused before any connection; HTTP:// to loopback works"
for base in "HTTP://10.255.255.1:18417" "http://10.255.255.1:18417" "Http://example.org"; do
  umask 077; sed "s#^HT_BASE_URL=.*#HT_BASE_URL=$base#" "$ENV_SYNC" > "$WORK/ht_bad.env"; chmod 600 "$WORK/ht_bad.env"
  tag=$(echo "$base" | tr -c 'A-Za-z0-9\n' '_')
  $SYNC pull --outbox $OUTBOX --state $STATE --env-file "$WORK/ht_bad.env" --record "$OUT/17-$tag.traffic.jsonl" 2> "$OUT/17-$tag.stderr.txt"; r=$?
  rc "17-refuse-$tag" $r; expect_rc "17-refuse-$tag" $r 3
  expect_eq "17-$tag-refusal-message" "$(grep -c 'refusing to send credentials' "$OUT/17-$tag.stderr.txt")" 1
  expect_eq "17-$tag-no-request-recorded" "$( [ -f "$OUT/17-$tag.traffic.jsonl" ] && echo recorded || echo none )" none
done
sed "s#^HT_BASE_URL=http://#HT_BASE_URL=HTTP://#" "$ENV_SYNC" > "$WORK/ht_upper_loop.env"; chmod 600 "$WORK/ht_upper_loop.env"
$SYNC pull --outbox $OUTBOX --db $DB --state $STATE --env-file "$WORK/ht_upper_loop.env" --report "$OUT/17-uppercase-loopback.report.json" > /dev/null; r=$?; rc 17-uppercase-loopback-accepted $r; expect_rc 17-uppercase-loopback-accepted $r 0

step "18 sink-side (Core DB) checks"
{
 echo "-- live tickets per project / min,max version"; sqlite3 "$COREDB" "select p.identifier, count(*), min(t.version), max(t.version) from ticket t join project p on p.id=t.project_id where t.deleted=0 group by 1 order by 1"
 echo "-- soft-deleted tickets (HRD-015 original)"; sqlite3 "$COREDB" "select substr(title,1,9), deleted from ticket where deleted=1"
 echo "-- HRD-007 history (version|action|user)"; sqlite3 "$COREDB" "select h.version, h.action, h.user_id from ticket_history h join ticket t on t.id=h.ticket_id where t.title like '[HRD-007]%' and t.deleted=0 order by h.version"
 echo "-- HRD-009 history (version|action|user)"; sqlite3 "$COREDB" "select h.version, h.action, h.user_id from ticket_history h join ticket t on t.id=h.ticket_id join project p on p.id=t.project_id where p.identifier='HRD' and t.title like '[HRD-009]%' order by h.version"
 echo "-- probe side effects: history rows with title 'probe' (must be 0)"; sqlite3 "$COREDB" "select count(*) from ticket where title='probe'"
} > "$OUT/18-sink-side-core-db.txt" 2>&1
expect_eq 18-no-probe-titles "$(sqlite3 "$COREDB" "select count(*) from ticket where title='probe'")" 0

step "19 secret scan of this run's evidence"
{ if grep -rEl 'eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}' "$OUT" --exclude=19-secret-scan.txt; then echo "JWT-LIKE STRING FOUND"; else echo "no JWT-like strings"; fi
  for f in "$ENV_SYNC" "$ENV_REMOTE"; do while IFS='=' read -r k v; do [ "$k" = HT_PASSWORD ] && { grep -rFl --exclude=19-secret-scan.txt -- "$v" "$OUT" && echo "PASSWORD FOUND"; }; done < "$f"; done
  echo "password scan done"; } > "$OUT/19-secret-scan.txt"
expect_eq 19-secret-scan "$(grep -c 'FOUND' "$OUT/19-secret-scan.txt")" 0

step "20 inputs + manifest"
{ echo "outbox_sha256=$(sha256sum $OUTBOX | cut -d' ' -f1)"; echo "registry_db_sha256=$(sha256sum $DB | cut -d' ' -f1)"
  for f in tools/helixtrack-sync/helixtrack_sync.py tools/helixtrack-sync/test_helixtrack_sync.py; do echo "$f sha256=$(sha256sum $ROOT/$f | cut -d' ' -f1)"; done
  for f in run_live.sh fault_proxy.py live_actions.py live_assert.py run_mutants.py; do echo "$f sha256=$(sha256sum $HERE/$f | cut -d' ' -f1)"; done
  echo "run_ts=$TS"; } > "$OUT/20-inputs.txt"
echo "assertions_pass=$(grep -c '^PASS' "$ASSERT_LOG") assertions_fail=$(grep -c '^FAIL' "$ASSERT_LOG")" | tee "$OUT/21-assertion-summary.txt"
kill $PROXY_PID 2>/dev/null; wait $PROXY_PID 2>/dev/null
( cd "$OUT" && sha256sum $(ls | grep -v '^SHA256SUMS$') > SHA256SUMS )
mv -T "$OUT" "$HERE/runs/$TS" && ln -sfn "runs/$TS" "$HERE/latest.tmp.$TS" && mv -T "$HERE/latest.tmp.$TS" "$HERE/latest"
echo "RUN_DIR=$HERE/runs/$TS"
