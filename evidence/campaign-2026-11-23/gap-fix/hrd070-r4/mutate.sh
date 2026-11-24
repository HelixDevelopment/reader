#!/usr/bin/env bash
# HRD-070 round 4 mutation kill proof (one mutant per RULES.md row): apply each mutant to a scratch COPY (never the tree) and require the fuzz seed corpus to fail.
set -u
H=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd); R=${R:-$(cd "$H/../../../.." && pwd)}; S=${S:-$(mktemp -d)}; export GOFLAGS=
rm -rf "$S/w"; mkdir -p "$S/w"; cp -r "$R/core" "$R/server" "$S/w/"; [ -f "$R/go.work" ] && cp "$R/go.work" "$S/w/"; : 
EXPECT=KILLED
run() { # name dir pkg file sedexpr runre
  local name=$1 dir=$2 pkg=$3 file=$4 expr=$5 re=$6
  ( cd "$S/w/$dir" && cp "$file" "$file.orig" && sed -i "$expr" "$file" && ! cmp -s "$file" "$file.orig" ) || { echo "MUTANT $name: NOT APPLIED"; return; }
  if ( cd "$S/w/$dir" && go test "$pkg" -run "$re" -count=1 >"$S/$name.out" 2>&1 ); then if [ "$EXPECT" = SURVIVED ]; then echo "MUTANT $name: SURVIVED (equivalent, expected)"; else echo "MUTANT $name: SURVIVED (BAD)"; fi; else if [ "$EXPECT" = SURVIVED ]; then echo "MUTANT $name: KILLED (BAD: expected equivalent)"; else echo "MUTANT $name: KILLED"; fi; grep -m2 -E "^\s+--- FAIL|FAIL:|fuzz_test|lcpfuzz_test" "$S/$name.out" | head -2; fi
  ( cd "$S/w/$dir" && mv "$file.orig" "$file" )
  ( cd "$S/w/$dir" && go test "$pkg" -run "$re" -count=1 >/dev/null 2>&1 ) && echo "  restored: GREEN again" || echo "  restored: STILL RED (BAD)"
}
L=pkg/lcp/license.go

