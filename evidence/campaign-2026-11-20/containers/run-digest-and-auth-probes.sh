#!/usr/bin/env bash
# ATM-024 residual probes, after run-distribution-and-compose.sh:
#  (a) manifest-digest equality across push and a cache-free pull (the earlier run
#      proved config-id equality but its HEAD header probe returned empty);
#  (b) rejection needle proofs on the containerized read surface, which the earlier
#      run could not show because HELIX_OPDS_USERNAME/HELIX_OPDS_PASSWORD were unset
#      and the server then serves the read API unauthenticated by design.
# Disposable names only; removed at exit. Credentials generated per run, never echoed.
set -uo pipefail
cd "$(dirname "$0")/../../.." || exit 1
LOG="evidence/campaign-2026-11-20/containers/distribution.log"
ENVFILE="$(mktemp /tmp/helix-auth-env.XXXXXX)"; chmod 600 "$ENVFILE"
PROJ=helixqual-dist
BUILT=localhost/helixqual-dist-helix:v1
PUSHED=localhost/helixqual-dist-helix:v3
REG=helixqual-dist-registry
REGPORT=15001
APPPORT=18183
LIB=$(mktemp -d /tmp/helix-auth-library.XXXXXX)

cat >"$ENVFILE" <<EOF
ADMIN=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
OPDSU=reader-needle
OPDSP=$(head -c 18 /dev/urandom | od -An -tx1 | tr -d ' \n')
EOF
ADMIN=$(grep '^ADMIN=' "$ENVFILE" | cut -d= -f2)
OPDSU=$(grep '^OPDSU=' "$ENVFILE" | cut -d= -f2)
OPDSP=$(grep '^OPDSP=' "$ENVFILE" | cut -d= -f2)

trap '{
  echo "--- cleanup ---"
  podman rm -f helixqual-dist-auth $REG >/dev/null 2>&1
  podman rmi -f "$BUILT" "$PUSHED" >/dev/null 2>&1
  rm -f "$ENVFILE"; rmdir "$LIB"
}' EXIT

