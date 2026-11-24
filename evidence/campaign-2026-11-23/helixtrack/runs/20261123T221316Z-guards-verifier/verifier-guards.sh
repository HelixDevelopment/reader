#!/usr/bin/env bash
set -u
S=$1; ROOT=/home/milosvasic/Projects/helix_reader; HE=$ROOT/evidence/campaign-2026-11-23/helixtrack
TS=$(date -u +%Y%m%dT%H%M%SZ); OUT=$HE/runs/$TS-guards-verifier; mkdir "$OUT" || exit 9; W=$S/wd/g$TS; mkdir -p $W; umask 077
SYNC="python3 -I $ROOT/tools/helixtrack-sync/helixtrack_sync.py"
OB=$ROOT/docs/tracking/helixtrack-outbox.json; ENV=$S/creds/sync.env
A=$OUT/assertions.txt; : > $A
ex(){ if [ "$2" = "$3" ]; then echo "PASS $1 actual=$2 expected=$3" | tee -a $A; else echo "FAIL $1 actual=$2 expected=$3" | tee -a $A; fi; }
reqs(){ podman logs hrd029live3_core 2>&1 | grep -c 'Received /do request'; }
STATE=$W/state.json
# --- G1a lock held externally
python3 -I -c '
import fcntl,os,sys,time
fd=os.open(sys.argv[1],os.O_RDWR|os.O_CREAT,0o600); fcntl.flock(fd,fcntl.LOCK_EX); open(sys.argv[2],"w").write("held"); time.sleep(40)' $STATE.lock $W/held &
HP=$!; until [ -f $W/held ]; do sleep 0.2; done
echo '{"sentinel":"running-instance-report"}' > $W/report.json
b=$(reqs); $SYNC push --outbox $OB --state $STATE --env-file $ENV --record $OUT/g1a.traffic.jsonl --report $W/report.json 2> $OUT/g1a.stderr.txt; r=$?; a=$(reqs)
ex G1a-lock-exit3 $r 3; ex G1a-no-extra-requests $((a-b)) 0
ex G1a-lock-message $(grep -c 'another helixtrack-sync instance holds the lock' $OUT/g1a.stderr.txt) 1
ex G1a-report-not-clobbered "$(cat $W/report.json | tr -d '\n')" '{"sentinel":"running-instance-report"}'
ex G1a-no-traffic-file "$([ -f $OUT/g1a.traffic.jsonl ] && echo present || echo absent)" absent
ex G1a-no-state-file "$([ -f $STATE ] && echo present || echo absent)" absent
kill $HP 2>/dev/null; wait $HP 2>/dev/null
# --- G1b genuine race: two real pushes, project HRM
cL="--outbox $OB --env-file $ENV --project-key HRM --project-name HRM-lock --state $W/stL.json"
$SYNC push $cL --report $OUT/g1b.A.report.json > $OUT/g1b.A.out 2> $OUT/g1b.A.err & PA=$!
for i in $(seq 1 100); do [ -f $W/stL.json.lock ] && break; sleep 0.05; done
$SYNC push $cL --report $OUT/g1b.B.report.json > $OUT/g1b.B.out 2> $OUT/g1b.B.err; rB=$?
wait $PA; rA=$?
echo "A=$rA B=$rB" > $OUT/g1b.exit_codes.txt
ex G1b-race-one-wins-one-refused "$(printf '%s' "$rA,$rB" | sed 's/^0,3$/ok/;s/^3,0$/ok/')" ok
ex G1b-core-tickets-no-dupes "$(sqlite3 $S/run/db/core.sqlite "select count(*)||'/'||count(distinct substr(t.title,1,9)) from ticket t join project p on p.id=t.project_id where p.identifier='HRM' and t.deleted=0")" "75/75"
# --- G2 corrupt state
for kv in 'null-item|{"schema":1,"items":{"HRD-001":null}}' 'truncated|{"items": {"HRD-0' 'items-list|{"items":[1,2]}' 'bad-ticketid|{"items":{"HRD-001":{"ticket_id":7}}}' 'root-list|[]' 'binary|'; do
  n=${kv%%|*}; c=${kv#*|}; if [ $n = binary ]; then head -c 64 /dev/urandom > $W/s_$n.json; else printf '%s' "$c" > $W/s_$n.json; fi
  cp $W/s_$n.json $W/s_$n.before; b=$(reqs)
  $SYNC pull --outbox $OB --state $W/s_$n.json --env-file $ENV --record $OUT/g2-$n.traffic.jsonl --report $OUT/g2-$n.report.json 2> $OUT/g2-$n.stderr.txt; r=$?; a=$(reqs)
  ex G2-$n-exit3 $r 3; ex G2-$n-no-requests $((a-b)) 0
  ex G2-$n-state-untouched "$(cmp -s $W/s_$n.json $W/s_$n.before && echo same || echo changed)" same
  ex G2-$n-error-report "$(python3 -I -c 'import json,sys;d=json.load(open(sys.argv[1]));print(d["aborted"],d["error"]["type"])' $OUT/g2-$n.report.json)" "True SyncError"
done
# --- G3 userinfo / parser-differential URLs
i=0; for base in 'http://thrw3_sync:x@127.0.0.1:18417' 'http://127.0.0.1:18417@10.255.255.1' 'http://10.255.255.1\@127.0.0.1:18417' 'http://127.0.0.1:18417#@evil' 'http://127.0.0.1:18417/?a=b' 'http://127.0.0.1:18417 evil' 'https://user@example.org'; do
  i=$((i+1)); python3 -I -c 'import sys;print("\n".join(("HT_BASE_URL="+sys.argv[2]) if l.startswith("HT_BASE_URL=") else l for l in open(sys.argv[1]).read().split("\n")),end="")' $ENV "$base" > $W/bad$i.env; chmod 600 $W/bad$i.env; b=$(reqs)
  $SYNC pull --outbox $OB --state $W/st3.json --env-file $W/bad$i.env --record $OUT/g3-$i.traffic.jsonl 2> $OUT/g3-$i.stderr.txt; r=$?; a=$(reqs)
  ex G3-$i-exit3 $r 3; ex G3-$i-no-core-requests $((a-b)) 0
  ex G3-$i-no-traffic-file "$([ -f $OUT/g3-$i.traffic.jsonl ] && echo present || echo absent)" absent
  ex G3-$i-url-not-echoed "$(grep -c -F -e '@' -e 'evil' $OUT/g3-$i.stderr.txt)" 0
done
# env-file mode 0640 refused
cp $ENV $W/mode640.env; chmod 640 $W/mode640.env; b=$(reqs)
$SYNC pull --outbox $OB --state $W/st3.json --env-file $W/mode640.env 2> $OUT/g3-mode640.stderr.txt; r=$?; a=$(reqs); ex G3-env-0640-refused-exit3 $r 3; ex G3-env-0640-no-requests $((a-b)) 0
# --- G4 garbage 200 HTML injected by proxy at create#3 (project HRY)
python3 -I $S/garbage_proxy.py 18428 x $OUT/g4-proxy.log & GP=$!; sleep 1
sed 's#:18417#:18428#' $ENV > $W/gp.env; chmod 600 $W/gp.env
cZ="--outbox $OB --env-file $W/gp.env --project-key HRY --project-name HRY-garbage --state $W/stZ.json"
$SYNC push $cZ --record $OUT/g4a.traffic.jsonl --report $OUT/g4a.report.json > $OUT/g4a.stdout.txt 2> $OUT/g4a.stderr.txt; r=$?
ex G4a-garbage-html-exit3 $r 3; cat $OUT/g4a.stderr.txt
ex G4a-no-traceback "$(grep -c Traceback $OUT/g4a.stderr.txt)" 0
ex G4a-partial-report "$(python3 -I -c 'import json,sys;d=json.load(open(sys.argv[1]));print(d["aborted"],d["error"]["key"],d["counts"].get("create"))' $OUT/g4a.report.json)" "True HRD-003 2"
ex G4a-state-has-2 "$(python3 -I -c 'import json,sys;print(sorted(json.load(open(sys.argv[1]))["items"]))' $W/stZ.json)" "['HRD-001', 'HRD-002']"
ex G4a-core-tickets-2-or-3 "$(sqlite3 $S/run/db/core.sqlite "select count(*) from ticket t join project p on p.id=t.project_id where p.identifier='HRY' and t.deleted=0")" 2
kill $GP 2>/dev/null; wait $GP 2>/dev/null
$SYNC push --outbox $OB --env-file $ENV --project-key HRY --project-name HRY-garbage --state $W/stZ.json --report $OUT/g4b.report.json > /dev/null 2> $OUT/g4b.stderr.txt; r=$?
ex G4b-resume-exit0 $r 0
ex G4b-core-final-75 "$(sqlite3 $S/run/db/core.sqlite "select count(*)||'/'||count(distinct substr(t.title,1,9)) from ticket t join project p on p.id=t.project_id where p.identifier='HRY' and t.deleted=0")" "75/75"
# --- secrets + modes
echo "assertions_pass=$(grep -c ^PASS $A) assertions_fail=$(grep -c ^FAIL $A)" | tee $OUT/summary.txt
echo "state_mode=$(stat -c %a $W/stZ.json) lock_mode=$(stat -c %a $W/stZ.json.lock) env_modes=$(stat -c %a $ENV $W/gp.env | tr '\n' ' ')" | tee $OUT/modes.txt
( cd $OUT && sha256sum $(ls | grep -v SHA256SUMS) > SHA256SUMS )
echo OUT=$OUT