C=core; LRE='Fuzz|TestHRD070|TestSeed'; L=pkg/lcp/license.go; SF=pkg/lcp/lsd.go
# --- License (table L) ---
run L01_id core ./pkg/lcpfuzz $L 's/if w.ID == "" {/if false {/' "$LRE"
run L02_provider core ./pkg/lcpfuzz $L 's/if w.Provider == "" {/if false {/' "$LRE"
run L04_issued_parse core ./pkg/lcpfuzz $L '/issued, err := parseTime/{n;s/if err != nil {/if false {/}' "$LRE"
run L05_updated_parse core ./pkg/lcpfuzz $L 's/"updated"); err != nil {/"updated"); false {/' "$LRE"
run L06_updated_used core ./pkg/lcpfuzz $L 's/if w.Updated != "" {/if false {/' "$LRE"
run L07_profile core ./pkg/lcpfuzz $L 's/if w.Encryption.Profile == "" {/if false {/' "$LRE"
run L08a_ck_value core ./pkg/lcpfuzz $L 's/w.Encryption.ContentKey.EncryptedValue == "" || //' "$LRE"
run L08b_ck_algorithm core ./pkg/lcpfuzz $L 's/ || w.Encryption.ContentKey.Algorithm == ""//' "$LRE"
run L09a_uk_algorithm core ./pkg/lcpfuzz $L 's/w.Encryption.UserKey.Algorithm == "" || //' "$LRE"
run L09b_uk_keycheck core ./pkg/lcpfuzz $L 's/ || w.Encryption.UserKey.KeyCheck == ""//' "$LRE"
run L10_text_hint core ./pkg/lcpfuzz $L 's/if hint.UserKey.TextHint == nil {/if false {/' "$LRE"
run L11_links_nonempty core ./pkg/lcpfuzz $L 's/if len(w.Links) == 0 {/if false {/' "$LRE"
run L12_link_href core ./pkg/lcpfuzz $L 's/if l.Href == "" {/if false {/' "$LRE"
run L13a_sig_algorithm core ./pkg/lcpfuzz $L 's/w.Signature.Algorithm == "" || //' "$LRE"
run L13b_sig_certificate core ./pkg/lcpfuzz $L 's/ || w.Signature.Certificate == ""//' "$LRE"
run L13c_sig_value core ./pkg/lcpfuzz $L 's/ || w.Signature.Value == ""//' "$LRE"
run L14_rights core ./pkg/lcpfuzz $L 's/err := json.Unmarshal(w.Rights, &lic.Rights); err != nil {/err := json.Unmarshal(w.Rights, \&lic.Rights); false {/' "$LRE"
run L15_unknown_sorted core ./pkg/lcpfuzz $L 's/sort.Strings(lic.UnknownMembers)/sort.Strings(nil)/' "$LRE"
run L16_unknown_filter core ./pkg/lcpfuzz $L 's/if !knownMembers\[k\] {/if true {/' "$LRE"
run L00_syntax core ./pkg/lcpfuzz $L 's/if err := json.Unmarshal(doc, &w); err != nil {/if false {/' "$LRE"
EXPECT=SURVIVED
run L03_issued_empty_EQUIV core ./pkg/lcpfuzz $L 's/if w.Issued == "" {/if false {/' "$LRE"
EXPECT=KILLED
# --- Status document (table S) ---
run S01_trailing core ./pkg/lcpfuzz $SF 's/!errors.Is(err, io.EOF) {/false {/' "$LRE"
run S03_id core ./pkg/lcpfuzz $SF 's/if w.ID == nil || \*w.ID == "" {/if false {/' "$LRE"
run S04_status_required core ./pkg/lcpfuzz $SF 's/if w.Status == nil {/if false {/' "$LRE"
run S05_message_required core ./pkg/lcpfuzz $SF 's/if w.Message == nil {/if false {/' "$LRE"
run S06_updated_required core ./pkg/lcpfuzz $SF 's/if w.Updated == nil || w.Updated.License == nil || w.Updated.Status == nil {/if false {/' "$LRE"
run S07_links_required core ./pkg/lcpfuzz $SF 's/if w.Links == nil {/if false {/' "$LRE"
run S08_vocab core ./pkg/lcpfuzz $SF 's/if !d.Status.valid() {/if false {/' "$LRE"
run S08b_vocab_expired_dropped core ./pkg/lcpfuzz $SF 's/, StatusCancelled, StatusExpired:/, StatusCancelled:/' "$LRE"
run S09_updated_license core ./pkg/lcpfuzz $SF 's/"updated.license"); err != nil {/"updated.license"); false {/' "$LRE"
run S10_updated_status core ./pkg/lcpfuzz $SF 's/"updated.status"); err != nil {/"updated.status"); false {/' "$LRE"
run S11_href core ./pkg/lcpfuzz $SF 's/wl.Href == nil || \*wl.Href == ""/wl.Href == nil/' "$LRE"
run S12_rel_error core ./pkg/lcpfuzz $SF '/rels, err := parseRels/{n;s/if err != nil {/if false {/}' "$LRE"
run S12b_rel_empty core ./pkg/lcpfuzz $SF 's/if one == "" {/if false {/' "$LRE"
run S12c_rel_type core ./pkg/lcpfuzz $SF 's/if json.Unmarshal(raw, &many) != nil {/if false {/' "$LRE"
run S13_license_link core ./pkg/lcpfuzz $SF 's/if !hasLicense {/if false {/' "$LRE"
run S14_pr_end_parse core ./pkg/lcpfuzz $SF '/"potential_rights.end")/{n;s/if err != nil {/if false {/}' "$LRE"
run S14b_pr_end_assigned core ./pkg/lcpfuzz $SF 's/d.PotentialRightsEnd = &t/_ = t/' "$LRE"
run S15_event_type core ./pkg/lcpfuzz $SF 's/if !statusEventTypes\[\*we.Type\] {/if false {/' "$LRE"
run S15b_event_type_renew core ./pkg/lcpfuzz $SF 's/"renew": true,/"renew": false,/' "$LRE"
run S16_event_ts core ./pkg/lcpfuzz $SF '/fmt.Sprintf("events\[%d\].timestamp", i))/{n;s/if err != nil {/if false {/}' "$LRE"
run S02_decode core ./pkg/lcpfuzz $SF 's/if err := dec.Decode(&w); err != nil {/if false {/' "$LRE"
# --- KOSync (table K) ---
K=internal/kosync/service.go; KRE='FuzzPutProgressBody|FuzzRegisterInput|TestHRD070'
run K01_doc_empty server ./internal/kosync $K 's/if progress.Document == "" || /if /' "$KRE"
run K02_progress_empty server ./internal/kosync $K 's/ || progress.Progress == ""//' "$KRE"
run K03_device_empty server ./internal/kosync $K 's/ || progress.Device == "" ||/ ||/' "$KRE"
run K04_doc_len server ./internal/kosync $K 's/len(progress.Document) > 255 || //' "$KRE"
run K04b_doc_len_off_by_one server ./internal/kosync $K 's/len(progress.Document) > 255/len(progress.Document) >= 255/' "$KRE"
run K05_progress_len server ./internal/kosync $K 's/ || len(progress.Progress) > 255//' "$KRE"
run K05b_progress_len_off_by_one server ./internal/kosync $K 's/len(progress.Progress) > 255/len(progress.Progress) >= 255/' "$KRE"
run K06_device_len server ./internal/kosync $K 's/ || len(progress.Device) > 255//' "$KRE"
run K06b_device_len_off_by_one server ./internal/kosync $K 's/len(progress.Device) > 255/len(progress.Device) >= 255/' "$KRE"
run K07_deviceid_len server ./internal/kosync $K 's/ || len(progress.DeviceID) > 255//' "$KRE"
run K07b_deviceid_len_off_by_one server ./internal/kosync $K 's/len(progress.DeviceID) > 255/len(progress.DeviceID) >= 255/' "$KRE"
run K08_nan server ./internal/kosync $K 's/math.IsNaN(progress.Percentage) || //' "$KRE"
EXPECT=SURVIVED
run K09_inf_EQUIV server ./internal/kosync $K 's/math.IsInf(progress.Percentage, 0) || //' "$KRE"
EXPECT=KILLED
run K10_pct_negative server ./internal/kosync $K 's/ || progress.Percentage < 0//' "$KRE"
run K10b_pct_zero_rejected server ./internal/kosync $K 's/progress.Percentage < 0/progress.Percentage <= 0/' "$KRE"
run K11_pct_upper_dropped server ./internal/kosync $K 's/ || progress.Percentage > 1 {/ {/' "$KRE"
run K11b_pct_upper_loose server ./internal/kosync $K 's/progress.Percentage > 1 {/progress.Percentage > 1.0000001 {/' "$KRE"
run K11c_pct_one_rejected server ./internal/kosync $K 's/progress.Percentage > 1 {/progress.Percentage >= 1 {/' "$KRE"
run K12_timestamp server ./internal/kosync $K 's/progress.Timestamp = s.now().Unix()/_ = s.now/' "$KRE"
run K13_reg_switch server ./internal/kosync $K 's/if !s.registration {/if false {/' "$KRE"
run K14_user_min server ./internal/kosync $K 's/{3,32}/{2,32}/' "$KRE"
run K14b_user_min_up server ./internal/kosync $K 's/{3,32}/{4,32}/' "$KRE"
run K14c_user_max server ./internal/kosync $K 's/{3,32}/{3,33}/' "$KRE"
run K14d_user_max_down server ./internal/kosync $K 's/{3,32}/{3,31}/' "$KRE"
run K14e_user_charset server ./internal/kosync $K 's/\[A-Za-z0-9_-\]/[A-Za-z0-9-]/' "$KRE"
run K15_pass_min server ./internal/kosync $K '/validUsername.MatchString/s/len(password) < 8/len(password) < 7/' "$KRE"
run K15b_pass_min_up server ./internal/kosync $K '/validUsername.MatchString/s/len(password) < 8/len(password) < 9/' "$KRE"
run K16_pass_max server ./internal/kosync $K '/validUsername.MatchString/s/len(password) > 72/len(password) > 73/' "$KRE"
run K16b_pass_max_down server ./internal/kosync $K '/validUsername.MatchString/s/len(password) > 72/len(password) > 71/' "$KRE"
run K17_get_doc_empty server ./internal/kosync $K 's/if document == "" || len(document) > 255 {/if len(document) > 255 {/' "$KRE"
run K17b_get_doc_len server ./internal/kosync $K 's/if document == "" || len(document) > 255 {/if document == "" {/' "$KRE"
run K17c_get_doc_len_off server ./internal/kosync $K 's/len(document) > 255 {/len(document) >= 255 {/' "$KRE"
# --- OPDS (table O) ---
OR='FuzzOPDSGeneration'; M=internal/opds
run O01_feed_title12 server ./internal/opds $M/opds12.go 's/"Helix Publications", updated, MIME12Acquisition/"Helix Pubs", updated, MIME12Acquisition/' "$OR"
run O02_entry_title12 server ./internal/opds $M/detail.go 's/Entry12{Title: e.Title, /Entry12{Title: "", /' "$OR"
run O03_entry_id12 server ./internal/opds $M/detail.go 's/ID: "urn:helix:edition:" + e.ID, Updated: stamp(e.UpdatedAt), Language/ID: e.ID, Updated: stamp(e.UpdatedAt), Language/' "$OR"
run O04_acq_href12 server ./internal/opds $M/detail.go 's/Rel: RelAcquisition, HREF: "\/v1\/files\/" + url.PathEscape(file.ID)/Rel: RelAcquisition, HREF: "\/v1\/files\/" + file.ID/' "$OR"
run O05_opensearch_template server ./internal/opds $M/opds12.go 's/Template: base + /Template: /' "$OR"
run O06_opensearch_xml server ./internal/opds $M/opds12.go '63s/return xml.MarshalIndent(d, "", "  ")/b, e := xml.MarshalIndent(d, "", "  "); return b[:len(b)-1], e/' "$OR"
run O07_acq12_xml server ./internal/opds $M/opds12.go '147s/return xml.MarshalIndent(f, "", "  ")/b, e := xml.MarshalIndent(f, "", "  "); return b[:len(b)-1], e/' "$OR"
run O08_feed_title20 server ./internal/opds $M/opds20.go 's/Title: "Helix Publications", NumberOfItems/Title: "Helix Pubs", NumberOfItems/' "$OR"
run O09_numberOfItems server ./internal/opds $M/opds20.go 's/NumberOfItems: page.Total/NumberOfItems: 1/' "$OR"
run O10_pub_title20 server ./internal/opds $M/detail.go 's/Metadata2{Title: e.Title, /Metadata2{Title: "", /' "$OR"
run O10b_pub_title_tag server ./internal/opds $M/model.go '/^type Metadata2/,/^}/s/json:"title"/json:"-"/' "$OR"
run O11_identifier20 server ./internal/opds $M/detail.go 's/Identifier: "urn:helix:edition:" + e.ID, Modified/Identifier: e.ID, Modified/' "$OR"
run O12_self20 server ./internal/opds $M/detail.go 's/"\/opds\/2.0\/editions\/" + url.PathEscape(e.ID), Rel: "self"/"\/opds\/2.0\/editions\/" + e.ID, Rel: "self"/' "$OR"
run O13_acq_href20 server ./internal/opds $M/detail.go 's/Link2{HREF: "\/v1\/files\/" + url.PathEscape(file.ID)/Link2{HREF: "\/v1\/files\/" + file.ID/' "$OR"
run O14_edition20_context server ./internal/opds $M/detail.go 's/p.Context = "https:\/\/readium.org\/webpub-manifest\/context.jsonld"/p.Context = ""/' "$OR"
run O15_feed_context20 server ./internal/opds $M/opds20.go '/^func Acquisition20/,/^}/s/Context: "https:\/\/readium.org\/webpub-manifest\/context.jsonld"/Context: ""/' "$OR"
