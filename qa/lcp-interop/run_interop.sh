#!/usr/bin/env bash
# Host driver for HRD-046: Readium LCP reference-server interop, only inside a rootless, network-less
# container (tools/isolation/run_container.sh). Prerequisites are built by qa/lcp-interop/README.md.
# usage: qa/lcp-interop/run_interop.sh <iso_work_dir> <evidence_dir>
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)
ISO=${1:?iso work dir}; EV=${2:?evidence dir}
export ISO_WORK=$ISO
export HELIX_LCP_INTEROP_ADMIN_USER=interop-admin
HELIX_LCP_INTEROP_ADMIN_PASS=$(head -c 24 /dev/urandom | base64 | tr -d '/+=\n'); export HELIX_LCP_INTEROP_ADMIN_PASS
HELIX_LCP_INTEROP_PASSPHRASE=$(head -c 12 /dev/urandom | base64 | tr -d '/+=\n'); export HELIX_LCP_INTEROP_PASSPHRASE
# credentials travel in a 0600 env file in the read-only bind (never on a command line; run_container.sh refuses bare -e names)
umask 077; mkdir -p "$ISO/in"; printf 'export HELIX_LCP_INTEROP_ADMIN_USER=%s HELIX_LCP_INTEROP_ADMIN_PASS=%s HELIX_LCP_INTEROP_PASSPHRASE=%s\n' \
  "$HELIX_LCP_INTEROP_ADMIN_USER" "$HELIX_LCP_INTEROP_ADMIN_PASS" "$HELIX_LCP_INTEROP_PASSPHRASE" > "$ISO/in/creds.env"
"$REPO/tools/isolation/fresh_out.sh" "$EV" >/dev/null 2>&1 || true
mkdir -p "$EV"; rm -rf "$ISO/raw" "$ISO/proof.json"; mkdir -m 0700 "$ISO/raw"
"$REPO/tools/isolation/run_container.sh" -n hrdiso-lcpinterop -m 2g -c 2 -t 600 -p "$ISO/proof.json" -o "$ISO/raw" \
  -r "$ISO/bin:/work/bin" -r "$ISO/in:/work/in" -w "$EV:/out" \
  -- sh -c '. /work/in/creds.env && sh /work/in/inside.sh' 2>&1 | tee "$EV/run.log" || true
[ -f "$ISO/raw/stdout.raw" ] && cp "$ISO/raw/stdout.raw" "$EV/container_stdout.txt"; [ -f "$ISO/raw/stderr.raw" ] && cp "$ISO/raw/stderr.raw" "$EV/container_stderr.txt" || true
rm -f "$ISO/in/creds.env"; [ -f "$ISO/proof.json" ] && cp "$ISO/proof.json" "$EV/container_proof.json"
