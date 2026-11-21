#!/usr/bin/env bash
# ATM-024 remaining acceptance: image distribution round-trip, and compose execution.
# Scoped to disposable names this script owns and removes; it never touches the
# pre-existing helixqual-pg/helixqual-redis qualification containers or any container
# belonging to another project. Credentials are generated per run into a 0600 scratch
# file and are never echoed.
set -uo pipefail
cd "$(dirname "$0")/../../.." || exit 1
EVID="evidence/campaign-2026-11-20/containers"
LOG="$EVID/distribution.log"
ENVFILE="$(mktemp /tmp/helix-dist-env.XXXXXX)"
chmod 600 "$ENVFILE"
PROJ=helixqual-dist
BUILT=localhost/helixqual-dist-helix:v1
REGPORT=15000
APP_PORT=18181
PULLED_PORT=18182
REG=helixqual-dist-registry
PUSHED=127.0.0.1:$REGPORT/helix-reader:v1
LIB=$(mktemp -d /tmp/helix-dist-library.XXXXXX)
TLS=$(mktemp -d /tmp/helix-dist-tls.XXXXXX)
PGNAME="${PROJ}_postgres_1"

cat >"$ENVFILE" <<EOF
HELIX_DB_PASSWORD=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
HELIX_ADMIN_TOKEN=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
EOF
TOKEN=$(grep HELIX_ADMIN_TOKEN "$ENVFILE" | cut -d= -f2)

cleanup() {
  {
    echo "--- cleanup ---"
    podman-compose -f deploy/docker-compose.yml down -v --remove-orphans 2>&1 | tail -3
    podman rm -f $REG helixqual-dist-pulled >/dev/null 2>&1
    podman rmi -f "$BUILT" "$PUSHED" localhost/helixqual-dist-helix:v2 >/dev/null 2>&1
    rm -f "$ENVFILE"
    rmdir "$LIB" "$TLS" 2>&1
  } >>"$LOG" 2>&1
}
trap cleanup EXIT
export COMPOSE_PROJECT_NAME=$PROJ HELIX_PUBLISH_PORT=$APP_PORT HELIX_IMAGE="$BUILT" \
       HELIX_LIBRARY_PATH=$LIB HELIX_TLS_DIR=$TLS HELIX_HTTP3=false

{
  echo "# ATM-024 distribution + compose parity, $(date -Is)"
  echo "# podman: $(podman --version)"
  echo "## step 1: build the shipped recipe"
  time podman build -f server/Dockerfile -t "$BUILT" . 2>&1 | tail -4
  BUILT_ID=$(podman inspect "$BUILT" --format '{{.Id}}')
  echo "built_config_id=$BUILT_ID"
  podman inspect "$BUILT" --format 'created={{.Created}} size_bytes={{.Size}} user={{.Config.User}} entrypoint={{.Config.Entrypoint}}'
} >>"$LOG" 2>&1

