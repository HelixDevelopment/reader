# Helix Reader — GEMINI.md

## INHERITED FROM submodules/constitution/GEMINI.md

All rules in `submodules/constitution/GEMINI.md` and the `submodules/constitution/Constitution.md` it
references apply unconditionally. This file extends them and never weakens them; where it disagrees
with the constitution submodule, the constitution wins. This carrier is kept in lockstep with
CLAUDE.md, AGENTS.md, QWEN.md and GEMINI.md (§11.4.157): the project sections below are identical in all four.

@submodules/constitution/GEMINI.md

Critical base rules restated for agents that do not follow imports: no bluffing (§11.4); no guessing language (§11.4.6);
never force-push (§11.4.113); no secrets in tracked files (§11.4.10); no CI/CD (§11.4.156); work lands on `main` (§11.4.288),
here through pull requests (§11.4.297); independent review of every change (§11.4.142); 60% RAM cap on heavy work (§12.6).

---

## Project overview

Helix Reader (item key `HRD`): a self-hosted ebook/comic library. Go core (`core/`), Go server (`server/`: OPDS, OPDS-PSE,
KOSync, WebDAV/rclone sources), clients (`clients/`: web, desktop, KOReader, Android), research and QA under `research/`, `qa/`,
`evidence/`, `docs/`. Own-org submodules live in `submodules/` (constitution is pinned there; third-party ones are never edited,
§11.4.288(A)). Dependency manifest: `helix-deps.yaml` (§11.4.31).

## Project-specific MANDATORY constraints

### Delivery: pull-request mode (§11.4.297)

Work is delivered by PR, not by direct push to `main`. The decision is DATA in `config/delivery.conf` (id `PRD-2026-11-23-1`);
the tooling is `scripts/pr/deliver.py`, the workflow `docs/pr-delivery.md`. Rules in short: one task => one branch
`task/<HRD-id>-<slug>` => one PR; every commit carries a plain `Roadmap-Item: HRD-nnn` trailer; push to remote `github` only
(never through `origin`), fast-forward only, never force (§11.4.113); the merge is a merge commit (squash and rebase are
forbidden) taken by the author agent only after the INDEPENDENT source-correctness GO for exactly the merged head
(§11.4.142/.209/.235; the self-review is never the GO); read back, then fast-forward the `m3w` mirror, then retire the branch.
Protection of `main` on the platform and the local direct-push guard are UNCONFIRMED (§11.4.297(G)). No CI/CD (§11.4.156).

### Workable items (`scripts/workable_items.py`, DB `docs/workable_items.db`)

The SQLite DB is authoritative; `docs/Issues*`, `docs/Fixed*` (md/html/pdf) and `docs/tracking/helixtrack-outbox.json` are
GENERATED: never hand-edit, regenerate with `python3 scripts/workable_items.py export`. Check with `validate` and `diff`.
Lifecycle: `add`, `update`, `start`, `pause`, `close`, `reopen`, `operator-block`, `unblock`. Evidence has two tiers
(`start`/`pause`: file or dir under `evidence/`; `close`/`reopen`/`update --evidence`: an existing relative path). `rehash` only
for reviewed out-of-band edits: get the candidate with `rehash --dry-run`, then `--confirm <hash>`. `import-md` is disaster
recovery only; `--allow-overwrite` needs `--actor` and `--reason`. Free text refuses control and invisible characters. Details,
hardening rounds and "Known limits" (the hash is a tripwire, git history is the control): `docs/workable-items.md`.

### Test isolation (HARD RULE, operator, 2026-11-23)

Tests never touch the logged-in user's desktop, display, windows, input devices, clipboard, browser or apps. Every GUI/E2E/
emulator run is inside a rootless container (Containers submodule, driven by HelixQA); the host runs only non-GUI processes.
Policy and architecture: `docs/test-isolation.md`; tooling: `tools/isolation/` (`run_container.sh`, `assert_host_untouched.sh`,
`reap.sh`, guard). Our containers are named `hrdiso-*`; other projects' containers are never touched. No GUI on the host, ever.

### Gates, tests, evidence

- Local gates (the only enforcement; no CI): `scripts/gates/run_local_gates.sh` (self-test `scripts/gates/test_run_local_gates.sh`).
- Evidence manifest and claim-vs-reality ledger: `scripts/gates/build_evidence_manifest.sh`, `docs/claim_reality_ledger.tsv`,
  `docs/evidence/`. Seal an evidence dir with `tools/isolation/seal_evidence.sh <dir>` (writes `SHA256SUMS`); evidence files are
  frozen once recorded. Evidence cited by a PR is a repository-relative path with its sha256 (§11.4.297(E)).
- Qualification: `python3 scripts/verify_workable_items.py`, `python3 -m unittest -v tests.test_workable_items`,
  `python3 -m unittest tests.test_pr_deliver`, Go tests under `core/` and `server/`. Every new gate has a paired mutant (§1.1).

### Anti-bluff and no-guessing

A PASS needs captured runtime evidence from this session (§11.4, §11.4.5, §11.4.107); say `UNCONFIRMED:` instead of guessing
(§11.4.6). Never print or commit a secret; tokens come from the environment (§11.4.10).

## Project overrides of universal rules

None.
