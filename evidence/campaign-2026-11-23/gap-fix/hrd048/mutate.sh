#!/bin/bash
# HRD-048: inject real leaks into the server; the soak must FAIL (RED), clean tree must PASS.
cd "$(dirname "$0")/../../../../server" || exit 1
export GIN_MODE=release; T=$(mktemp -d); F=internal/httpapi/router.go
cp $F "$T/orig"
run() { go test ./internal/httpapi -run 'TestSoakServerRequestMixDoesNotLeak' -count=1 >"$T/out" 2>&1; }
run && echo "BASELINE: GREEN" || echo "BASELINE: RED (bad)"
inject() { python3 - "$1" <<'P'
import sys
s=open("internal/httpapi/router.go").read()
a="func (s *Server) content(c *gin.Context) {\n"
assert a in s
open("internal/httpapi/router.go","w").write(s.replace(a,a+sys.argv[1],1)+"\nvar leakSink [][]byte\nvar leakN int\n")
P
}
inject '	leakSink = append(leakSink, make([]byte, 512))
'
if run; then echo "MUTANT heap_leak_512B_per_content_request_(256B_per_iteration): SURVIVED (bad)"; else echo "MUTANT heap_leak_512B_per_content_request_(256B_per_iteration): KILLED"; grep -m2 "leak detected\|SOAK" "$T/out"; fi
cp "$T/orig" $F
inject '	leakN++
	if leakN%200 == 0 {
		go func() { select {} }()
	}
'
if run; then echo "MUTANT goroutine_leak_1_per_200_content_requests: SURVIVED (bad)"; else echo "MUTANT goroutine_leak_1_per_200_content_requests: KILLED"; grep -m2 "leak detected" "$T/out"; fi
cp "$T/orig" $F
run && echo "RESTORED: GREEN" || echo "RESTORED: RED (bad)"

# rclone helper hygiene: a missing cmd.Wait() leaves zombies; the remote soak / concurrency probe must catch it.
cp internal/sources/rclone.go "$T/rclone.orig"
python3 - <<'P'
p="internal/sources/rclone.go"
s=open(p).read()
a="err := r.cmd.Wait()"
assert a in s
open(p,"w").write(s.replace(a,"var err error",1))
P
if go test ./internal/httpapi -run 'TestRcloneConcurrentRangePeakProcesses' -count=1 >"$T/out" 2>&1; then echo "MUTANT rclone_missing_cmd_Wait: SURVIVED (bad)"; else echo "MUTANT rclone_missing_cmd_Wait: KILLED"; grep -m2 "lingered\|outlived" "$T/out"; fi
cp "$T/rclone.orig" internal/sources/rclone.go
