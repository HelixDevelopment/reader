# HRD-050 evidence provenance (sealed dir `hrd050/` is not altered)
- run_A/run_B headers say head=e9c89d3 (the base) and the runs were made on a DIRTY tree that included
  server/internal/httpapi/load_concurrent_test.go, which is NOT part of this PR. The sealed live logs are therefore
  NOT reproducible from the PR head; headers of future runs record dirty_files and dirty_diff_sha.
- gate_negative/*, skips_live.txt, netskips_live.txt and skips_anywhere_live.txt were produced by the OLD bare-name gate (79eb80f, before package keying); they are not outputs of the current gate.
- Not produced by the current script (earlier iterations / manual runs, kept because sealed): skips_live.txt,
  race_load_output.json, first_attempt_live_log_RED_relpath_and_race.log, first_attempt_noenv.log, race_reverify_opds_fix.log,
  baseline_core_noenv.log, baseline_server_noenv.log (RED baseline, `--no-env`), gate_negative/* (recorded gate outputs, superseded
  by tests/live/test_run_gated_tests.sh), mutation/* (output of tests/live/mutation_check.sh, live only).
- Evidence from other code: everything involving TestLiveLoadIngestConcurrentWithReads.
- Disclosure: PR heads 79eb80f, 467372f and 10cc4c9 ended the live run with `tools/isolation/seal_evidence.sh`, which is NOT tracked in the repository (it existed only as an untracked directory in the author's main checkout). The sealed evidence was produced where that tool existed; from a clean clone those heads could not complete a live run. Fixed after 10cc4c9 by the in-script `seal_dir`.
