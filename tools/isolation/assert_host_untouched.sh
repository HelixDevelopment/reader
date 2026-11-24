#!/usr/bin/env bash
# Repo gate for the host-isolation policy.
#   assert_host_untouched.sh            lint tests (guard + evader matrix + pinned waivers + bind allow-list + proxy route policy) and the
#                                       repo scan   (fast, no containers)
#   assert_host_untouched.sh --full     additionally the container tests: network isolation (only the proxy socket reachable),
#                                       runtime checks, kill -9 reaper (needs rootless podman + localhost/helix-gui-test:1)
# The lint is a lint; the container (run_container.sh) is the boundary.
set -eu
H=$(cd "$(dirname "$0")" && pwd); R=$(cd "$H/../.." && pwd)
python3 -I -m unittest discover -s "$H/guard" >&2
python3 -I -m unittest discover -s "$H/tests" -p 'test_guard_matrix.py' >&2
python3 -I -m unittest discover -s "$H/tests" -p 'test_bindcheck.py' >&2
python3 -I -m unittest discover -s "$H/tests" -p 'test_waivers.py' >&2
python3 -I -m unittest discover -s "$H/tests" -p 'test_proxy_policy.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round1.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round2.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round3.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round4.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round5.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round6.py' >&2
python3 -I -W ignore -m unittest discover -s "$H/tests" -p 'test_review_round8.py' >&2
python3 -I "$H/check_raw_consumers.py" "$R" >&2       # R5-B1(d)/R6: defence in depth - attached run_container.sh calls need -o/-d, no raw_* outside rawrun.py, no hand-captured podman output
if [ "${1:-}" = --full ]; then
  # the real-server half of the N1 test (admin routes refused, no source created) needs a helix-server binary; say so when it is absent
  : "${HELIX_SERVER_BIN:=$HOME/.cache/claude-tmp/helix-server-fresh}"; [ -x "$HELIX_SERVER_BIN" ] || { echo "NOTICE: HELIX_SERVER_BIN not set/executable - real-server N1 test will SKIP" >&2; unset HELIX_SERVER_BIN; }
  export HELIX_SERVER_BIN
  python3 -I -m unittest discover -s "$H/tests" -p 'test_network_isolation.py' -v >&2
  python3 -I -m unittest discover -s "$H/tests" -p 'test_runtime_checks.py' >&2
  python3 -I -m unittest discover -s "$H/tests" -p 'test_reaper.py' >&2
  shift
fi
exec python3 -I "$H/guard/assert_host_untouched.py" "${1:-$R}"
