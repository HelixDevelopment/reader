#!/usr/bin/env bash
# Needle gate for the remote-source delivery harness (HRD-027, Phase 1).
#
# The local-source gate (needle_delivery.sh) proves the range path works when a
# connector can revalidate. This one proves the opposite half: the harness notices
# when a connector that *cannot* revalidate is allowed to pretend otherwise. Each
# of those is a claim the remote harness makes on its own, so each needs its own
# needle; reusing the local gate's cases would leave the remote contract untested.
#
# A mutation that passes is a hole in the harness, not a clean run: this script
# exits non-zero in that case.
#
# Restores from a cp snapshot, never from git — the working tree carries
# uncommitted campaign work.
set -u

SERVER_DIR="${SERVER_DIR:-$HOME/Projects/helix_reader/server}"
ROUTER="$SERVER_DIR/internal/httpapi/router.go"
HTTPGO="$SERVER_DIR/internal/sources/http.go"
EVIDENCE="${EVIDENCE_DIR:-$HOME/Projects/helix_reader/evidence/campaign-2026-11-20/remote/needles}"
BACKUP="$(mktemp -d /tmp/needle-remote-backup-XXXXXX)"
CONTAINER="${CONTAINER:-helixqual-pg}"
BIN=/tmp/httpapi_remote_needle

mkdir -p "$EVIDENCE"
cp "$ROUTER" "$BACKUP/router.go"
cp "$HTTPGO" "$BACKUP/http.go"
restore() { cp "$BACKUP/router.go" "$ROUTER"; cp "$BACKUP/http.go" "$HTTPGO"; }
trap restore EXIT

RESULTS="$EVIDENCE/needle_gate_results.json"
: > "$EVIDENCE/needle_log.txt"
echo '[' > "$RESULTS"
FIRST=1
FAILURES=0

# run_case <name> <file> <expected-message>
run_case() {
  local name="$1" file="$2" expect="$3"
  local out="$EVIDENCE/${name}.log" status="NEEDLE_DID_NOT_FIRE" rc=0
  if (cd "$SERVER_DIR" && go test -c -o "$BIN" ./internal/httpapi/) >"$out" 2>&1; then
    podman cp "$BIN" "$CONTAINER:/tmp/httpapi_remote_needle" >>"$out" 2>&1
    # The remote harness needs no database: the origin is an in-process
    # httptest.Server, which is why the payload labels itself in_process_loopback.
    podman exec "$CONTAINER" bash -c "cd /tmp && GIN_MODE=release HELIX_LOAD_RUN=1 \
      HELIX_REMOTE_SIZES=262144 HELIX_LOAD_LEVELS=1,4 HELIX_DELIVERY_ROUNDS=2 \
      HELIX_DELIVERY_TIMEOUT_SECONDS=30 \
      timeout 600 /tmp/httpapi_remote_needle -test.run TestLiveLoadRemoteSource -test.v" >>"$out" 2>&1
    rc=$?
    if grep -qF -- "$expect" "$out"; then
      if [ "$rc" -ne 0 ]; then status="FIRED"; else status="MUTATION_PASSED_TEST"; fi
    fi
    printf '%s rc=%d status=%s expected=%s\n' "$name" "$rc" "$status" "$expect" >> "$EVIDENCE/needle_log.txt"
    grep -m3 -F -- "$expect" "$out" >> "$EVIDENCE/needle_log.txt"
  else
    printf '%s status=BUILD_FAILED\n' "$name" >> "$EVIDENCE/needle_log.txt"
  fi
  [ "$FIRST" -eq 1 ] && FIRST=0 || sed -i '$ s/$/,/' "$RESULTS"
  printf '  {"case": "%s", "target": "%s", "expected_failure": %s, "status": "%s"}\n' \
    "$name" "$file" "$(python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$expect")" "$status" >> "$RESULTS"
  echo "$name: $status"
  [ "$status" = "FIRED" ] || FAILURES=$((FAILURES+1))
}

# 1. A non-revalidating source starts advertising the scan-time strong ETag.
#    The hash describes bytes nobody confirmed are still there, so a client that
#    caches or resumes on it would splice representations.
restore
python3 - "$ROUTER" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\t\tc.Header("ETag", etag)\n\t}\n\trangeHeader := c.GetHeader("Range")'
assert old in s, "etag anchor"
new = '\t\tc.Header("ETag", etag)\n\t}\n\tif !current {\n\t\tc.Header("ETag", etag)\n\t}\n\trangeHeader := c.GetHeader("Range")'
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case remote_etag_lie "$ROUTER" "advertised strong validator"

