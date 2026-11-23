#!/usr/bin/env bash
# HRD-070 mutation kill proof: apply each mutant to a scratch COPY (never the tree) and require the fuzz seed corpus to fail.
set -u
R=/home/milosvasic/Projects/helix_reader; S=${S:-/home/milosvasic/.cache/claude-tmp/gapnext_070}
rm -rf "$S/w"; mkdir -p "$S/w"; cp -r "$R/core" "$R/server" "$S/w/"; [ -f "$R/go.work" ] && cp "$R/go.work" "$S/w/"
run() { # name dir pkg file sedexpr runre
  local name=$1 dir=$2 pkg=$3 file=$4 expr=$5 re=$6
  ( cd "$S/w/$dir" && cp "$file" "$file.orig" && sed -i "$expr" "$file" && ! cmp -s "$file" "$file.orig" ) || { echo "MUTANT $name: NOT APPLIED"; return; }
  if ( cd "$S/w/$dir" && go test "$pkg" -run "$re" -count=1 >"$S/$name.out" 2>&1 ); then echo "MUTANT $name: SURVIVED (BAD)"; else echo "MUTANT $name: KILLED"; grep -m2 -E "^\s+--- FAIL|FAIL:|fuzz_test|lcpfuzz_test" "$S/$name.out" | head -2; fi
  ( cd "$S/w/$dir" && mv "$file.orig" "$file" )
  ( cd "$S/w/$dir" && go test "$pkg" -run "$re" -count=1 >/dev/null 2>&1 ) && echo "  restored: GREEN again" || echo "  restored: STILL RED (BAD)"
}
run lcp_id_check core ./pkg/lcpfuzz pkg/lcp/license.go 's/if w.ID == "" {/if false {/' FuzzParseLicense
run kosync_pct_upper server ./internal/kosync internal/kosync/service.go 's/ || progress.Percentage > 1 {/ {/' FuzzPutProgressBody
run kosync_username_len server ./internal/kosync internal/kosync/service.go 's/{3,32}/{1,64}/' FuzzRegisterInput
run opds12_truncate server ./internal/opds internal/opds/opds12.go '147s/return xml.MarshalIndent(f, "", "  ")/b, e := xml.MarshalIndent(f, "", "  "); return b[:len(b)-1], e/' FuzzOPDSGeneration
