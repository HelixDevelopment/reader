#!/usr/bin/env bash
# HRD-070 mutation kill proof: apply each mutant to a scratch COPY (never the tree) and require the fuzz seed corpus to fail.
set -u
H=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd); R=${R:-$(cd "$H/../../../.." && pwd)}; S=${S:-$(mktemp -d)}; export GOFLAGS=
rm -rf "$S/w"; mkdir -p "$S/w"; cp -r "$R/core" "$R/server" "$S/w/"; [ -f "$R/go.work" ] && cp "$R/go.work" "$S/w/"; : 
run() { # name dir pkg file sedexpr runre
  local name=$1 dir=$2 pkg=$3 file=$4 expr=$5 re=$6
  ( cd "$S/w/$dir" && cp "$file" "$file.orig" && sed -i "$expr" "$file" && ! cmp -s "$file" "$file.orig" ) || { echo "MUTANT $name: NOT APPLIED"; return; }
  if ( cd "$S/w/$dir" && go test "$pkg" -run "$re" -count=1 >"$S/$name.out" 2>&1 ); then echo "MUTANT $name: SURVIVED (BAD)"; else echo "MUTANT $name: KILLED"; grep -m2 -E "^\s+--- FAIL|FAIL:|fuzz_test|lcpfuzz_test" "$S/$name.out" | head -2; fi
  ( cd "$S/w/$dir" && mv "$file.orig" "$file" )
  ( cd "$S/w/$dir" && go test "$pkg" -run "$re" -count=1 >/dev/null 2>&1 ) && echo "  restored: GREEN again" || echo "  restored: STILL RED (BAD)"
}
L=pkg/lcp/license.go
run lcp_id_check core ./pkg/lcpfuzz $L 's/if w.ID == "" {/if false {/' FuzzParseLicense
run lcp_provider_check core ./pkg/lcpfuzz $L 's/if w.Provider == "" {/if false {/' FuzzParseLicense
run lcp_encryption_profile core ./pkg/lcpfuzz $L 's/if w.Encryption.Profile == "" {/if false {/' FuzzParseLicense
run lcp_content_key core ./pkg/lcpfuzz $L 's/if w.Encryption.ContentKey.EncryptedValue == "" || w.Encryption.ContentKey.Algorithm == "" {/if false {/' FuzzParseLicense
run lcp_user_key core ./pkg/lcpfuzz $L 's/if w.Encryption.UserKey.Algorithm == "" || w.Encryption.UserKey.KeyCheck == "" {/if false {/' FuzzParseLicense
run lcp_links_nonempty core ./pkg/lcpfuzz $L 's/if len(w.Links) == 0 {/if false {/' FuzzParseLicense
run lcp_link_href core ./pkg/lcpfuzz $L 's/if l.Href == "" {/if false {/' FuzzParseLicense
run lcp_signature_M4 core ./pkg/lcpfuzz $L 's/if w.Signature.Algorithm == "" || w.Signature.Certificate == "" || w.Signature.Value == "" {/if false {/' FuzzParseLicense
run lsd_id_check core ./pkg/lcpfuzz pkg/lcp/lsd.go 's/if w.ID == nil || \*w.ID == "" {/if false {/' 'FuzzParseStatusDocument'
run lsd_status_vocab core ./pkg/lcpfuzz pkg/lcp/lsd.go 's/if !d.Status.valid() {/if false {/' 'FuzzParseStatusDocument'
run lsd_license_link core ./pkg/lcpfuzz pkg/lcp/lsd.go 's/if !hasLicense {/if false {/' 'FuzzParseStatusDocument'
run lsd_links_required core ./pkg/lcpfuzz pkg/lcp/lsd.go 's/if w.Links == nil {/if false {/' 'FuzzParseStatusDocument'
run kosync_pct_upper server ./internal/kosync internal/kosync/service.go 's/ || progress.Percentage > 1 {/ {/' FuzzPutProgressBody
run kosync_username_len server ./internal/kosync internal/kosync/service.go 's/{3,32}/{1,64}/' FuzzRegisterInput
run opds12_truncate server ./internal/opds internal/opds/opds12.go '147s/return xml.MarshalIndent(f, "", "  ")/b, e := xml.MarshalIndent(f, "", "  "); return b[:len(b)-1], e/' FuzzOPDSGeneration