# 2. Accept-Ranges is advertised while the size was never confirmed live. The
#    route still refuses to serve a range, so the header would be a promise the
#    next request breaks.
restore
python3 - "$ROUTER" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\t\tc.Header("ETag", etag)\n\t}\n\trangeHeader := c.GetHeader("Range")'
assert old in s, "ranges anchor"
new = '\t\tc.Header("ETag", etag)\n\t}\n\tif !current {\n\t\tc.Header("Accept-Ranges", "bytes")\n\t}\n\trangeHeader := c.GetHeader("Range")'
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case remote_ranges_lie "$ROUTER" "while the route cannot honour a range"

# 3. The range decision stops requiring live confirmation: the route starts
#    answering a genuine 206 for a source whose current size it never learned.
#    sizeKnown is deliberately left false so the full GET still passes the
#    validator-absence checks and this isolates the range decision alone; that
#    also means the window length has to come from the stale catalog size, which
#    is exactly the splice the gate exists to catch.
restore
python3 - "$ROUTER" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\tserveRange := current && sizeKnown && rangeHeader != "" &&'
assert old in s, "serveRange anchor"
new = '\tserveRange := f.Size > 0 && rangeHeader != "" &&'
s = s.replace(old, new, 1)
old2 = '\t\tstart, length, err := singleRange(rangeHeader, size)'
assert old2 in s, "singleRange anchor"
s = s.replace(old2, '\t\tstart, length, err := singleRange(rangeHeader, f.Size)', 1)
open(p, 'w').write(s)
MUT
run_case remote_range_unvalidated "$ROUTER" "must get a full 200"

# 4. The connector silently fetches the object twice per client request. Every
#    byte-count and latency figure would still look sane; only the origin's own
#    request counter exposes it.
restore
python3 - "$HTTPGO" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = 'func (h *HTTP) Open(ctx context.Context, name string) (io.ReadCloser, error) {\n\tresp, err := h.do(ctx, http.MethodGet, name)'
assert old in s, "Open anchor"
new = ('func (h *HTTP) Open(ctx context.Context, name string) (io.ReadCloser, error) {\n'
       '\tif extra, err := h.do(ctx, http.MethodGet, name); err == nil {\n'
       '\t\t_, _ = io.Copy(io.Discard, extra.Body)\n'
       '\t\t_ = extra.Body.Close()\n'
       '\t}\n'
       '\tresp, err := h.do(ctx, http.MethodGet, name)')
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case remote_double_fetch "$HTTPGO" "not one-for-one"

# 5. One byte of the remote body is flipped in the connector. If the integrity
#    check were only comparing lengths, this would sail through.
restore
python3 - "$HTTPGO" <<'MUT'
import sys
p = sys.argv[1]; s = open(p).read()
old = '\tresp, err := h.do(ctx, http.MethodGet, name)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn resp.Body, nil\n}'
assert old in s, "Open return anchor"
new = ('\tresp, err := h.do(ctx, http.MethodGet, name)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n'
       '\treturn corruptReadCloser{resp.Body}, nil\n}\n\n'
       'type corruptReadCloser struct{ io.ReadCloser }\n\n'
       'func (c corruptReadCloser) Read(p []byte) (int, error) {\n'
       '\tn, err := c.ReadCloser.Read(p)\n'
       '\tif n > 0 {\n\t\tp[0] ^= 0xFF\n\t}\n'
       '\treturn n, err\n}')
open(p, 'w').write(s.replace(old, new, 1))
MUT
run_case remote_bytes_corrupt "$HTTPGO" "catalog recorded"

restore
sed -i '$ s/$/,/' "$RESULTS"
if [ "$FAILURES" -eq 0 ]; then
  printf '  {"summary": "PASS", "cases": 5}\n]\n' >> "$RESULTS"
else
  printf '  {"summary": "FAIL", "cases_not_detected": %d}\n]\n' "$FAILURES" >> "$RESULTS"
fi
python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$RESULTS" || echo "WARN: results JSON malformed"
echo "undetected mutations: $FAILURES"
if [ "$FAILURES" -eq 0 ]; then exit 0; fi
exit 1
