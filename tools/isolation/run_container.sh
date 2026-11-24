#!/usr/bin/env bash
# run_container.sh - the ONLY sanctioned way to run GUI/E2E tests: a rootless, resource-capped podman
# container with a PRIVATE Xvfb (image entrypoint). See docs/test-isolation.md.
# usage: run_container.sh [-n hrdiso-NAME] [-m MEM] [-c CPUS] [-t SECONDS] [-i image] [-b SOCKDIR] [-d]
#                         [-r src:dst]... [-w src:dst]... [-e K=V]... -- cmd...
#   -b SOCKDIR  directory holding the ONE unix socket (h.sock) of the test server proxy; mounted at /run/helixsock and
#               bridged to 127.0.0.1:38191 INSIDE the container. Without -b the container has no connectivity at all.
#   -p FILE     also write the container's isolation facts (podman inspect subset, env names only) to FILE. FILE must be a HOST-ONLY path: absolute,
#               canonical, its directory a real (non-link) directory that is NOT inside any rw-bound source (review round 3: a container could
#               otherwise rewrite the proof). Publish it into the evidence dir AFTER the container stopped (safe_io put).
#   -t          maximum lifetime in seconds (podman --timeout; default 1800): an orphaned container dies by itself
#   -r / -w     read-only / read-write bind; sources must pass tools/isolation/bindcheck.py (allow-list)
#   -e          explicit container env (the host env is never inherited)
# Guarantees: --network none (no host loopback, no LAN, no internet); no --net=host / --privileged; all caps dropped;
# no-new-privileges; memory+swap cap; pids cap; positive oom_score_adj; refuses host display/session/input mounts/env;
# the in-container launch.sh asserts at runtime that no host display/session/socket is present (exit 65).
# Container names must start hrdiso- or hrdisorev-; stale ones from a killed runner are reaped on every start.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)
NAME=hrdiso-$$; MEM=2g; CPUS=2; LIFE=1800; IMG=localhost/helix-gui-test:1; BINDS=(); DSTS=(); ENVS=(); DETACH=(); BRIDGE=; PROOF=; RWSRC=()
while getopts "dn:m:c:t:i:b:p:r:w:e:" o; do case $o in
  d) DETACH=(-d);; n) NAME=$OPTARG;; m) MEM=$OPTARG;; c) CPUS=$OPTARG;; t) LIFE=$OPTARG;; i) IMG=$OPTARG;; b) BRIDGE=$OPTARG;; p) PROOF=$OPTARG;;
  r) IFS=: read -r s d <<<"$OPTARG"; case " ${DSTS[*]:-} " in *" $d "*) echo "run_container: duplicate destination $d" >&2; exit 64;; esac
     real=$(python3 -I "$HERE/bindcheck.py" ro "$s" "$d") || exit 64; DSTS+=("$d"); BINDS+=(-v "$real:$d:ro");;
  w) IFS=: read -r s d <<<"$OPTARG"; case " ${DSTS[*]:-} " in *" $d "*) echo "run_container: duplicate destination $d" >&2; exit 64;; esac
     real=$(python3 -I "$HERE/bindcheck.py" rw "$s" "$d") || exit 64; DSTS+=("$d"); BINDS+=(-v "$real:$d:rw"); RWSRC+=("$real");;
  e) ENVS+=(-e "$OPTARG");;
  *) exit 2;; esac; done; shift $((OPTIND-1)); [ "${1:-}" = -- ] && shift
[ $# -gt 0 ] || { echo "no command" >&2; exit 2; }
case $NAME in hrdiso-*|hrdisorev-*) ;; *) echo "run_container: name must start with hrdiso-" >&2; exit 64;; esac
# env passthrough is refused BY NAME (a bare name makes podman copy the HOST value); only explicit NAME=VALUE with a plain name is accepted
for b in "${ENVS[@]:-}"; do [ -z "$b" ] && continue; [ "$b" = -e ] && continue; n=${b%%=*}; N=${n^^}
  case $b in *=*) ;; *) echo "run_container: env must be explicit NAME=VALUE (a bare name passes the host value through): $b" >&2; exit 64;; esac
  case $n in ''|[0-9]*|*[!A-Za-z0-9_]*) echo "run_container: invalid env name: $n" >&2; exit 64;; esac
  case $N in DISPLAY*|*_DISPLAY|WAYLAND*|XAUTHORITY*|ICEAUTHORITY*|*DBUS*|XDG_*|SSH_AUTH_SOCK|SSH_*|PULSE*|PIPEWIRE*|GNOME_*|KDE_*|GTK_*|QT_QPA*|DESKTOP_SESSION|SESSION_MANAGER|GPG_AGENT*|GDK_*) echo "run_container: refusing host display/session env: $n" >&2; exit 64;; esac; done  # isolation: denylist
