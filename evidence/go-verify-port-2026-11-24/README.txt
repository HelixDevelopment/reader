# Go verification port evidence (2026-11-24, re-recorded after the independent pre-review NO-GO)

Layout (sealed with sha256 in SHA256SUMS; verify with: cd <copy of this dir> && sha256sum -c SHA256SUMS):
- mutation-python/results.json: scripts/verify_go_mutation.py, full 104-mutant campaign (76 controls), PASS (unchanged from the first recording).
- mutation-go/results.json: tools/verify/cmd/gomutate, same catalogue, same tree, RE-RUN after the scratch-removal and copyTree changes: PASS, 104/104, controls 76.
- mutation-parity.json: per-id comparison of status, reason, expected, test, exit (recomputed): 104 ids, equal id sets, 0 differences.
- mutation-selftest/results.json: gomutate --selftest, 25 runner and UI mutants, all killed with the predicted assertion (includes the four scratch-removal
  mutants, three isolation-gate mutants, the action-budget and fixture-verdict mutants).
- ui-live/<run>/: uiverify runs in the rootless container (qa/isolation/run_uiverify.sh; container_proof.json: network none, cap-drop, no-new-privileges,
  /out bound to this directory's own subdirectory, so the bind paths in the proofs exist). Every run also holds fixture/fixture-results.json
  (the locator/visibility engine against known content).

Live runs (same base bundle, same good binary unless stated):
| run | what differs | ui-results | fixture |
| good | none | 64 pass / 0 fail | 30 pass / 0 fail |
| mutant-next-noop | Next handler is a no-op | 60/4: only reader-next-delta | 30/0 |
| mutant-font-removed | fonts/NotoSans-SemiBold.woff deleted | 56/8: only console-errors, missing-assets | 30/0 |
| mutant-disclosure-text-changed | disclosure strings changed | 60/4: only sample-data-disclosure | 30/0 |
| mutant-source-validation-removed | validation message text changed | 60/4: only source-validation | 30/0 |
| mutant-scan-state-text-changed | scan state text changed | 60/4: only source-scan-state | 30/0 |
| mutant-conflict-text-changed | conflict text changed | 60/4: only sync-conflict-delta | 30/0 |
| hv-mutant-always-visible | hv.js isVisible returns true (reviewer mutation) | 64/0 (product checks alone would NOT catch it) | 25/5: FAILS (bad-hidden-display-none, bad-hidden-visibility, bad-hidden-zero-size, bad-visible-filter-drops-hidden, good-role-button-visible-only); run exit 1 |
| hv-mutant-never-visible | hv.js isVisible returns false | 0/4 viewport-run | 22/8: FAILS; run exit 1 |

Provenance (replayable, M6): the base bundle is demos/interactive_static_demo built with 'npm ci && npm run build' (tree sha256 abf6e5d45927fb8747d300774eff7a0b0908e5b0118ef5d7b24f7e3ecc287110);
each mutant dist is that bundle with exact-once replacements recorded in ui-live/mutant-*/mutation.json (anchor, replacement, sha256 of every changed file,
tree sha256), produced by qa/isolation/make_ui_mutants.py; the engine mutant binaries are produced by qa/isolation/make_hv_mutant_bins.sh (reproduced twice: identical sha256).
Binary sha256: bf8fad6eca7e2c5ab5d3d916ef1e95e8e6e9536d1c7cff17311e6be10965651c (good), 308a8a087c9eb5a0c3a1f4a7a93236fccb40f411d3f1c644aea2727129139f72 (hv always-visible), 4de7fe33fb063a66e03c770d3c381171cca1d6a1ee8628ac9762d86b3c2457b8 (hv never-visible).
The first recording (3 live runs from product-source mutants whose exact diffs were not kept) was replaced by this one; its numbers (64/0, 60/4, 56/8) are reproduced above.
Per-mutant logs of the Go campaigns are omitted (bulk); the results.json files carry id, status, reason and test.
