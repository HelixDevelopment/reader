#!/usr/bin/env bash
# Host driver for HRD-046: Readium LCP reference-server interop, only inside a rootless, network-less
# container (tools/isolation/run_container.sh). Prerequisites are built as described in qa/lcp-interop/README.md.
# usage: qa/lcp-interop/run_interop.sh <iso_work_dir> <evidence_dir> [oracle_epub_name]
#   <iso_work_dir>/bin must hold lcpserver lsdserver lcpencrypt lcp.test; <iso_work_dir>/in must hold sample.epub,
#   cert-edrlab-test.pem (and lorem.epub for the wrong-oracle mutant). <evidence_dir> must lie below <repo>/evidence/.
#   oracle_epub_name defaults to sample.epub; any other name is the wrong-oracle MUTANT (the run is then expected to FAIL).
# Exit status carries the verdict: 0 = the interop test passed AND the secret scan passed AND the evidence was sealed;
#   non-zero = interop failure (its exit code), secret-scan failure (3), or a driver error. Nothing is sealed on a scan failure.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); REPO=$(cd "$HERE/../.." && pwd)
ISO=${1:?iso work dir}; EV=${2:?evidence dir}; ORACLE=${3:-sample.epub}
export ISO_WORK=$ISO
trap 'rm -f "$ISO/in/creds.env"' EXIT
export HELIX_LCP_INTEROP_ADMIN_USER=interop-admin
HELIX_LCP_INTEROP_ADMIN_PASS=$(head -c 24 /dev/urandom | base64 | tr -d '/+=\n'); export HELIX_LCP_INTEROP_ADMIN_PASS
HELIX_LCP_INTEROP_PASSPHRASE=$(head -c 12 /dev/urandom | base64 | tr -d '/+=\n'); export HELIX_LCP_INTEROP_PASSPHRASE
# credentials travel in a 0600 env file in the read-only bind (never on a command line; run_container.sh refuses bare -e names)
umask 077; mkdir -p "$ISO/in"; install -m 0644 "$HERE/inside.sh" "$ISO/in/inside.sh"
printf 'export HELIX_LCP_INTEROP_ADMIN_USER=%s HELIX_LCP_INTEROP_ADMIN_PASS=%s HELIX_LCP_INTEROP_PASSPHRASE=%s HELIX_LCP_ORACLE_EPUB=%s\n' \
  "$HELIX_LCP_INTEROP_ADMIN_USER" "$HELIX_LCP_INTEROP_ADMIN_PASS" "$HELIX_LCP_INTEROP_PASSPHRASE" "$ORACLE" > "$ISO/in/creds.env"
"$REPO/tools/isolation/fresh_out.sh" "$EV" >/dev/null
mkdir -p "$EV"; rm -rf "$ISO/raw" "$ISO/proof.json"; mkdir -m 0700 "$ISO/raw"
set +e
"$REPO/tools/isolation/run_container.sh" -n hrdiso-lcpinterop -m 2g -c 2 -t 600 -p "$ISO/proof.json" -o "$ISO/raw" \
  -r "$ISO/bin:/work/bin" -r "$ISO/in:/work/in" -w "$EV:/out" \
  -- sh -c '. /work/in/creds.env && sh /work/in/inside.sh' 2>&1 | tee "$EV/run.log"
crc=${PIPESTATUS[0]}
set -e
[ -f "$ISO/raw/stdout.raw" ] && cp "$ISO/raw/stdout.raw" "$EV/container_stdout.txt"
[ -f "$ISO/raw/stderr.raw" ] && cp "$ISO/raw/stderr.raw" "$EV/container_stderr.txt"
rm -f "$ISO/in/creds.env"; [ -f "$ISO/proof.json" ] && cp "$ISO/proof.json" "$EV/container_proof.json"
# host-side secret scan of the final evidence directory (the in-container scan cannot see the files copied here);
# the values are passed through a 0600 pattern file and never printed
SCAN=$(mktemp); printf '%s\n' "$HELIX_LCP_INTEROP_ADMIN_PASS" "$HELIX_LCP_INTEROP_PASSPHRASE" > "$SCAN"
bad=0
grep -rIil 'content-encryption-key' "$EV" >/dev/null 2>&1 && { echo "host secret scan: content-encryption-key marker found in $EV" >&2; bad=1; }
grep -rIl -F -f "$SCAN" "$EV" >/dev/null 2>&1 && { echo "host secret scan: a credential value found in $EV" >&2; bad=1; }
rm -f "$SCAN"
if [ "$bad" = 1 ]; then echo "NOT sealing $EV; do not commit it" >&2; exit 3; fi
"$REPO/tools/isolation/seal_evidence.sh" "$EV"
rc=$(sed -n 's/^helix_interop_exit=//p' "$EV/helix_interop_exit.txt" 2>/dev/null | head -1)
[ -n "$rc" ] || { echo "no helix_interop_exit.txt: the run did not reach the interop test (container exit $crc)" >&2; exit 2; }
if [ "$rc" != 0 ] || [ "$crc" != 0 ]; then echo "interop exit=$rc container exit=$crc" >&2; exit 1; fi
