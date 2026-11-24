# Pull-request delivery (constitution section 11.4.297)

Status words (11.4.297(G)): the delivery tool is PROVISIONAL (exercised only against a stub platform, `tests/test_pr_deliver.py`,
until a real PR has been read back). Branch protection of `main` on GitHub is UNCONFIRMED (the token cannot read it). The local
direct-push guard declared as `expected` is not installed: UNCONFIRMED.

Decision: `PRD-2026-11-23-1` (operator, 2026-11-23: "switch to PR driven work -> you create PRs, you do review by independent
agents spawned, and you merge PRs into the main!"), standing together with the 2026-11-16 mandate that all PRs are created, reviewed
and merged by the agent and assigned to the main repository owner. Everything is DATA in `config/delivery.conf`; an unknown key or a
value outside its closed set makes `scripts/pr/deliver.py` refuse.

## Rules the tool enforces

- One task id, one branch `task/<HRD-id>-<slug>`, one open PR. Every commit carries a plain `Roadmap-Item: HRD-nnn` trailer (two ids
  need a `Multi-Task-Reason:` trailer). A new branch starts at the remote base tip; commits on HEAD the remote base lacks are refused.
- Push goes to remote `github` only (never `origin`, which also pushes to the mirror), fast-forward only, never force, and the remote
  tip is read back with `git ls-remote`. `run()` refuses `--force*`, `+refspec`, `--squash`, `--rebase`, `--admin`, `-D`.
- Landing is a merge commit: first parent = previous `main` tip, second = reviewed head, checked after the merge. Squash and rebase are
  forbidden; if the platform lands one anyway the tool reports a VIOLATION and KEEPS the task branch.
- The reviewed head contains the remote base tip. If `main` moved: `git merge github/main` INTO the task branch (never rebase), push,
  re-review, new GO.
- The independent source-correctness GO (11.4.142/.209/.235, producer != verifier) for exactly the merged head is merge condition (a).
  The author's self-review is never the GO.
- Reviewer list is author-only (D)(6) with policy `report-and-merge`: `open` ends with exit code 3, posts a `HELIX-REVIEWER-REPORT`
  comment with the platform's refusal text, and `merge` accepts that read-back report for condition (c).

## Workflow (one task)

```bash
D="python3 scripts/pr/deliver.py"; T="--task HRD-123 --slug short-name"
$D plan $T                       # read-only: config, branch, base tip, ahead/behind, trailers, mirror parity table
$D start $T                      # branch from the remote base tip and switch to it
# ... commit; every commit message ends with:  Roadmap-Item: HRD-123
#     evidence a commit cites: a line  "Evidence: evidence/<path> sha256 <64 hex>"  (also "red|green <path> sha256 <hex>")
$D open $T --test-types "unit,integration"     # trailers + evidence checks, ff-only push, ls-remote read-back, PR, reviewers; exit 3 = expected author-only report
# spawn the independent reviewer agent on the exact head; it writes evidence/reviews/<id>-<sha8>.txt containing the lines
#   VERDICT: GO     and     HEAD: <full 40-hex sha>
$D self-review $T --head <sha> --body-file <my-reading-record.txt> --findings-open 0
$D review-record $T --head <sha> --reviewer-label "<reviewer agent id>" --author-label "<author agent id>" --verdict-file evidence/reviews/<file>
$D merge $T                      # refuses unless every condition (a)-(g) reads back; then reads back state, merge commit, parents, ancestry
$D mirror $T                     # ff-only push of the merged main to m3w, then PRESENT / MIRROR-LAG / UNREACHABLE table
$D retire $T                     # deletes the branch only if it is an ancestor of main on github AND every mirror
```

Add `--dry-run` before the subcommand (`deliver.py --dry-run open ...`) to print every mutating git/gh command without running it.

## Evidence citations: the latest citation of a path decides (clause E)

Every `Evidence:` / `red` / `green` citation in every commit of the PR range must be repository-relative and under `evidence/`
(absolute, `..` or outside-dir paths are INCONCLUSIVE and refused, even when a later commit cites the same path again). The cited
sha256 must equal the hash of the path in the head tree for the LATEST citation of that path (newest commit first in `git log --topo-order`;
two citations of one path in the same commit are both treated as latest). Each EARLIER citation of the same path is SUPERSEDED (a later commit
legitimately regenerated the file) but must still equal the path's content in ITS OWN citing commit's tree (`git show <commit>:<path>`;
absent there = refused), so a forged or wrong earlier hash is still refused. Superseded citations are printed as
`NOTE: evidence <path>: earlier citation(s) superseded ...`. A path cited once, an absent path, or a mismatch on the latest citation is refused
exactly as before. Limit: with parallel branches and skewed commit dates "latest" may be a commit that is not after the other citing commit;
that fails closed (a false refusal), never a false pass.

## How the independent GO is produced and verified

1. The reviewer is a separately spawned agent that is not the author (its label must differ from the author label). It is given the
   head sha, reads the whole diff and runs the gates. Its verdict file names `HEAD: <sha>` and `VERDICT: GO`.
2. The verdict file lives under `evidence/` (clause E). It cannot be committed into the task branch before the review (a commit would move
   the reviewed head); keep it untracked, or commit it in a later PR that records the GO.
3. `review-record` refuses a head that is not the PR's current head, an equal author/reviewer label, a verdict without both lines, or a
   path outside `evidence/`. It stores `.git/helix-pr/go/<head>.json` (head, labels, path, sha256) and posts `HELIX-GO head=... sha256=...`.
4. `merge` re-hashes the verdict file, requires the record for the exact PR head and the `HELIX-GO` comment for that head on the platform.
   A new commit changes the head and therefore needs a new GO and a new self-review. `HELIX-GO` comments are written by the same account
   as everything else here (one gh account); the independence rests on the separate reviewer agent, not on the platform identity (honest boundary,
   11.4.297(D)(6)).

## Base moved / submodules / mirrors

- Base moved: `merge` refuses (g). Merge `github/main` into the task branch, push via `open`, obtain a new GO.
- Submodule changes (clause F): own-org submodules get their own repository PR on a branch of the same name; it lands first (merge commit),
  is mirrored, and only then does the parent PR bump the gitlink; the parent PR body names the submodule PR, its merge commit and a parity
  table. This tool currently operates on ONE repository (`--repo`): run it once per repository with that repository's config; the
  cross-repository ordering and the pin-bump parity gate (`CM-SUBMODULE-PIN-MIRROR-PARITY`, OWED-GATE-184) are NOT implemented: UNCONFIRMED.
  Third-party submodules are never committed to.
- Mirror `m3w`: `mirror` pushes the merged `github/main` commit by plain push (ff-only by nature). Whether the project has another
  sanctioned push path (11.4 section 2.1) than a direct push is UNCONFIRMED. A diverged mirror is refused, never forced.

## Verification

```bash
python3 -m unittest tests.test_pr_deliver          # hermetic: local bare remotes + stub gh, no network (PROVISIONAL evidence)
python3 scripts/pr/mutate_deliver.py               # paired mutants on temp copies; control first; all must be KILLED
scripts/gates/run_local_gates.sh                   # includes the two canon pr-delivery gates and config validation
```

## UNCONFIRMED / not implemented

- Platform branch protection and merge-method restrictions of `HelixDevelopment/reader` (squash/rebase disabled in repo settings): not readable with this token.
- Local direct-push guard (`direct_push_guard=expected`): not installed.
- A real PR round-trip against GitHub: never run; the stub models `gh` fields used (`pr list/view/create/edit/comment/review/merge`) and may differ from the real CLI.
- Required approvals (condition (f)): none configured, so not checked. Merge window staleness report (24 h): not implemented.
- `ff-only` landing form: not implemented (the tool refuses it); only `merge-commit`.

## Authentication (operator decision, 2026-11-24)

The `GITHUB_TOKEN` fine-grained token in the shell environment cannot create pull
requests ("Resource not accessible by personal access token (createPullRequest)").
The operator chose to run the PR tool's `gh` calls with that variable unset so `gh`
uses the stored keyring login of the same account (scopes repo, read:org, gist,
admin:public_key). Run every PR-tool command as:

    env -u GITHUB_TOKEN -u GH_TOKEN python3 scripts/pr/deliver.py <subcommand> ...

Git pushes keep using SSH. The token is never printed or written to a file.
