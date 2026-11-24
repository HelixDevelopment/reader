#!/usr/bin/env bash
# hrd026_server.sh - start/stop a throwaway helix-server + redis for HRD-026 real-reader interop runs.
# usage: hrd026_server.sh start <helix-server-bin> <state-dir> | stop <state-dir>
# state-dir holds: env (0600, throwaway credentials, NEVER print), lib/ (fixture library), server.pid, server.log
# Loopback only. Redis = disposable podman container hrd026-redis on a high loopback port.
set -eu
cmd=$1; BIN=${2:-}; ST=${3:-}
[ "$cmd" = stop ] && ST=$2
rport=36379; port=38090
case $cmd in
start)
  mkdir -p "$ST/lib"; umask 077
  if [ ! -f "$ST/env" ]; then
    { echo "ADMIN=$(head -c18 /dev/urandom | base64 | tr -dc A-Za-z0-9)"
      echo "OPDS_USER=opds$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n')"
      echo "OPDS_PASS=$(head -c18 /dev/urandom | base64 | tr -dc A-Za-z0-9)"
      echo "KO_USER=ko$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n')"
      echo "KO_PASS=$(head -c12 /dev/urandom | base64 | tr -dc A-Za-z0-9)"
      echo "PORT=$port"; } > "$ST/env"; chmod 600 "$ST/env"
  fi
  . "$ST/env"
  podman container exists hrd026-redis || podman run -d --rm --name hrd026-redis -p 127.0.0.1:$rport:6379 docker.io/library/redis:7.4-alpine >/dev/null
  for _ in $(seq 1 30); do [ "$(podman exec hrd026-redis redis-cli ping 2>/dev/null)" = PONG ] && break; sleep 1; done
  podman exec hrd026-redis redis-cli flushall >/dev/null
  HELIX_ADDRESS=127.0.0.1:$port HELIX_REDIS_URL=redis://127.0.0.1:$rport HELIX_ADMIN_TOKEN="$ADMIN" \
  HELIX_OPDS_USERNAME="$OPDS_USER" HELIX_OPDS_PASSWORD="$OPDS_PASS" HELIX_KOSYNC_REGISTRATION=true HELIX_HTTP3=false \
  HELIX_COVER_CACHE="$ST/covers" HELIX_EXTERNAL_URL=http://127.0.0.1:$port GIN_MODE=release \
    nohup "$BIN" > "$ST/server.log" 2>&1 &
  echo $! > "$ST/server.pid"
  for _ in $(seq 1 60); do curl -fs http://127.0.0.1:$port/readyz >/dev/null 2>&1 && { echo ready; exit 0; }; sleep 0.5; done
  echo "server not ready" >&2; exit 1 ;;
stop)
  [ -f "$ST/server.pid" ] && { kill "$(cat "$ST/server.pid")" 2>/dev/null || true; rm -f "$ST/server.pid"; }
  podman rm -f hrd026-redis >/dev/null 2>&1 || true ;;
esac
