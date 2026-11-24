# evidence/h3-cronet - reading guide (outside the seals)

This file is deliberately NOT inside `android/` or `matrix/`: those directories are sealed (`SHA256SUMS`) and frozen.

## android/
- The sealed run was made with `LAB_SYSTEM_CA=1` (per-run CA bound into the emulator system store; see `adb_root.txt`, `system_ca.txt`),
  `EXPECT_PROTO` unset (default `HTTP/2.0`), `EXPECT_H3` unset. The verdict JSON does not record this; this README does. Newer runs of
  `qa/h3/run_android_h3.sh` write the same facts to `run_env.txt`.
- `apk_sha256.txt` is the file the script now writes under that same name (older script revisions wrote `apk.sha256`; the sealed
  file name is `apk_sha256.txt`). It lists host-absolute paths of the two lab APKs (`app-lab.apk`, `app-lab-androidTest.apk`) with sha256.
- Android h3 is UNCONFIRMED: server lines show HTTP/2.0 + br; the netlog shows QUIC sessions failing with net_error -356.

## matrix/
- Verdict files (`*.verdict.json`) are produced by `qa/h3/check_h3.py` from the SERVER access log slices (`*.server.log`).
- `p1.json`..`p4.json` are exploratory Chromium probes made while finding the private-CA flags (p1: SPKI-pin flag only, h2 on every round;
  p2: SPKI-pin + `--enable-quic` + `--origin-to-force-quic-on`; p3/p4: `--ignore-certificate-errors` + `--enable-quic` [+ `--quic-version=h3`]). They are not produced by `run_h3_live.sh` and no verdict cites them.
- `desk_new.json` is a byte-identical earlier copy of `desktop-new.json` (same sha256); `desktop-noq.json` / `chromium-noforce.json` are the
  client-side JSON of the no-quic-origin / no-force controls. Verdicts cite the `*.verdict.json` files only.
- Server identity: the server in all runs is the `helix-server` binary built from this PR head inside the lab container; the verdict files do not
  carry its sha256 (known limit, follow-up; §11.4.289(D)).
