#!/usr/bin/env bash
# start_server.sh - host-side (NO GUI) helix-server + disposable redis for isolated container runs.
# usage: start_server.sh start <helix-server-bin> <state-dir> <fixture-dir> | stop <state-dir> | ids
# Loopback only. Redis and the server are PER RUN: container hrdiso-redis-<runid>, free loopback ports picked per run, recorded
# in <state-dir>/run.env (RID RNAME RPORT SPORT) - so parallel suites never share (or flush) each other's redis. Containers run with --network none and reach the
# server ONLY through the unix socket <state-dir>/sock/h.sock (logging proxy -> server), handed in via
# run_container.sh -b <state-dir>/sock and bridged to 127.0.0.1:38191 inside the container. Credentials are throwaway,
# in <state-dir>/env (0600). The server has no access log, so the proxy records method/path/status/User-Agent
# (<state-dir>/access.log) and KOSync bodies (<state-dir>/access.log.bodies, secrets redacted).
# Bounded lifetime: server/proxy run under `timeout 3600`, redis under podman --timeout 3600; stale hrdiso-* leftovers
# of a killed run are reaped at start (tools/isolation/reap.sh).
set -eu
cmd=$1; HERE=$(cd "$(dirname "$0")" && pwd)
pport=38191
newids() {  # RID RNAME RPORT SPORT : fresh run id, two free loopback ports (checked by binding)
  RID=$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n')
  set -- $(python3 -I - <<'PY'
import socket
ps = []
for _ in range(2):
    s = socket.socket(); s.bind(("127.0.0.1", 0)); ps.append(s)
print(*[s.getsockname()[1] for s in ps])
PY
)
  RNAME=hrdiso-redis-$RID; RPORT=$1; SPORT=$2
}
case $cmd in
ids) newids; echo "$RID $RNAME $RPORT $SPORT"; exit 0 ;;
esac
case $cmd in
start)
  BIN=$2; ST=$3; FIX=$4; mkdir -p "$ST/sock"; umask 077; chmod 700 "$ST/sock"
  "$HERE/reap.sh" >/dev/null || true
  # N4: the OWNER FILE exists (naming this live script) BEFORE any container is created, and the container carries a lease
  # timestamp; the reaper honours a startup grace window for both. server.pid is rewritten with the server's pid later.
  echo $$ > "$ST/server.pid"
  newids; port=$SPORT; rport=$RPORT; rname=$RNAME
  printf 'RID=%s\nRNAME=%s\nRPORT=%s\nSPORT=%s\n' "$RID" "$RNAME" "$RPORT" "$SPORT" > "$ST/run.env"; echo "$RNAME" > "$ST/redis.name"
  [ ${#ST} -lt 80 ] || { echo "state dir path too long for a unix socket" >&2; exit 2; }
  [ -f "$ST/env" ] || { { echo "ADMIN=$(head -c18 /dev/urandom|base64|tr -dc A-Za-z0-9)"
    echo "OPDS_USER=opds$(head -c4 /dev/urandom|od -An -tx1|tr -d ' \n')"
    echo "OPDS_PASS=$(head -c18 /dev/urandom|base64|tr -dc A-Za-z0-9)"; } > "$ST/env"; }
  # MODE=hq: fixed credentials the HelixQA banks hard-code (throwaway, loopback only); MODE=noauth: no OPDS Basic
  # (what the web bank's steps assume); default: random throwaway credentials.
  # MODE=hq: the OPDS reader credentials the HelixQA banks hard-code (read-only catalogue access, throwaway); MODE=noauth: no OPDS
  # Basic. The ADMIN token is ALWAYS random per run (never a fixed literal) and never enters a container; the banks' own
  # hard-coded admin literal is mapped by the proxy (policy helixqa, --rewrite-bearer) on two routes only.
  ADMIN_RND=$(head -c18 /dev/urandom|base64|tr -dc A-Za-z0-9)
  case ${MODE:-random} in
    hq) printf 'ADMIN=%s\nOPDS_USER=hq-reader\nOPDS_PASS=hq-password\n' "$ADMIN_RND" > "$ST/env" ;;
    noauth) printf 'ADMIN=%s\nOPDS_USER=\nOPDS_PASS=\n' "$ADMIN_RND" > "$ST/env" ;;
  esac

  . "$ST/env"
  podman run -d --rm --name $rname --timeout 3600 --label org.helix.isolation=redis --label "org.helix.owner-file=$ST/server.pid" --label "org.helix.lease-ts=$(date +%s)" --memory 128m --pids-limit 64 -p 127.0.0.1:$rport:6379 docker.io/library/redis:7.4-alpine >/dev/null
  for _ in $(seq 1 30); do [ "$(podman exec $rname redis-cli ping 2>/dev/null)" = PONG ] && break; sleep 1; done
  podman exec $rname redis-cli flushall >/dev/null
  HELIX_ADDRESS=127.0.0.1:$port HELIX_REDIS_URL=redis://127.0.0.1:$rport HELIX_ADMIN_TOKEN="$ADMIN" \
  HELIX_OPDS_USERNAME="${OPDS_USER:-}" HELIX_OPDS_PASSWORD="${OPDS_PASS:-}" HELIX_KOSYNC_REGISTRATION=true HELIX_HTTP3=false \
  HELIX_COVER_CACHE="$ST/covers" HELIX_EXTERNAL_URL=${EXTERNAL_URL:-http://127.0.0.1:$pport} GIN_MODE=release \
    nohup timeout 3600 "$BIN" > "$ST/server.log" 2>&1 & echo $! > "$ST/server.pid"
  PROXY_ADMIN_TOKEN="$ADMIN" nohup timeout 3600 python3 -I "$HERE/access_proxy.py" --policy "${PROXY_POLICY:-common}" ${PROXY_REWRITE:+--rewrite-bearer "$PROXY_REWRITE"} "unix:$ST/sock/h.sock" 127.0.0.1 $port "$ST/access.log" > /dev/null 2>&1 & echo $! > "$ST/proxy.pid"
  for _ in $(seq 1 60); do curl -fs http://127.0.0.1:$port/readyz >/dev/null 2>&1 && break; sleep 0.5; done
  curl -fs http://127.0.0.1:$port/readyz >/dev/null || { echo "server not ready" >&2; exit 1; }
  A="Authorization: Bearer $ADMIN"
  curl -fs -XPOST http://127.0.0.1:$port/v1/sources -H "$A" -d "{\"id\":\"${SOURCE_ID:-iso-lib}\",\"protocol\":\"local\",\"root\":\"$FIX\"}" >/dev/null
  curl -fs -XPOST http://127.0.0.1:$port/v1/scans -H "$A" -d "{\"source_id\":\"${SOURCE_ID:-iso-lib}\",\"prefix\":\"\"}" >/dev/null
  for _ in $(seq 1 40); do curl -fs http://127.0.0.1:$port/v1/scans -H "$A"|grep -q '"state":"succeeded"' && break; sleep 0.25; done
  echo ready ;;
stop)
  ST=$2
  for f in server.pid proxy.pid; do [ -f "$ST/$f" ] && { kill "$(cat "$ST/$f")" 2>/dev/null || true; rm -f "$ST/$f"; }; done
  [ -f "$ST/redis.name" ] && { rname=$(cat "$ST/redis.name"); case $rname in hrdiso-redis-*) podman rm -f "$rname" >/dev/null 2>&1 || true ;; esac; } ;;
esac