{
  echo "## step 2: local registry, push, and a pull that cannot be served from cache"
  podman rm -f $REG >/dev/null 2>&1
  podman run -d --name $REG -p 127.0.0.1:$REGPORT:5000 docker.io/library/registry:2 >/dev/null 2>&1
  for _ in $(seq 1 40); do curl -fsS -o /dev/null "http://127.0.0.1:$REGPORT/v2/" && break; sleep 0.5; done
  echo "registry_v2_endpoint=$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:$REGPORT/v2/)"
  podman tag "$BUILT" "$PUSHED"
  echo "--- push ---"
  podman push --tls-verify=false "$PUSHED" 2>&1 | tail -3
  echo "catalog=$(curl -s http://127.0.0.1:$REGPORT/v2/_catalog)"
  echo "tags=$(curl -s http://127.0.0.1:$REGPORT/v2/helix-reader/tags/list)"
  PUSH_DIGEST=$(curl -sI -H 'Accept: application/vnd.oci.image.index.v1+json,application/vnd.docker.distribution.manifest.v2+json' \
    "http://127.0.0.1:$REGPORT/v2/helix-reader/manifests/v1" | grep -i '^docker-content-digest' | tr -d '\r' | awk '{print $2}')
  echo "push_manifest_digest=$PUSH_DIGEST"
  echo "--- deleting every local copy (both tags) so the pull must come from the registry ---"
  podman rmi -f "$PUSHED" "$BUILT" 2>&1 | tail -2
  echo "local_helixqual_images_after_delete=$(podman images --format '{{.Repository}}:{{.Tag}}' | grep -cE 'helixqual-dist|127.0.0.1:15000' || true)"
  echo "--- pull ---"
  podman pull --tls-verify=false "$PUSHED" 2>&1 | tail -2
  PULL_DIGEST=$(curl -sI -H 'Accept: application/vnd.oci.image.index.v1+json,application/vnd.docker.distribution.manifest.v2+json' \
    "http://127.0.0.1:$REGPORT/v2/helix-reader/manifests/v1" | grep -i '^docker-content-digest' | tr -d '\r' | awk '{print $2}')
  PULLED_ID=$(podman inspect "$PUSHED" --format '{{.Id}}' 2>/dev/null)
  echo "pull_manifest_digest=$PULL_DIGEST"
  echo "pulled_config_id=${PULLED_ID:-none}"
  echo "digest_match=$([ -n "$PUSH_DIGEST" ] && [ "$PUSH_DIGEST" = "$PULL_DIGEST" ] && echo yes || echo no)"
  echo "config_id_matches_build=$([ -n "$BUILT_ID" ] && [ "$BUILT_ID" = "${PULLED_ID:-}" ] && echo yes || echo no)"
} >>"$LOG" 2>&1

{
  echo "## step 3: run the pulled artifact, not the local build"
  podman run -d --name helixqual-dist-pulled -p 127.0.0.1:$PULLED_PORT:8080 \
    --tls-verify=false -e HELIX_ADDRESS=0.0.0.0:8080 -e HELIX_ADMIN_TOKEN="$TOKEN" \
    -e HELIX_COVER_CACHE=/tmp/covers "$PUSHED" >/dev/null 2>&1
  READY=no
  for _ in $(seq 1 60); do
    [ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PULLED_PORT/healthz")" = "200" ] && { READY=yes; break; }
    sleep 1
  done
  echo "pulled_healthz_ready=$READY"
  echo "pulled_healthz=$(curl -s "http://127.0.0.1:$PULLED_PORT/healthz" | head -c 100)"
  echo "pulled_works=$(curl -s -u "admin:$TOKEN" "http://127.0.0.1:$PULLED_PORT/v1/works?limit=2" | head -c 140)"
  echo "pulled_opds20=$(curl -s -o /dev/null -w '%{content_type} http=%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$PULLED_PORT/opds/2.0")"
  echo "pulled_opds12=$(curl -s -o /dev/null -w '%{content_type} http=%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$PULLED_PORT/opds/1.2")"
  echo "runs_as=$(podman exec helixqual-dist-pulled id -un 2>&1)"
  echo "pulled_image_used=$(podman inspect helixqual-dist-pulled --format '{{.Image}}' 2>/dev/null)"
  podman logs helixqual-dist-pulled 2>&1 | head -4
  podman rm -f helixqual-dist-pulled >/dev/null 2>&1
} >>"$LOG" 2>&1