{
  echo ""
  echo "## step 6: manifest digest equality on a cache-free pull (follow-up, $(date -Is))"
  podman build -f server/Dockerfile -t "$BUILT" . >/dev/null 2>&1
  echo "build_exit=$?"
  BUILT_ID=$(podman inspect "$BUILT" --format '{{.Id}}')
  echo "built_config_id=$BUILT_ID"
  podman rm -f $REG >/dev/null 2>&1
  podman run -d --name $REG -p 127.0.0.1:$REGPORT:5000 docker.io/library/registry:2 >/dev/null 2>&1
  for _ in $(seq 1 40); do curl -fsS -o /dev/null "http://127.0.0.1:$REGPORT/v2/" && break; sleep 0.5; done
  podman tag "$BUILT" "127.0.0.1:$REGPORT/helix-reader:v1"
  podman push --tls-verify=false "127.0.0.1:$REGPORT/helix-reader:v1" 2>&1 | tail -1
  echo "--- GET manifest, capture headers ---"
  HDR=$(curl -s -D - -o /tmp/helix-manifest.$$.json -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json' \
    "http://127.0.0.1:$REGPORT/v2/helix-reader/manifests/v1")
  echo "$HDR" | grep -iE '^(HTTP/|content-type|docker-content-digest|content-length)' | tr -d '\r'
  PUSH_DIGEST=$(echo "$HDR" | grep -i '^docker-content-digest' | tr -d '\r' | awk '{print $2}')
  MANIFEST_TYPE=$(echo "$HDR" | grep -i '^content-type' | tr -d '\r' | awk '{print $2}')
  echo "push_manifest_digest=$PUSH_DIGEST manifest_content_type=$MANIFEST_TYPE"
  echo "local_inspect_digest=$(podman inspect "127.0.0.1:$REGPORT/helix-reader:v1" --format '{{.Digest}}' 2>&1)"
  sha256_of_manifest=$(sha256sum /tmp/helix-manifest.$$.json 2>/dev/null | awk '{print "sha256:"$1}')
  echo "sha256_of_fetched_manifest=$sha256_of_manifest"
  echo "--- remove both local tags so the pull must traverse the network ---"
  podman rmi -f "127.0.0.1:$REGPORT/helix-reader:v1" "$BUILT" >/dev/null 2>&1
  echo "local_images_matching=$(podman images --format '{{.Repository}}:{{.Tag}}' | grep -cE 'helixqual-dist|127.0.0.1:'"$REGPORT" || true)"
  podman pull --tls-verify=false "127.0.0.1:$REGPORT/helix-reader:v1" 2>&1 | tail -1
  PULL_HDR=$(curl -s -D - -o /dev/null -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json' \
    "http://127.0.0.1:$REGPORT/v2/helix-reader/manifests/v1")
  PULL_DIGEST=$(echo "$PULL_HDR" | grep -i '^docker-content-digest' | tr -d '\r' | awk '{print $2}')
  PULLED_ID=$(podman inspect "127.0.0.1:$REGPORT/helix-reader:v1" --format '{{.Id}}' 2>/dev/null)
  echo "pull_manifest_digest=$PULL_DIGEST"
  echo "pulled_config_id=$PULLED_ID"
  echo "digest_match=$( [ -n "$PUSH_DIGEST" ] && [ "$PUSH_DIGEST" = "$PULL_DIGEST" ] && echo yes || echo no )"
  echo "config_id_matches_build=$( [ "$BUILT_ID" = "$PULLED_ID" ] && echo yes || echo no )"
  echo "manifest_digest_equals_config_id_note=digest is the manifest bytes hash; config id is the config blob hash — both are compared independently above"
  rm -f /tmp/helix-manifest.$$.json
  podman rm -f $REG >/dev/null 2>&1
  podman rmi -f "127.0.0.1:$REGPORT/helix-reader:v1" >/dev/null 2>&1
} >>"$LOG" 2>&1

{
  echo ""
  echo "## step 7: credential-gated read surface in the container (needle proofs)"
  podman build -f server/Dockerfile -t "$BUILT" . >/dev/null 2>&1
  podman run -d --name helixqual-dist-auth -p 127.0.0.1:$APPPORT:8080 \
    -e HELIX_ADDRESS=0.0.0.0:8080 -e HELIX_ADMIN_TOKEN="$ADMIN" \
    -e HELIX_OPDS_USERNAME="$OPDSU" -e HELIX_OPDS_PASSWORD="$OPDSP" \
    -e HELIX_COVER_CACHE=/tmp/covers -v "$LIB:/library:ro" "$BUILT" >/dev/null 2>&1
  READY=no
  for _ in $(seq 1 60); do
    [ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APPPORT/opds/2.0")" != "000" ] && { READY=yes; break; }
    sleep 1
  done
  echo "container_ready=$READY"
  echo "auth_enabled_log=$(podman logs helixqual-dist-auth 2>&1 | grep -c 'authentication are disabled')"
  for p in /v1/works /opds/2.0 /opds/1.2 /opds/1.2/publications; do
    echo "no_creds $p -> $(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APPPORT$p")"
    echo "wrong_password $p -> $(curl -s -o /dev/null -w '%{http_code}' -u "$OPDSU:wrong-needle-password" "http://127.0.0.1:$APPPORT$p")"
    echo "right_creds $p -> $(curl -s -o /dev/null -w '%{http_code}' -u "$OPDSU:$OPDSP" "http://127.0.0.1:$APPPORT$p")"
  done
  # The real admin surface is the /v1 group behind Bearer (router.go:108-112);
  # there is no /admin/* prefix. /users/create is KoSync registration, which the
  # compose and image defaults leave disabled (HELIX_KOSYNC_REGISTRATION=false).
  for p in /v1/scans; do
    echo "admin_no_token GET $p -> $(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APPPORT$p")"
    echo "admin_wrong_token GET $p -> $(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer wrong-needle-token" "http://127.0.0.1:$APPPORT$p")"
    echo "admin_right_token GET $p -> $(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $ADMIN" "http://127.0.0.1:$APPPORT$p")"
  done
  echo "admin_delete_no_token -> $(curl -s -o /dev/null -w '%{http_code}' -X DELETE "http://127.0.0.1:$APPPORT/v1/works/needle-does-not-exist")"
  echo "admin_delete_wrong_token -> $(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "Authorization: Bearer wrong-needle-token" "http://127.0.0.1:$APPPORT/v1/works/needle-does-not-exist")"
  echo "admin_delete_right_token -> $(curl -s -w ' http=%{http_code}' -X DELETE -H "Authorization: Bearer $ADMIN" "http://127.0.0.1:$APPPORT/v1/works/needle-does-not-exist" | head -c 120)"
  echo "kosync_register_when_disabled -> $(curl -s -w ' http=%{http_code}' -X POST -H 'Content-Type: application/json' -d '{"username":"needle","password":"needle"}' "http://127.0.0.1:$APPPORT/users/create")"
  echo "opds_feed_head=$(curl -s -u "$OPDSU:$OPDSP" "http://127.0.0.1:$APPPORT/opds/2.0" | head -c 90)"
  echo "healthz_still_open=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$APPPORT/healthz")"
  podman rm -f helixqual-dist-auth >/dev/null 2>&1
} >>"$LOG" 2>&1
echo "DONE"