case $IMG in -*|*[!A-Za-z0-9_./:@-]*|'') echo "run_container: invalid image name: $IMG" >&2; exit 64;; esac
# strict positive ranges with documented caps (review round 2): 0 / "0k" / "0." / "1.2.3" mean unlimited or garbage to podman, and an unbounded
# value defeats the resource cap. -m 64m..16g (unit m or g required), -c 0.10..8 (at most two decimals), -t 1..14400 s (4 h).
[[ $MEM =~ ^([1-9][0-9]{0,5})([mMgG])$ ]] || { echo "run_container: invalid -m $MEM (positive integer + m or g)" >&2; exit 64; }
mem_n=${BASH_REMATCH[1]}; mem_mul=1; case ${BASH_REMATCH[2]} in g|G) mem_mul=1024;; esac; mem_mib=$((mem_n * mem_mul))
{ [ "$mem_mib" -ge 64 ] && [ "$mem_mib" -le 16384 ]; } || { echo "run_container: -m $MEM outside 64m..16g" >&2; exit 64; }
[[ $CPUS =~ ^([0-9]{1,2})(\.([0-9]{1,2}))?$ ]] || { echo "run_container: invalid -c $CPUS (number with at most two decimals)" >&2; exit 64; }
frac=${BASH_REMATCH[3]:-0}; [ ${#frac} -eq 2 ] || frac=${frac}0; [ "$frac" = 00 ] && frac=0
cpu_centi=$(( 10#${BASH_REMATCH[1]} * 100 + 10#$frac ))
{ [ "$cpu_centi" -ge 10 ] && [ "$cpu_centi" -le 800 ]; } || { echo "run_container: -c $CPUS outside 0.10..8" >&2; exit 64; }
[[ $LIFE =~ ^[1-9][0-9]{0,4}$ ]] && [ "$LIFE" -le 14400 ] || { echo "run_container: -t $LIFE outside 1..14400 seconds" >&2; exit 64; }
OWNER=$$; [ ${#DETACH[@]} -eq 0 ] || OWNER=$PPID   # detached: the runner exits at once, so the caller owns the container
LAUNCH=(sh /opt/isoinside/launch.sh)
if [ -n "$BRIDGE" ]; then
  real=$(python3 -I "$HERE/bindcheck.py" rw "$BRIDGE" /run/helixsock --bridge) || exit 64
  BINDS+=(-v "$real:/run/helixsock:rw"); RWSRC+=("$real"); LAUNCH+=(--bridge 38191 /run/helixsock/h.sock)
fi
# -p: the proof lives where the container cannot write (review round 3, R3-I2)
if [ -n "$PROOF" ]; then
  case $PROOF in /*) ;; *) echo "run_container: -p must be an absolute path: $PROOF" >&2; exit 64;; esac
  [[ $PROOF =~ ^/[A-Za-z0-9_./-]+$ ]] && [ "$(python3 -I -c 'import os,sys;print(os.path.normpath(sys.argv[1]))' "$PROOF")" = "$PROOF" ] || { echo "run_container: -p must be a canonical path of [A-Za-z0-9_./-]: $PROOF" >&2; exit 64; }
  PDIR=$(dirname "$PROOF"); [ -d "$PDIR" ] && [ ! -L "$PDIR" ] && [ "$(realpath "$PDIR")" = "$PDIR" ] || { echo "run_container: -p directory $PDIR must be an existing real directory (no symlink in its path)" >&2; exit 64; }
  for rs in "${RWSRC[@]:-}"; do [ -z "$rs" ] && continue
    case "$PDIR/" in "$rs"/*) echo "run_container: -p $PROOF lies inside the container-writable $rs: keep the proof in a host-only directory and publish it after the run" >&2; exit 64;; esac
  done
fi
podman image exists "$IMG" || { echo "image $IMG missing; build: podman build -t $IMG submodules/containers/images/gui-test" >&2; exit 66; }
"$HERE/reap.sh" >/dev/null || true
[ -z "$PROOF" ] || { python3 -I "$HERE/container_proof.py" "$NAME" "$PROOF" >/dev/null 2>&1 & }
RUN=(podman run "${DETACH[@]}" --rm --name "$NAME" --network none --timeout "$LIFE" \
  --memory "$MEM" --memory-swap "$MEM" --cpus "$CPUS" --pids-limit 512 --oom-score-adj 500 \
  --cap-drop ALL --security-opt no-new-privileges --userns=keep-id --shm-size 512m \
  --tmpfs "/tmp:rw,size=512m" --ulimit fsize=2147483648:2147483648 --label org.helix.isolation=gui-test --label "org.helix.owner-pid=$OWNER" --label "org.helix.lease-ts=$(date +%s)" \
  -v "$REPO/qa/isolation/inside:/opt/isoinside:ro" \
  "${BINDS[@]}" "${ENVS[@]}" "$IMG" "${LAUNCH[@]}" -- "$@")
# review round 4 (R4-B2): in ATTACHED mode the container's stdout/stderr are container-controlled bytes: they pass through sanitize.py (terminal sequences removed,
# every line prefixed "[container] ") before they reach the operator's terminal, a tee or a log; podman's exit status is preserved via PIPESTATUS.
# Detached mode prints only the container id (host-generated) and keeps the exec.
if [ ${#DETACH[@]} -gt 0 ]; then exec "${RUN[@]}"; fi
"${RUN[@]}" 2>&1 | python3 -I "$HERE/sanitize.py"
exit "${PIPESTATUS[0]}"
