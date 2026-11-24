# Gap Register to Workable-Item Map 2026-11-23

| Field | Value |
|---|---|
| Document | docs/gap-register-item-map-2026-11-23.md |
| Revision | 1.0 (2026-11-23) |
| Source | docs/gap-register-2026-11-23.md (40 gaps) |
| Method | Items filed through scripts/workable_items.py add/update only; DB backup at ~/.cache/claude-tmp/workable_items.db.pre-gapfile-20261123 |
| Result | 39 new items HRD-037..HRD-075 (all Queued, none started), 1 existing (HRD-036 covers GAP-009); back-link updates on HRD-012/013/014/015/021/022/023/025/026/027/028/029/031/032/036 |
| Tool gap | The tool has no link/composes command; composes_with stays [] and links are recorded in descriptions (cross-link text, both directions) |

Numbering: GAP-001..008 -> HRD-037..044; GAP-010..040 -> HRD-045..075 (GAP-009 is HRD-036).
GAP-016 is HRD-051 and GAP-017 is HRD-052 (the register lists GAP-017 before GAP-016 in its table; IDs follow GAP numbers).

| GAP-id | HRD-id | New/existing | Type | Severity | Related existing items | Rationale |
|---|---|---|---|---|---|---|
| GAP-001 | HRD-037 | new | Task | Critical | HRD-022 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-002 | HRD-038 | new | Feature | Critical | HRD-021 | new: HRD-021 lists web bank execution as NOT done but its acceptance is generic; GAP-002 has a specific measurable acceptance (0 SKIP, RED mutant). Linked both ways. |
| GAP-003 | HRD-039 | new | Task | Critical | HRD-021 | new: HRD-021 lists Vision/UI QA as unverified only; no item carries the 11.4.277 llmctl validation + per-screenshot verdict acceptance. Linked both ways. |
| GAP-004 | HRD-040 | new | Feature | Critical | HRD-022 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-005 | HRD-041 | new | Feature | High | HRD-022, HRD-040 (GAP-004) | new: HRD-022 was re-scoped by the operator to web+Linux+Android, so Android TV is outside its acceptance; linked. |
| GAP-006 | HRD-042 | new | Feature | High | HRD-022, HRD-040 (GAP-004) | new: Windows/macOS/iOS outside HRD-022 re-scope (BLOCKED); linked. |
| GAP-007 | HRD-043 | new | Feature | High | HRD-022, HRD-040 (GAP-004) | new: Linux desktop is in HRD-022 scope but acceptance there is generic; specific GUI-in-container + manifest acceptance; linked. |
| GAP-008 | HRD-044 | new | Feature | High | HRD-022, HRD-032 | new: HRD-022 notes plain-text only; HRD-032 (closed Implemented, rendering for every format) is the claimed origin - linked per 11.4.214, UNCONFIRMED that it is the same defect recurring. |
| GAP-009 | HRD-036 | existing | Bug | High | HRD-026, HRD-031 | existing: HRD-036 is the same Bug (same evidence, acceptance covers Range 206 + PSE link + RED baseline); not re-minted. HRD-036 description gains a 'covered by GAP-009' link. |
| GAP-010 | HRD-045 | new | Bug | High | HRD-027 | new: HRD-027 (In progress, Task) does not record the 152-176 vs 271.94 ops/s floor miss nor an A/B acceptance; HRD-027 is the origin of the floor (lock-scope work), linked both ways. |
| GAP-011 | HRD-046 | new | Task | High | HRD-031, HRD-026 | new: HRD-031 lists real LCP provider/reader interop as NOT done, generic acceptance; GAP-011 adds independent-oracle acceptance. Linked both ways (also HRD-026). |
| GAP-012 | HRD-047 | new | Task | High | HRD-021 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-013 | HRD-048 | new | Task | High | HRD-027 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-014 | HRD-049 | new | Task | High | HRD-012 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-015 | HRD-050 | new | Task | High | HRD-015 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-016 | HRD-051 | new | Task | Medium | HRD-059 (GAP-024) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-017 | HRD-052 | new | Task | High | HRD-021 | new: HRD-021 reports its own verifier PASS pass=21 while the constitution gate reports 4 DRIFT; no item tracks the drift. |
| GAP-018 | HRD-053 | new | Task | Medium | HRD-021 | new: pin lag not tracked by any item. |
| GAP-019 | HRD-054 | new | Task | Medium | HRD-021 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-020 | HRD-055 | new | Task | Medium | - | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-021 | HRD-056 | new | Task | Medium | HRD-021 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-022 | HRD-057 | new | Task | Medium | HRD-031, HRD-015, HRD-050 (GAP-015) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-023 | HRD-058 | new | Task | Medium | HRD-028 | new: ID-KEY-RENAME.md documents the decision but no item tracks the schema-rename-vs-mapping operator decision; HRD-028 (registry) linked. |
| GAP-024 | HRD-059 | new | Task | Medium | HRD-031, HRD-021, HRD-051 (GAP-016) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-025 | HRD-060 | new | Task | Medium | HRD-014 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-026 | HRD-061 | new | Feature | Medium | HRD-022, HRD-038 (GAP-002) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-027 | HRD-062 | new | Feature | Medium | HRD-022, HRD-040 (GAP-004) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-028 | HRD-063 | new | Feature | Medium | HRD-022 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-029 | HRD-064 | new | Task | Medium | HRD-022 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-030 | HRD-065 | new | Task | Medium | HRD-015, HRD-012 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-031 | HRD-066 | new | Task | Medium | HRD-026 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-032 | HRD-067 | new | Task | Medium | HRD-027, HRD-050 (GAP-015) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-033 | HRD-068 | new | Task | Medium | HRD-029 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-034 | HRD-069 | new | Feature | Medium | HRD-023, HRD-038 (GAP-002) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-035 | HRD-070 | new | Task | Medium | HRD-013 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-036 | HRD-071 | new | Task | Medium | HRD-012, HRD-051 (GAP-016) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-037 | HRD-072 | new | Task | Low | HRD-022, HRD-025 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-038 | HRD-073 | new | Task | Low | HRD-022, HRD-040 (GAP-004) | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-039 | HRD-074 | new | Task | Low | HRD-027, HRD-014 | new: no existing HRD-001..HRD-036 item carries this acceptance; linked to related items both ways in descriptions. |
| GAP-040 | HRD-075 | new | Task | Low | HRD-031, HRD-025, HRD-026, HRD-029, HRD-028 | new: no item tracks status-vs-evidence lag; touches HRD-025/026/029/031 (all linked). |

Notes: a returning defect links to its origin rather than minting a duplicate (11.4.214); where an existing item only partially covers a gap, a new item was minted and linked both ways. Root causes stay UNCONFIRMED as labelled in the register.
