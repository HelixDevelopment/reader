#!/usr/bin/env bash
# run_container.sh - the ONLY sanctioned way to run GUI/E2E tests: a rootless, resource-capped podman
# container with a PRIVATE Xvfb (image entrypoint). See docs/test-isolation.md.
# usage: run_container.sh [-n hrdiso-NAME] [-m MEM] [-c CPUS] [-t SECONDS] [-i image] [-b SOCKDIR] [-d]
#                         [-r src:dst]... [-w src:dst]... [-e K=V]... -- cmd...
#   -b SOCKDIR  directory holding the ONE unix socket (h.sock) of the test server proxy; mounted at /run/helixsock and
#               bridged to 127.0.0.1:38191 INSIDE the container. Without -b the container has no connectivity at all.
#   -p FILE     also write the container's isolation facts (podman inspect subset, env names only) to FILE
#   -t          maximum lifetime in seconds (podman --timeout; default 1800): an orphaned container dies by itself
#   -r / -w     read-only / read-write bind; sources must pass tools/isolation/bindcheck.py (allow-list)
#   -e          explicit container env (the host env is never inherited)
# Guarantees: --network none (no host loopback, no LAN, no internet); no --net=host / --privileged; all caps dropped;
# no-new-privileges; memory+swap cap; pids cap; positive oom_score_adj; refuses host display/session/input mounts/env;
# the in-container launch.sh asserts at runtime that no host display/session/socket is present (exit 65).
# Container names must start hrdiso- or hrdisorev-; stale ones from a killed runner are reaped on every start.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)
NAME=hrdiso-$$; MEM=2g; CPUS=2; LIFE=1800; IMG=localhost/helix-gui-test:1; BINDS=(); DSTS=(); ENVS=(); DETACH=(); BRIDGE=; PROOF=
while getopts "dn:m:c:t:i:b:p:r:w:e:" o; do case $o in
  d) DETACH=(-d);; n) NAME=$OPTARG;; m) MEM=$OPTARG;; c) CPUS=$OPTARG;; t) LIFE=$OPTARG;; i) IMG=$OPTARG;; b) BRIDGE=$OPTARG;; p) PROOF=$OPTARG;;
  r) IFS=: read -r s d <<<"$OPTARG"; case " ${DSTS[*]:-} " in *" $d "*) echo "run_container: duplicate destination $d" >&2; exit 64;; esac
     real=$(python3 -I "$HERE/bindcheck.py" ro "$s" "$d") || exit 64; DSTS+=("$d"); BINDS+=(-v "$real:$d:ro");;
  w) IFS=: read -r s d <<<"$OPTARG"; case " ${DSTS[*]:-} " in *" $d "*) echo "run_container: duplicate destination $d" >&2; exit 64;; esac
     real=$(python3 -I "$HERE/bindcheck.py" rw "$s" "$d") || exit 64; DSTS+=("$d"); BINDS+=(-v "$real:$d:rw");;
  e) ENVS+=(-e "$OPTARG");;
  *) exit 2;; esac; done; shift $((OPTIND-1)); [ "${1:-}" = -- ] && shift
[ $# -gt 0 ] || { echo "no command" >&2; exit 2; }
case $NAME in hrdiso-*|hrdisorev-*) ;; *) echo "run_container: name must start with hrdiso-" >&2; exit 64;; esac
for b in "${ENVS[@]:-}"; do case $b in
  *DISPLAY=*|*WAYLAND*|*XAUTHORITY*|*DBUS*|*XDG_RUNTIME_DIR*|*XDG_SESSION*|*SSH_AUTH_SOCK*) echo "run_container: refusing host display/session env: $b" >&2; exit 64;; esac; done  # isolation: denylist
OWNER=$$; [ ${#DETACH[@]} -eq 0 ] || OWNER=$PPID   # detached: the runner exits at once, so the caller owns the container
LAUNCH=(sh /opt/isoinside/launch.sh)
if [ -n "$BRIDGE" ]; then
  real=$(python3 -I "$HERE/bindcheck.py" rw "$BRIDGE" /run/helixsock --bridge) || exit 64
  BINDS+=(-v "$real:/run/helixsock:rw"); LAUNCH+=(--bridge 38191 /run/helixsock/h.sock)
fi
podman image exists "$IMG" || { echo "image $IMG missing; build: podman build -t $IMG submodules/containers/images/gui-test" >&2; exit 66; }
"$HERE/reap.sh" >/dev/null || true
[ -z "$PROOF" ] || { python3 -I "$HERE/container_proof.py" "$NAME" "$PROOF" >/dev/null 2>&1 & }
exec podman run "${DETACH[@]}" --rm --name "$NAME" --network none --timeout "$LIFE" \
  --memory "$MEM" --memory-swap "$MEM" --cpus "$CPUS" --pids-limit 512 --oom-score-adj 500 \
  --cap-drop ALL --security-opt no-new-privileges --userns=keep-id --shm-size 512m \
  --tmpfs /tmp:rw,size=512m --label org.helix.isolation=gui-test --label "org.helix.owner-pid=$OWNER" --label "org.helix.lease-ts=$(date +%s)" \
  -v "$REPO/qa/isolation/inside:/opt/isoinside:ro" \
  "${BINDS[@]}" "${ENVS[@]}" "$IMG" "${LAUNCH[@]}" -- "$@"
