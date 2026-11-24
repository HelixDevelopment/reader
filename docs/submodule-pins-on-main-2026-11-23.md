# Submodule pins vs `main` (2026-11-23)

| Field | Value |
|---|---|
| Scope | The 11 top-level submodules of `helix_reader` (constitution's nested ones excluded) |
| Mode | READ-ONLY investigation. Only `git fetch --all --prune` into each submodule's own object store. No checkout, no commit, no push, no pin moved |
| Parent HEAD | `652c03f` (branch `main`) |
| Governing anchors | §11.4.288 (all work on `main`, recursively), §11.4.195 (branch governance), §11.4.113 (never force-push, merge onto latest `main`) |
| Trigger | `submodules/constitution/scripts/sync/sync_all_upstreams.sh --check --no-fetch` reported ATTENTION: containers behind upstream; doc_processor, llm_orchestrator, llm_provider, vision_engine pinned off `main` |
| Fetch result | All 10 non-constitution submodules fetched fine (including `llms_verifier`, 383 MB object store; the fetch finished in seconds because the delta was small). One broken remote entry in `llm_orchestrator` (`sync-all`, see 3.4) |
| Status | Findings only. Every action below is a recommendation awaiting operator decision |

## Contents

1. Summary table
2. Cross-cutting finding
3. Per-submodule detail (3.1 containers ... 3.11 vision_engine)
4. Build/API evidence for helixqa's replace targets
5. Recommended operator actions
6. Method and caveats

## 1. Summary table

Notation: "tip+N / pin+M" means main tip has N commits not in the pin, the pin has M commits not in main tip.

| Submodule | Pin | Pin on `main`? | main tip vs pin | Pin is on | Recommendation |
|---|---|---|---|---|---|
| challenges | `d94e554` | YES (all 5 upstream mains) | 0 / 0 | main | KEEP-PIN (already on main) |
| constitution | `6d93ccd` | not examined (out of scope here; synced by its own script) | n/a | n/a | not assessed |
| containers | `22b4970` (tag vasic-1.0.0-dev-0.0.7) | YES, ancestor of main | tip+25 / pin+0 (github/origin/upstream main `aa99d46`); gitlab and vasic_digital_gitlab main `5ea870a` is +9 | main (behind) | FAST-FORWARD-TO-MAIN (safe, see 3.2); note the two upstream mains differ |
| doc_processor | `12a5fb7` | NO | tip+1 / pin+126 | `master` only (all 5 remotes) | DIVERGED (operator/upstream action) |
| helixqa | `b16a089` | YES (all 5 upstream mains) | 0 / 0 | main | KEEP-PIN (already on main) |
| llm_orchestrator | `8ee5795` | NO | tip+0 / pin+1 (main is a strict ancestor of the pin) | `master` of github/origin/github-helix | KEEP-PIN until upstream fast-forwards `main` to the pin (upstream action) |
| llm_provider | `bd302c8` | NO | tip+0 / pin+2 (main is a strict ancestor of the pin) | `master` of all 5 remotes | KEEP-PIN until upstream fast-forwards `main` to the pin (upstream action) |
| llms_verifier | `94e666d` (tag vasic-1.0.0-dev-0.0.7) | YES (github/gitlab/origin main) | 0 / 0 | main | KEEP-PIN (already on main) |
| security | `0513361` (tags 0.0.6, 0.0.7) | YES | 0 / 0 | main | KEEP-PIN (already on main) |
| superspec | `c20ac6c` (tag v1.0.2) | YES (origin/main) | 0 / 0 | main | KEEP-PIN (already on main; third-party WangX0111/superspec) |
| vision_engine | `1b2b991` | NO | tip+0 / pin+1 (main is a strict ancestor of the pin) | `origin/master` | KEEP-PIN until upstream fast-forwards `main` to the pin (upstream action) |

## 2. Cross-cutting finding

For llm_orchestrator, llm_provider and vision_engine the pin is NOT behind or beside `main`; it is AHEAD of it. In each, `origin/main` is a strict ancestor
of the pinned commit, and the extra commits live on `master`. The extra commit(s) are the Oct 26 2026 cascade
"Markdown-only docs - formats export OFF by default, remove generated PDF/HTML/DOCX exports" (only deletions of `*.pdf`/`*.html` exports and no Go changes:
`git diff pin origin/main -- go.mod '*.go'` is empty for all three). llm_provider additionally carries the merge commit `bd302c8` "Merge remote-tracking branch 'github/master'".

Consequence: "fast-forward the pin to main" would move the pin BACKWARDS (re-adding generated export files that the project deliberately removed). It is the
upstream `main` that needs to catch up, via a fast-forward push of `master` to `main` (no merge needed, no force). Containers shows the same
repair was already done upstream on 2026-11-05 (`cd65e1d` "Merge gh/main ... (section 11.4.288)", `a4cbf32` "keeping the Markdown-only docs decision"), so this is the established pattern.

## 3. Per-submodule detail

### 3.1 challenges
- Remotes: origin/github `git@github.com:vasic-digital/Challenges.git`, gitlab `git@gitlab.com:vasic-digital/challenges.git`, vasic_digital_github `.../challenges.git`, vasic_digital_gitlab `git@gitlab.com:vasic-digital/challenges.git`.
- Pin `d94e55492f10e29b2bd6eeca3c1be2adbef597f4` == tip of `main` on all 5 remotes. `master` exists on all remotes (github 791208f, gitlab f330765) with the pin NOT an ancestor (0 / 24, 0 / 242): legacy branch, not relevant.
- Recommendation: KEEP-PIN.

### 3.2 containers
- Remotes: origin and upstream `git@github.com:vasic-digital/Containers.git` (case variants), github, gitlab `git@gitlab.com:vasic-digital/containers.git`, vasic_digital_gitlab.
- Pin `22b49707f9bb3175ec51ee259032b1d4fd9d0dc6` is an ancestor of `main` on every remote (branches containing pin: github/gitlab/origin/upstream/vasic_digital_gitlab main). `master` is a legacy line (pin not an ancestor).
- Two upstream mains differ: github/origin/upstream `main` = `aa99d467ff` (pin+25), gitlab and vasic_digital_gitlab `main` = `5ea870a583` (pin+9). The GitLab main is a strict ancestor of the GitHub main (`rev-list --left-right --count github/main...gitlab/main` = 16 / 0); GitLab is simply 16 commits behind GitHub. That is an upstream mirror lag (needs a fast-forward push to gitlab), not a divergence.
- Commits pin..github/main (25, all code of Nov 5 and Nov 13): crossbuild SIGKILL pid<=1 refusal (§11.4.263), test-oracle labels, G1 typed run options (memory/cpus/pids/device/userns/publish/label/digest), G2 host-budget admission (`pkg/hostbudget`), G11 reachable-endpoint resolver, CAT-DEVEMU hardening rounds 2-4, CONT-DENY-STRUCT rounds 1-3 (host-path deny rules, fail closed), nasemu seed corpus (`pkg/nasemu/seed`, `cmd/nasemu-seed`).
- Diff: 51 files, +8068/-32; non-test Go: `pkg/runtime/run.go` (+312/-7), `pkg/runtime/run_policy.go` (+824, new), `pkg/emulator/*`, `pkg/hostbudget/*` (new), `pkg/nasemu/seed/*` (new), `pkg/crossbuild/subprocess_group_unix.go`, `pkg/vm/qemu.go`, `pkg/remote/runtime.go`. `go.mod`: only `golang.org/x/sys v0.45.0 // indirect` becomes direct (no version change).
- Behavioural caution: the new run policy fails closed on dangerous combinations (host-path denies, escalation refusals), so a consumer that passes bind mounts or extra args that the policy denies would now get an error. `WithRunVolumes` and `WithRunExtraArgs` still exist (moved within run.go, lines 426/436). No Go file outside `submodules/containers` references `RunOption`, `WithRunVolumes` or `WithRunExtraArgs` (grep over the parent tree), so no known in-tree consumer of the changed API. UNCONFIRMED: runtime behaviour of consumers (helixqa, llms_verifier, constitution/llmctl) against the new policy; only the compile level was checked (section 4).
- Recommendation: FAST-FORWARD-TO-MAIN, target the GitHub main (the operative upstream; origin). Would-be commands, NOT run:

```
git -C submodules/containers checkout aa99d467ff   # use the full SHA: git -C submodules/containers rev-parse github/main
git add submodules/containers
```

  Re-run the parent's gate/qualification suites afterwards. Separately, GitLab mirrors of containers `main` lag by 16 commits (upstream push, outside this repo).

### 3.3 doc_processor
- Remotes: origin/github `git@github.com:HelixDevelopment/DocProcessor.git`, gitlab `git@gitlab.com:helixdevelopment1/docprocessor.git`, vasic_digital_github, vasic_digital_gitlab (`vasic-digital/DocProcessor`).
- Pin `12a5fb76381a6bb9a1a668c67211ff073967ea44` is the tip of `master` on ALL 5 remotes. `main` exists only on github/origin (`86f1a98350`) and gitlab (`bea9e9edbd`); vasic_digital_* have no `main` at all.
- `main` forked from `master` long ago: merge-base `bea9e9e` (2026-05-28); pin has 126 commits not on main (through 2026-10-26), github/origin main has 1 commit not on master: `86f1a98` (2026-06-22) "govern(cascade): backfill 11.4.98 to 11.4.165 band plus CONST-047 to CONST-059" (AGENTS.md, CLAUDE.md, CONSTITUTION.md, +2158 lines, docs only). gitlab/main (`bea9e9e`) is behind github/main by 1.
- Code impact of moving to main: `git diff --stat pin origin/main -- go.mod '*.go'` = 23 files, +60/-2126, i.e. main lacks the pin's loader changes (e.g. `pkg/loader/markdown.go` -86 lines, tests removed). Moving the pin to main would be a code regression (REGRESSES; not safe).
- Recommendation: DIVERGED. Needs upstream action by the repo owner: merge `origin/main` (one docs-only commit) into `master` (a trivial merge, three governance docs; resolve to the newer text), then fast-forward-push the result to `main` on every upstream (and create/align `main` on vasic_digital_*). Never force. After that the pin (or its merge descendant) is on main. Until then KEEP-PIN.

### 3.4 llm_orchestrator
- Remotes (all fetch URLs): origin/github `git@github.com:HelixDevelopment/LLMOrchestrator.git`, github-helix/github-vasic/gitlab-helix/gitlab-vasic (HTTPS), vasic_digital_github, vasic_digital_gitlab. DEFECT: the submodule has remotes named `push-all`, `setup-remotes` and `sync-all` whose configured URL is garbage (multi-line script output stored as a URL: "Fetching from all remotes... Done."); `git fetch --all` reports `fatal: protocol ... not supported / could not fetch sync-all`. The install_upstreams script wrote command output into remote config. Not changed here (read-only); recommend the operator remove those three remotes.
- Pin `8ee57956359c1d1b609bb9a0aa53887c06969941` is the tip of `master` on origin/github/github-helix and is NOT on any main. `main` on every remote = `c84c410daa`, a strict ancestor of the pin (pin has 1 commit that main lacks, main has 0 that pin lacks). On github-vasic, gitlab-vasic, vasic_digital_* both main and master = `c84c410`, i.e. those mirrors do NOT have the pin commit at all (push gap).
- The 1 commit: `8ee5795` (2026-10-26) cascade "Markdown-only docs ... remove generated PDF/HTML/DOCX exports" (28 files, -6240, no Go change; `git diff pin origin/main -- go.mod '*.go'` empty).
- Recommendation: KEEP-PIN. Do not move it to main (that would re-add the removed exports). Upstream fix: push `master` to `main` as a fast-forward on every remote (`git push <remote> 8ee5795:main`, non-force), also push the pin commit to the vasic-digital mirrors. After that the pin is on main with no pin change.

### 3.5 llm_provider
- Remotes: origin/github `git@github.com:HelixDevelopment/LLMProvider.git`, gitlab `git@gitlab.com:vasic-digital/LLMProvider.git`, vasic_digital_github, vasic_digital_gitlab.
- Pin `bd302c8ab57d0040df87ccbb90c8ccf969bf687b` = tip of `master` on all 5 remotes; main `53e4b2ec00` is a strict ancestor of the pin (pin+2: `5e0dd84` the Markdown-only docs cascade, and merge `bd302c8` "Merge remote-tracking branch 'github/master'"). Go diff pin vs main: empty (docs only; 20 files, -4047).
- Recommendation: KEEP-PIN; upstream fast-forward `master` to `main` (same pattern as 3.4).

### 3.6 llms_verifier
- Remotes: origin/github `git@github.com:vasic-digital/LLMsVerifier.git`, gitlab `git@gitlab.com:vasic-digital/LLMsVerifier.git`.
- Pin `94e666d416f0bde022d784555ec81ddd4389bb04` (tag vasic-1.0.0-dev-0.0.7) == tip of `main` on origin, github and gitlab; 0 / 0 (it was not skipped: fetch completed). The helixqa replace target is `../llms_verifier/llm-verifier` (subdirectory module).
- Recommendation: KEEP-PIN.

### 3.7 security
- Remotes: origin/github `git@github.com:vasic-digital/Security.git`, gitlab. Pin `0513361e2f9cdfa2c3b1da687a0d66e5eb7c8fd1` (tags vasic-1.0.0-dev-0.0.6 and -0.0.7) == main tip on all remotes, 0 / 0. Recommendation: KEEP-PIN.

### 3.8 superspec
- Remote: origin `git@github.com:WangX0111/superspec.git` (third party, not own-org; §11.4.288(A) applies the commit-on-main duty to own repositories only). Pin `c20ac6c1ba069cc9a72dacb8044b7b193d3dde81` (tag v1.0.2) == origin/main tip, 0 / 0. Recommendation: KEEP-PIN.

### 3.9 helixqa
- Remotes: origin/github `git@github.com:HelixDevelopment/HelixQA.git`, gitlab and vasic_digital_gitlab `git@gitlab.com:vasic-digital/HelixQA.git`, vasic_digital_github. Pin `b16a0897d85eb2360038d62a875854d0bfcb217b` == main tip on all 5 remotes. `master` is a legacy line, pin+0/451 behind-by-design (pin not ancestor). Recommendation: KEEP-PIN.

### 3.10 vision_engine
- Remotes: origin `git@github.com:HelixDevelopment/VisionEngine.git`, `GitHub` (vasic-digital/VisionEngine), `GitLab` (vasic-digital/visionengine).
- Pin `1b2b991c31d46fd43447f2456f163032dfdcaf94` = `origin/master` tip, NOT on any main. main = master = `d52fa447d1` on GitHub/GitLab remotes (those mirrors lack the pin commit); origin/main = `d52fa447d1`, strict ancestor of the pin (pin+1: `1b2b991` Markdown-only docs cascade, 30 files, -5603, no Go change; Go diff empty).
- Recommendation: KEEP-PIN; upstream fast-forward `master` to `main` on origin, and push the pin commit to the two mirrors.

### 3.11 constitution
Out of scope of this investigation (it has its own sync script and its own nested submodules). Pin `6d93ccd19e1c2fe0e1a97ece22b49e962ef3005e` (helixconstitution-v73.4-3). Not assessed.

## 4. Build/API evidence for helixqa's replace targets

`submodules/helixqa/go.mod` replace block: challenges, containers, doc_processor, llm_orchestrator, llm_provider, llms_verifier/llm-verifier, security, vision_engine (all `../<dir>`).

| Module | `git diff --stat pin <main tip> -- go.mod '*.go'` | Reading |
|---|---|---|
| doc_processor | 23 files, +60/-2126 (main lacks 126 pinned commits) | main tip would regress code: NOT a candidate |
| llm_orchestrator | empty | main tip is code-identical to the pin; irrelevant to build |
| llm_provider | empty | same |
| vision_engine | empty | same |
| containers (github/main) | 51 files +8068/-32; go.mod: `golang.org/x/sys` indirect to direct | additive API (new options, `hostbudget`, `nasemu/seed`); `WithRunVolumes`/`WithRunExtraArgs` retained |
| security, llms_verifier, challenges | pin == main tip (empty) | n/a |

Compile evidence (this session, scratch copies extracted with `git archive` under the session scratchpad, so nothing in the repo was touched): `GOWORK=off GOFLAGS=-mod=mod go build ./...` of containers at the pin: no errors; at `github/main` (`aa99d46`): no errors; `go vet ./pkg/runtime/` at the tip: no output. This proves only that containers' tip compiles standalone; it does NOT show helixqa (or llms_verifier, constitution/llmctl, which replace containers) builds or passes tests against it. UNCONFIRMED: helixqa build and test with containers at `aa99d46`; run that in a scratch tree before bumping.

## 5. Recommended operator actions

1. containers: decide to bump the pin to `github/main` `aa99d46`; run helixqa/llms_verifier builds in a scratch tree first (section 4). The two commands are in 3.2.
2. llm_orchestrator, llm_provider, vision_engine: no pin change. Ask the upstream owner (or authorize a non-force push) to fast-forward `main` to `master`, and push the pin commits to the mirrors lacking them. That satisfies §11.4.288 with zero pin movement.
3. doc_processor: merge `origin/main` (single docs commit `86f1a98`) into `master` and fast-forward `main` everywhere; this is the only true divergence. Needs upstream action.
4. containers GitLab mirrors: fast-forward push `main` (16 commits behind GitHub).
5. llm_orchestrator: remove the three garbage remotes (`push-all`, `setup-remotes`, `sync-all`); investigate how install_upstreams.sh stored command output as a URL.
6. challenges, helixqa, llms_verifier, security, superspec: nothing to do.

## 6. Method and caveats

- Commands: `git submodule status`, `git remote -v`, `git fetch --all --prune` (background, per submodule), `git merge-base --is-ancestor`, `git rev-list --left-right --count`, `git branch -r --contains <pin>`, `git log pin..tip`, `git diff --stat pin tip -- go.mod '*.go'`. "main" means each remote's `main`; `master` lines were also measured because several pins live there.
- "tip+N / pin+M" counts come from `git rev-list --left-right --count pin...tip` (tip-only / pin-only).
- No submodule worktree or pin was modified; submodule working trees were clean (0 dirty paths) and on the pinned SHA. Fetch only added remote-tracking refs to the submodule object stores under `.git/modules`.
- Test/behavior claims beyond compile level of containers are UNCONFIRMED (section 4).
