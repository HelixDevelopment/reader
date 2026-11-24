#!/usr/bin/env bash
# hrd026_server.sh - start/stop a throwaway helix-server + redis for HRD-026 real-reader interop runs.
# usage: hrd026_server.sh start <helix-server-bin> <state-dir> | stop <state-dir>
# state-dir holds: env (0600, throwaway credentials, NEVER print), lib/ (fixture library), server.pid, server.bin, server.log
# Loopback only. Redis = disposable podman container hrdiso-hrd026-redis on a high loopback port.
# Hygiene (constitution 11.4.287(C)): every argument is guarded, the state dir must be absolute and under an allowed
# run root (HRD026_RUN_ROOT, default $TMPDIR or /tmp), and stop only signals a pid whose /proc/<pid>/exe is the
# recorded server binary.
set -eu
cmd=${1:?usage: start <bin> <state-dir> | stop <state-dir>}
case $cmd in
  start) BIN=${2:?helix-server binary}; ST=${3:?state-dir} ;;
  stop)  ST=${2:?state-dir}; BIN= ;;
  *) echo "unknown command $cmd" >&2; exit 2 ;;
esac
RUNROOT=${HRD026_RUN_ROOT:-${TMPDIR:-/tmp}}
case $ST in "$RUNROOT"/?*) ;; *) echo "state-dir $ST must be an absolute path under $RUNROOT" >&2; exit 2;; esac
case $ST in *..*) echo "state-dir must not contain .." >&2; exit 2;; esac
RC=hrdiso-hrd026-redis
RIMG=docker.io/library/redis:7.4-alpine   # tag only: image digest UNCONFIRMED (not recorded in any existing evidence)
rport=36379; port=38090
case $cmd in
start)
  case $BIN in /*) ;; *) echo "binary must be absolute" >&2; exit 2;; esac
  [ -x "$BIN" ] || { echo "not executable: $BIN" >&2; exit 2; }
  mkdir -p "${ST:?}/lib"; umask 077
  if [ ! -f "$ST/env" ]; then
    { echo "ADMIN=$(head -c18 /dev/urandom | base64 | tr -dc A-Za-z0-9)"
      echo "OPDS_USER=opds$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n')"
      echo "OPDS_PASS=$(head -c18 /dev/urandom | base64 | tr -dc A-Za-z0-9)"
      echo "KO_USER=ko$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n')"
      echo "KO_PASS=$(head -c12 /dev/urandom | base64 | tr -dc A-Za-z0-9)"
      echo "PORT=$port"; } > "$ST/env"; chmod 600 "$ST/env"
  fi
  . "$ST/env"
  podman container exists "$RC" || podman run -d --rm --name "$RC" -p "127.0.0.1:$rport:6379" "$RIMG" >/dev/null
  for _ in $(seq 1 30); do [ "$(podman exec "$RC" redis-cli ping 2>/dev/null)" = PONG ] && break; sleep 1; done
  podman exec "$RC" redis-cli flushall >/dev/null
  readlink -f -- "$BIN" > "$ST/server.bin"
  HELIX_ADDRESS=127.0.0.1:$port HELIX_REDIS_URL=redis://127.0.0.1:$rport HELIX_ADMIN_TOKEN="$ADMIN" \
  HELIX_OPDS_USERNAME="$OPDS_USER" HELIX_OPDS_PASSWORD="$OPDS_PASS" HELIX_KOSYNC_REGISTRATION=true HELIX_HTTP3=false \
  HELIX_COVER_CACHE="$ST/covers" HELIX_EXTERNAL_URL=http://127.0.0.1:$port GIN_MODE=release \
    nohup "$BIN" > "$ST/server.log" 2>&1 &
  echo $! > "$ST/server.pid"
  for _ in $(seq 1 60); do curl -fs "http://127.0.0.1:$port/readyz" >/dev/null 2>&1 && { echo ready; exit 0; }; sleep 0.5; done
  echo "server not ready" >&2; exit 1 ;;
stop)
  if [ -f "$ST/server.pid" ] && [ -f "$ST/server.bin" ]; then
    pid=$(cat "$ST/server.pid"); want=$(cat "$ST/server.bin")
    case $pid in ''|*[!0-9]*|0|1) echo "refusing pid '$pid'" >&2; pid= ;; esac
    if [ -n "$pid" ]; then
      have=$(readlink -f "/proc/$pid/exe" 2>/dev/null || true)
      if [ -n "$have" ] && [ "$have" = "$want" ]; then kill "$pid" 2>/dev/null || true
      else echo "pid $pid is not our server (exe '$have' != '$want'); not signalling" >&2; fi
    fi
    rm -f "$ST/server.pid"
  fi
  podman rm -f "$RC" >/dev/null 2>&1 || true ;;
esac
