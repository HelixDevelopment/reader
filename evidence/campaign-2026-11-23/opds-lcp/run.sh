#!/usr/bin/env bash
# Live qualification of HRD-031 LCP fulfillment over OPDS against a freshly
# built helix-server process (in-memory catalog; no database needed).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../../.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
cd "$repo/server"
go build -o "$scratch/helix-server" ./cmd/helix-server
sha256sum "$scratch/helix-server" | sed "s#$scratch/##" > "$here/binary.sha256"
HELIX_LCP_LIVE_BIN="$scratch/helix-server" HELIX_LCP_LIVE_EVIDENCE="$here" \
  go test -count=1 -v -run TestLCPLiveServerQualification ./internal/httpapi/ | tee "$here/live.log"
( cd "$here" && sha256sum results.json feed-1.2.xml live.log > evidence.sha256 )
