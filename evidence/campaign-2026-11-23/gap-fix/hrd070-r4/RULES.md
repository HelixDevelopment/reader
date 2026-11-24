# HRD-070 round 4 - enumerated enforced rules (RULES.md)

Every row: production rule (file:line, found by locating the mutated text), the oracle that asserts it, the boundary seed run that exercises it (in the SEED run, `go test`, no fuzzing needed), and the mutant that must turn the seed run RED. Result column is copied from mutation.log of this round.

| mutant id | target | production location | oracle (test target) | result |
|---|---|---|---|---|
| L01_id | L | core/pkg/lcp/license.go:189 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L02_provider | L | core/pkg/lcp/license.go:192 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L04_issued_parse | L | core/pkg/lcp/license.go:201-203 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L05_updated_parse | L | core/pkg/lcp/license.go:207 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L06_updated_used | L | core/pkg/lcp/license.go:206 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L07_profile | L | core/pkg/lcp/license.go:211 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L08a_ck_value | L | core/pkg/lcp/license.go:214 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L08b_ck_algorithm | L | core/pkg/lcp/license.go:214 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L09a_uk_algorithm | L | core/pkg/lcp/license.go:222 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L09b_uk_keycheck | L | core/pkg/lcp/license.go:222 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L10_text_hint | L | core/pkg/lcp/license.go:233 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L11_links_nonempty | L | core/pkg/lcp/license.go:236 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L12_link_href | L | core/pkg/lcp/license.go:240 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L13a_sig_algorithm | L | core/pkg/lcp/license.go:244 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L13b_sig_certificate | L | core/pkg/lcp/license.go:244 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L13c_sig_value | L | core/pkg/lcp/license.go:244 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L14_rights | L | core/pkg/lcp/license.go:255 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L15_unknown_sorted | L | core/pkg/lcp/license.go:266 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L16_unknown_filter | L | core/pkg/lcp/license.go:260 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L00_syntax | L | core/pkg/lcp/license.go:182 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | KILLED |
| L03_issued_empty_EQUIV | L | core/pkg/lcp/license.go:198 | FuzzParseLicense / TestHRD070LegalDocumentsAreAccepted | SURVIVED (equivalent: no observable behaviour change) |
| S01_trailing | S | core/pkg/lcp/lsd.go:201 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S03_id | S | core/pkg/lcp/lsd.go:204 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S04_status_required | S | core/pkg/lcp/lsd.go:207 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S05_message_required | S | core/pkg/lcp/lsd.go:210 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S06_updated_required | S | core/pkg/lcp/lsd.go:213 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S07_links_required | S | core/pkg/lcp/lsd.go:216 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S08_vocab | S | core/pkg/lcp/lsd.go:220 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S08b_vocab_expired_dropped | S | core/pkg/lcp/lsd.go:74 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S09_updated_license | S | core/pkg/lcp/lsd.go:224 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S10_updated_status | S | core/pkg/lcp/lsd.go:227 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S11_href | S | core/pkg/lcp/lsd.go:232 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S12_rel_error | S | core/pkg/lcp/lsd.go:235 (parseRels) | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S12b_rel_empty | S | core/pkg/lcp/lsd.go:281 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S12c_rel_type | S | core/pkg/lcp/lsd.go:287 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S13_license_link | S | core/pkg/lcp/lsd.go:243 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S14_pr_end_parse | S | core/pkg/lcp/lsd.go:247 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S14b_pr_end_assigned | S | core/pkg/lcp/lsd.go:251 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S15_event_type | S | core/pkg/lcp/lsd.go:258 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S15b_event_type_renew | S | core/pkg/lcp/lsd.go:187 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S16_event_ts | S | core/pkg/lcp/lsd.go:264 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| S02_decode | S | core/pkg/lcp/lsd.go:198 | FuzzParseStatusDocument / TestSeedStatusIsAccepted | KILLED |
| K01_doc_empty | K | server/internal/kosync/service.go:71 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K02_progress_empty | K | server/internal/kosync/service.go:71 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K03_device_empty | K | server/internal/kosync/service.go:71 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K04_doc_len | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K04b_doc_len_off_by_one | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K05_progress_len | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K05b_progress_len_off_by_one | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K06_device_len | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K06b_device_len_off_by_one | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K07_deviceid_len | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K07b_deviceid_len_off_by_one | K | server/internal/kosync/service.go:72 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K08_nan | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K09_inf_EQUIV | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | SURVIVED (equivalent: no observable behaviour change) |
| K10_pct_negative | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K10b_pct_zero_rejected | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K11_pct_upper_dropped | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K11b_pct_upper_loose | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K11c_pct_one_rejected | K | server/internal/kosync/service.go:73 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K12_timestamp | K | server/internal/kosync/service.go:76 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K13_reg_switch | K | server/internal/kosync/service.go:39 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K14_user_min | K | server/internal/kosync/service.go:18 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K14b_user_min_up | K | server/internal/kosync/service.go:18 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K14c_user_max | K | server/internal/kosync/service.go:18 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K14d_user_max_down | K | server/internal/kosync/service.go:18 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K14e_user_charset | K | server/internal/kosync/service.go:18 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K15_pass_min | K | server/internal/kosync/service.go:42 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K15b_pass_min_up | K | server/internal/kosync/service.go:42 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K16_pass_max | K | server/internal/kosync/service.go:42 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K16b_pass_max_down | K | server/internal/kosync/service.go:42 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K17_get_doc_empty | K | server/internal/kosync/service.go:86 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K17b_get_doc_len | K | server/internal/kosync/service.go:86 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| K17c_get_doc_len_off | K | server/internal/kosync/service.go:86 | FuzzPutProgressBody / TestHRD070*/FuzzRegisterInput | KILLED |
| O01_feed_title12 | O | server/internal/opds/opds12.go:126 | FuzzOPDSGeneration | KILLED |
| O02_entry_title12 | O | server/internal/opds/detail.go:17 | FuzzOPDSGeneration | KILLED |
| O03_entry_id12 | O | server/internal/opds/detail.go:17 | FuzzOPDSGeneration | KILLED |
| O04_acq_href12 | O | server/internal/opds/opds12.go (acquisition link construction) | FuzzOPDSGeneration | KILLED |
| O05_opensearch_template | O | server/internal/opds/opds12.go:62 | FuzzOPDSGeneration | KILLED |
| O06_opensearch_xml | O | server/internal/opds/opds12.go (OpenSearch description) | FuzzOPDSGeneration | KILLED |
| O07_acq12_xml | O | server/internal/opds/opds12.go (acquisition XML) | FuzzOPDSGeneration | KILLED |
| O08_feed_title20 | O | server/internal/opds/opds20.go:15 | FuzzOPDSGeneration | KILLED |
| O09_numberOfItems | O | server/internal/opds/opds20.go:15 | FuzzOPDSGeneration | KILLED |
| O10_pub_title20 | O | server/internal/opds/detail.go:58 | FuzzOPDSGeneration | KILLED |
| O10b_pub_title_tag | O | server/internal/opds/model.go:128-130 (Metadata2.Title json tag) | FuzzOPDSGeneration | KILLED |
| O11_identifier20 | O | server/internal/opds/detail.go:58 | FuzzOPDSGeneration | KILLED |
| O12_self20 | O | server/internal/opds/detail.go:58 / opds20.go:15 | FuzzOPDSGeneration | KILLED |
| O13_acq_href20 | O | server/internal/opds/detail.go:58 | FuzzOPDSGeneration | KILLED |
| O14_edition20_context | O | server/internal/opds/model.go:136 | FuzzOPDSGeneration | KILLED |
| O15_feed_context20 | O | server/internal/opds/opds20.go:10,15 | FuzzOPDSGeneration | KILLED |

