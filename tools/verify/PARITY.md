# Behaviour parity: Python verifiers vs Go port

Sources: `scripts/verify_go_mutation.py`, `scripts/verify_ui.py` (both kept, nothing removed).
Evidence (sealed, sha256 in `SHA256SUMS`): `evidence/go-verify-port-2026-11-24/`.

## verify_go_mutation.py -> `cmd/gomutate`

| Behaviour | Go | Evidence |
|---|---|---|
| 104-mutant catalogue, same ids/files/anchors/tests/expect | embedded `catalogue.json`, generated from the Python `MUTANTS` | mutation-parity.json: equal id sets |
| Copies every top-level module into a scratch tree; originals never mutated | yes, hashed before and after | `original_tree_unchanged: true` in both results |
| Anchor must occur exactly once | yes ("mutation anchor count N") | runner mutant `runner-accepts-absent-anchor` killed |
| Positive controls run first and must pass | yes (76 controls) | both results: controls 76 |
| FAIL needs failing test AND predicted assertion in output | yes | `runner-ignores-predicted-assertion` killed |
| Build failure, timeout (124), wrong-assertion are not a kill | yes | killed: `-build-failure-marker`, `-timeout-read-as-verdict` |
| Verdict and reason strings | identical | 104 ids, 0 status/reason differences |
| Summary | pass 104 / fail 0 / controls 76 / with_predicted_assertion 104 | both PASS |
| Added (not in Python) | `--only` (unknown id is an error), `--list`, `--selftest`, scratch removal only of a marked strict descendant of `.helix-verify` (11.4.287) | 25 self-test mutants, all killed |

## verify_ui.py -> `cmd/uiverify`

| Behaviour | Go | Evidence |
|---|---|---|
| 16 checks x 4 viewports (375x812, 768x1024, 1024x900, 1440x1000), reduced motion | yes | check-id set compared with the Python source in a test; live 64/64 |
| Report schema (`schema_version`, `viewports`, `summary`) | same | `TestReportJSONShapeMatchesPython` |
| A viewport error becomes one FAIL `viewport-run`, other viewports still run | yes | unit test plus runner mutant |
| Runs only inside the sanctioned isolation container | `RequireContainer` needs `/run/.containerenv` AND the read-only `launch.sh` marker AND `HRDISO_ISOLATED=1` AND loopback-only network AND no host display env; exit 3, no override. It cannot prove rootlessness itself: that is `container_proof.json` of the launcher | unit test with 8 refusal cases, mutants `ui-browser-launch-allowed-outside-container`, `ui-isolation-env-not-required`, `ui-host-network-allowed` killed |
| Locator/visibility engine (`hv.js`) is correct | `uiverify --fixture`: 30 golden-good/golden-bad cases on a page of known content, run in the container before the product checks | `ui-live/good/fixture`; engine mutants `isVisible -> true` and `-> false` FAIL the fixture (`ui-live/hv-mutant-*`) |
| Live run | container, network none, caps dropped | ui-live/good: 64 pass / 0 fail |
| Detects real regressions (product negative controls) | six replayable bundle mutants, each failing exactly its predicted check ids | `ui-live/mutant-*` (see the evidence README for the table); 7 of 16 check ids have a live negative control: `reader-next-delta`, `console-errors`, `missing-assets`, `sample-data-disclosure`, `source-validation`, `source-scan-state`, `sync-conflict-delta`; the other nine have none yet |

## Intentional differences (UI)

- `.first` is lenient (no Playwright strict mode).
- Console errors also include log-entry errors (UNCONFIRMED whether Playwright's console events include them).
- `Attr` and `Style` wait for the element like Playwright and fail on timeout (a missing element is a `viewport-run` FAIL); an attribute that is absent on a present element reads as empty.
- `copyTree` follows symlinked directories and copies their contents like `copytree(symlinks=False)`.
- Chromium is launched once per viewport (fresh storage per viewport, like a new Playwright context); every browser call has a deadline of twice the action budget.
- `scripts/verify_ui_mutation.py` is referenced by older docs but does not exist in the tree: nothing to port.
- `getByText` is approximated: deepest element, case-insensitive substring unless exact.
- A no-op `Reset filters` count call is dropped.
- Actionability poll with a 15 s default timeout.
- Chromium comes from the ms-playwright build inside the container image.

## UNCONFIRMED

- Live Python-vs-Go UI run on identical input: the container image has no python `playwright`, and a host run is forbidden. Parity rests on check-id comparison plus live Go positive and negative controls.
- Playwright vs chromedp edge cases (strict mode, text matching, console capture breadth).
- Only the `demos/interactive_static_demo` prototype was exercised; its dist was built online on the host (non-GUI) and is not tracked.
- chromedp v0.15.1 / cdproto 20260405 were taken from the offline module cache.
