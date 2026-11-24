# tools/verify: Go product-verification tools

Go replacements for the Python product-verification scripts (operator mandate: Go base, Python does not carry product verification).
Separate module (`go.mod` here) so `core/` and `server/` do not gain the chromedp dependency.

| Tool | Replaces | Runs where |
|---|---|---|
| `cmd/gomutate` | `scripts/verify_go_mutation.py` | host (no GUI; runs `go test` in scratch copies of `core/` and `server/`) |
| `cmd/uiverify` | `scripts/verify_ui.py` | ONLY inside the container that `tools/isolation/run_container.sh` starts (`qa/isolation/run_uiverify.sh`): it refuses to start a browser unless `/run/.containerenv`, the read-only `/opt/isoinside/launch.sh` marker, `HRDISO_ISOLATED=1` (exported by launch.sh after its host-display/socket assertions), a loopback-only network and no host display variable are all present. This is a second line of defence, not a proof of rootlessness (that is `container_proof.json`). |

The Python scripts are kept until parity is proven and the operator accepts their removal (constitution 11.4.122). `scripts/verify_ui_mutation.py` (named in the main README before this change) is not in the tree and is not ported. `scripts/verify_ui.py` drives a host browser and must not be run on the host.

## gomutate

```bash
cd tools/verify && go build -o /path/gomutate ./cmd/gomutate
gomutate --root <repo>                 # all 104 product mutants (catalogue: internal/mutation/catalogue.json, ported 1:1)
gomutate --root <repo> --only id,id    # subset; an unknown id is an error, never a vacuous PASS
gomutate --root <repo> --list
gomutate --root <repo> --selftest      # the runner's own paired mutants (mutation/selftest_catalogue.json)
```

Semantics are those of the Python script: each mutant runs in a copy of every top-level Go module; the anchor (`old`) must occur exactly
once; a positive control runs every distinct targeted test on the pristine copy first; a mutant counts as rejected only when the test fails
AND its output contains the predicted assertion (`expect`); build failures, timeouts and wrong-assertion failures are FAILs; the original
files are hashed before and after. Output: `evidence/go-mutation/results.json` (same schema) plus one log per mutant and per control.
Additions over Python: `--only`, `--selftest`, the per-test wall budget kills the whole process group (never pgid <= 1), and the scratch
directory removal (`removeScratch`) deletes ONLY a strict descendant of a `.helix-verify` directory that this tool created (marker file with fixed content, or empty), never `$HOME`, an ancestor of it, a protected child, a system directory or a symlink; `--work` is validated before anything is deleted (constitution 11.4.287).

## uiverify

```bash
(cd tools/verify && CGO_ENABLED=0 go build -trimpath -o "$HELIX_UIVERIFY_BIN" ./cmd/uiverify)
(cd demos/interactive_static_demo && npm ci && npm run build)          # produces dist/
ISO_WORK=$HOME/.cache/claude-tmp/<dir> HELIX_UIVERIFY_BIN=... qa/isolation/run_uiverify.sh demos/interactive_static_demo/dist evidence/<fresh-dir>
```

Same sixteen checks per viewport (phone 375, tablet 768, desktop 1024, wide 1440), same `ui-results.json` schema (`schema_version` 1). The
binary serves the built directory itself on the container loopback (no node, no network) and drives the Playwright-bundled Chromium through
chromedp. `uiverify --fixture` first runs the locator/visibility engine (`internal/uiverify/hv.js`, which replaces Playwright's validated locators) against an embedded page of known content (golden-good and golden-bad cases: hidden by display/visibility/zero size must NOT be visible, an absent label must NOT be found, a covered, disabled or hidden button must refuse a click); `uiverify_live.sh` runs it before the product checks and fails the run if the engine is wrong. Product negative controls: `qa/isolation/make_ui_mutants.py` builds replayable mutants of the built bundle (exact replacements recorded with sha256). Known semantic differences from the Python/Playwright version are listed in `PARITY.md`.

## Tests

`go test -p 2 ./...` (hermetic: real `go test` on a temp module for the mutation runner; a fake page for the UI check logic, which is
UNIT-CLASS evidence only). `gomutate --selftest` kills seeded mutants of the runner and of the UI checks. The live evidence is the container run.