{
  echo "## step 4: compose execution parity (podman-compose; docker-compose is not installed on this host)"
  podman build -f server/Dockerfile -t "$BUILT" . >/dev/null 2>&1
  echo "rebuild_for_compose_exit=$?"
  set -a; . "$ENVFILE"; set +a
  podman-compose -f deploy/docker-compose.yml up -d 2>&1 | tail -4
  READY=no
  for _ in $(seq 1 90); do
    [ "$(curl -s -o /dev/null -w '%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/healthz")" = "200" ] && { READY=yes; break; }
    sleep 2
  done
  echo "compose_healthz_ready=$READY"
  echo "compose_healthz=$(curl -s -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/healthz" | head -c 100)"
  echo "compose_readyz=$(curl -s -w ' http=%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/readyz" | head -c 220)"
  echo "compose_works=$(curl -s -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/v1/works?limit=3" | head -c 160)"
  echo "compose_opds12=$(curl -s -o /dev/null -w 'content_type=%{content_type} http=%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/opds/1.2")"
  echo "compose_opds20=$(curl -s -o /dev/null -w 'content_type=%{content_type} http=%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/opds/2.0")"
  echo "compose_statuses=$(podman ps --filter name=$PROJ --format '{{.Names}}={{.Status}}' | tr '\n' ';')"
  echo "compose_scoped_volumes=$(podman volume ls --format '{{.Name}}' | grep -cE "^${PROJ}_" || true)"
  echo "--- stack-internal Postgres reached by the app (proves HELIX_POSTGRES_DSN wiring) ---"
  MIG="not-run"
  for _ in $(seq 1 30); do
    MIG=$(podman exec $PGNAME psql -U helix -d helix -Atc 'select string_agg(version::text, chr(44) order by version) from helix_schema_migrations' 2>&1 | head -2)
    case "$MIG" in
      *rror*|*exist*|*not-run*) sleep 2 ;;
      *) break ;;
    esac
  done
  echo "compose_migrations=$MIG"
  echo "compose_works_table=$(podman exec $PGNAME psql -U helix -d helix -Atc "select count(*) from information_schema.tables where table_name='works'" 2>&1 | head -2)"
  echo "compose_redis_ping=$(podman exec ${PROJ}_redis_1 redis-cli ping 2>&1 | head -1)"
  echo "--- restart persistence: recreate containers, data must survive ---"
  podman-compose -f deploy/docker-compose.yml restart 2>&1 | tail -3
  sleep 10
  echo "after_restart_readyz=$(curl -s -w ' http=%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/readyz" | head -c 220)"
  echo "after_restart_migrations=$(podman exec $PGNAME psql -U helix -d helix -Atc 'select count(*) from helix_schema_migrations' 2>&1 | head -2)"
  echo "after_restart_volumes=$(podman volume ls --format '{{.Name}}' | grep -cE "^${PROJ}_" || true)"
  echo "--- upgrade and rollback by tag, same stack ---"
  podman tag "$BUILT" localhost/helixqual-dist-helix:v2
  HELIX_IMAGE=localhost/helixqual-dist-helix:v2 podman-compose -f deploy/docker-compose.yml up -d helix 2>&1 | tail -2
  sleep 8
  echo "upgraded_image=$(podman inspect ${PROJ}_helix_1 --format '{{.Config.Image}}' 2>/dev/null)"
  echo "upgraded_healthz=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/healthz")"
  HELIX_IMAGE=$BUILT podman-compose -f deploy/docker-compose.yml up -d helix 2>&1 | tail -2
  sleep 8
  echo "rolled_back_image=$(podman inspect ${PROJ}_helix_1 --format '{{.Config.Image}}' 2>/dev/null)"
  echo "rolled_back_healthz=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/healthz")"
} >>"$LOG" 2>&1

{
  echo "## step 5: rejection probes (needle checks that the stack is not vacuously up)"
  echo "unauth_works=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APP_PORT/v1/works")"
  echo "wrong_token_works=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:wrong-token-needle" "http://127.0.0.1:$APP_PORT/v1/works")"
  echo "good_token_works=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:$TOKEN" "http://127.0.0.1:$APP_PORT/v1/works")"
  echo "published_ports=$(podman port ${PROJ}_helix_1 2>/dev/null | tr '\n' ';')"
} >>"$LOG" 2>&1

echo "DONE"