## Reviewer finding mapping (review of b8aa215)

| finding | rule | mutant row(s) that kill it | note |
|---|---|---|---|
| N1 | absent `text_hint` rejected | L10_text_hint | killed |
| N2 | malformed `updated` rejected / used | L05_updated_parse, L06_updated_used | The earlier claim that this was already killed was FALSE (no malformed-updated seed existed). Killed only from this round. |
| N4 | `rights` must decode | L14_rights | killed |
| N5 | events `type` vocabulary | S15_event_type, S15b_event_type_renew | killed |
| N6 | events `timestamp` parse | S16_event_ts | killed |
| N7 | `potential_rights.end` parse + assigned | S14_pr_end_parse, S14b_pr_end_assigned | killed |
| N8 | empty `rel` rejected | S12b_rel_empty (also S12_rel_error, S12c_rel_type) | killed |
| N9 | trailing JSON data rejected | S01_trailing | killed |
| N11 | kosync percentage upper bound `>1` | K11_pct_upper_dropped, K11b_pct_upper_loose | killed |
| N12/N13 | kosync min/max lengths | K04..K07b, K14..K16b (off-by-one both sides) | killed |
| N14 | OPDS2 `metadata.title` | O08_feed_title20, O10_pub_title20, O10b_pub_title_tag | killed |

## Equivalent mutants (survive by construction, not a gap)

- L03_issued_empty_EQUIV: an empty `issued` is also rejected by `parseTime`, so the explicit emptiness check is redundant.
- K09_inf_EQUIV: infinity is also rejected by the range check, so the explicit Inf check is redundant.

## Not enforced by production (lenient on purpose; the oracles do not demand it)

- Members differing only in letter case: encoding/json picks one; documents with such duplicates are skipped by the oracles (`hasCaseDup`).
- Unknown extra members in the license are accepted and reported via `UnknownMembers`.

