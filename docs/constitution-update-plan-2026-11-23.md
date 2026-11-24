# Constitution post-update plan (2026-11-23)

| Field | Value |
|---|---|
| Script | `submodules/constitution/scripts/post_update_hook.sh` (1370 lines, read in full) |
| Constitution pin | `6d93ccd19e1c2fe0e1a97ece22b49e962ef3005e` (parent HEAD 652c03f) |
| Governing anchors | §11.4.164 (auto-propagation), §11.4.32 (post-pull validation), §11.4.228(D) (label-sync wiring), §11.4.234 (hook checks are a stage of a dedicated commit/push script, never a blocking pre-push), §11.4.272 (skill activation), §11.4.281(C) (Jev bootstrap) |
| Status | Plan + in-repo skill links APPLIED; everything outside the repo AWAITING OPERATOR |

## 1. The script has no global dry-run flag

Only `--project-root DIR` is accepted. Dry-run exists per step by env var: `LABEL_SYNC_AUTOWIRE`, `LABEL_SYNC_ADAPTERS`,
`AGENT_LABEL_DEFS_AUTOWIRE` (all default dry-run) and `CONSTITUTION_JEV_HOOK=off`, `CONSTITUTION_SYNC_CHECK`. The plan below was
therefore produced by static trace plus each sub-installer's own `--check` / `--dry-run`.

Change detection used by a bare run: `git diff --diff-filter=AM ORIG_HEAD..HEAD` inside the constitution submodule
(ORIG_HEAD = 6374feb7 exists here) = 8048 files. Relevant classes found: 6 skill files, 0 `mcp/*.json`, 1 real git hook name
(`post-merge`; other scripts/hooks files are skipped as not git hook names), actions/plugins files, 16 label-sync files, 8 `scripts/jev/*` files,
`submodules/claude_toolkit` gitlink. Operator env: `TYPESAFE_API_KEY`, `OPENROUTER_API_KEY`, `LLMCTL_API_KEY`, `CLAUDE_CONFIG_DIR`
are set (values not read), so a Jev key is "real".

## 2. Step-by-step effect table

| Step | What a bare run does | Location | Reversible | Decision |
|---|---|---|---|---|
| 1 detect | read-only git diff | submodule | n/a | ran as part of trace |
| 5b hub gates | runs 13 `cm_*` gates read-only | none | n/a | RAN (section 4) |
| 5c upstream sync | `sync_all_upstreams.sh --check --no-fetch`, offline, read-only (writes a report under `~/.cache/helix_sync_upstreams/`) | cache dir | yes | RAN (section 4) |
| 4f agent label defs | DRY-RUN default: would write 10 files `.claude/agents/T1-main-{sonnet,opus}-{low..max}.md` | in repo | yes (delete) | NOT applied (dry-run output only; adds 10 agent types, needs operator consent) |
| 2 skills | for CHANGED skills: symlink `skills/<n>` into `<project>/skills/` and run each `register.sh` | in repo | yes | NOT applied as written (see 4b) |
| 4b plugins | `install_cli_agent_plugins.sh <project>`: (i) regenerates `plugins/helix/commands/*` and `actions/generated/*` INSIDE the constitution submodule, (ii) links 11 skills to `.claude/skills/`, (iii) `wire_prompt_hook` edits project `.claude/settings.json` (UserPromptSubmit `action_prefix_expand`), (iv) `claude plugin marketplace add` + `install` in `$CLAUDE_CONFIG_DIR` | repo + submodule + user | partly | only (ii) APPLIED; rest AWAITING OPERATOR |
| 4b skill activation | `skill_activate.sh session-init`: PRUNE demotes non-core skills (the 15 tracked `speckit-*` links would be demoted/unlinked), then activates core | in repo `.claude/skills` | per its docs yes, but changes tracked files | AWAITING OPERATOR (not applied) |
| 4 git hooks | copies `post-merge` into `.git/hooks/` (backup first). Hooks dir is repo-local (`.git/hooks`, no core.hooksPath). Effect: every future merge re-runs this hook incl. Jev bootstrap, which conflicts in spirit with §11.4.234 | `.git/hooks` | yes | NOT applied; AWAITING OPERATOR |
| 3 MCP | no `mcp/*.json` changed, so nothing merged | none | n/a | no-op |
| 2b jev MCP repair | STALE_JEV_MCP=1 because Jev files changed: inspects project `.mcp.json` (absent) and `~/.mcp.json`; deletes the `jev` entry only if it still holds an unexpanded `${`; timestamped backup `~/.mcp.json.prejev.<ts>` first | OUTSIDE (`~/.mcp.json`) | yes (backup) | AWAITING OPERATOR |
| 4c label-sync | DRY-RUN default. With `LABEL_SYNC_AUTOWIRE=apply` it merges Stop/SessionEnd/etc. hooks and a statusLine into project `.claude/settings.json` (backup first) | in repo | yes | dry-run output captured, NOT applied |
| 4d adapters | only if `scripts/label-sync/adapters/*` changed (it did). DRY-RUN default; `apply` edits the USER configs of kimi/opencode/crush/aider/codex/gemini/qwen under `$HOME` | OUTSIDE | via backups | AWAITING OPERATOR |
| 4e Jev bootstrap | because Jev files changed AND a real key is present, runs `bootstrap.sh --non-interactive --default-on` (timeout 900 s). Per its header: one authenticated HTTPS probe per hosted pin (network), may clone `claude_toolkit` into `~/.local/share/helix`, and under default-on runs `jev-setup` which edits AGENT HOMES (e.g. `~/.claude*` settings/MCP), records `~/.local/state/jev/consent.json` | OUTSIDE + network | `jev-setup --uninstall` | AWAITING OPERATOR |
| 5 validate | `chmod +x` untracked scripts, `bash -n`, shellcheck | submodule | yes | read-only in practice |

## 3. Applied (in-repo, reversible)

Pre-state backup: `/home/milosvasic/.cache/claude-tmp/claude-1000/-home-milosvasic-Projects-helix-reader/4783d69d-a063-560b-d52a-4fcb6adc6609/scratchpad/pre-hook-backup.tgz`
(`.claude`, `.git/hooks`, `.gitmodules`); `git status` before/after saved beside it as `status.before` / `status.after`.

Command: `bash submodules/constitution/scripts/install_cli_agent_plugins.sh --skills-only --project-root .`

Result: `linked 11 skill(s) into .claude/skills/ (0 refused, 0 failed)`. Diff of `git status`: exactly 11 new untracked
relative symlinks (`.claude/skills/{action-prefix-system,constitution,jev-decision-layer,label-sync,media-validator,multitrack,
reporting-workable-items,scheduled-work-queue,session-sync,skill-catalog,workable-item-lifecycle}` -> `../../submodules/constitution/skills/<n>`).
No tracked file changed; constitution submodule stayed clean.

Undo: `cd .claude/skills && rm action-prefix-system constitution jev-decision-layer label-sync media-validator multitrack reporting-workable-items scheduled-work-queue session-sync skill-catalog workable-item-lifecycle` (symlinks only), or restore from the tarball.

Note: these 11 skills raise per-session skill-listing cost; §11.4.272 prefers on-demand activation. Operator may prefer to remove all but `constitution` and `skill-catalog`.

## 4. Validator verdicts (pasted)

```
verify_helix_deps_manifests.sh --root .   VERDICT: PASS (pass=21 fail=0 blind=0 skip_third_party=18 known=0)
CM-HUB-ENGINE-NO-OWNORG-NESTING: PASS
CM-LLMCTL-SUBMODULE-NONRECURSIVE: PASS
CM-FULL-SUITE-MANIFEST-COMPLETE: PASS (manifest matches the census, 1061 files; control needle seen)
CM-HELIX-DEPS-MANIFEST-PRESENT --root .: PASS
sync_all_upstreams.sh --check --no-fetch: RESULT: ATTENTION exit=4
    submodules/containers behind; doc_processor, llm_orchestrator, llm_provider, vision_engine: attention pin-not-on-main
install_cli_agent_plugins.sh --check (after skill links): 11 skill symlinks present; plugin 'helix' installed+enabled;
    WARN UserPromptSubmit prefix hook NOT registered in .claude/settings.json; VERDICT: NOT-WIRED
```

The other 9 gates in step 5b were not run individually; `run_full_suite.sh` was not run (long, writes evidence). Both are
available to the operator. ATTENTION from the sync check is a real finding: four own-org submodule pins are not on `main`
(§11.4.288) and `containers` is behind its upstream; not fixed here (needs operator approval, no push).

## 5. AWAITING OPERATOR (outside repo / behaviour-changing) — exact commands and risk

1. Jev bootstrap + agent-home wiring (largest effect). Command: `bash submodules/constitution/scripts/jev/bootstrap.sh --non-interactive --default-on`. Risk: network probe with real keys, clone into `~/.local/share/helix`, edits agent homes. Reversible by `jev-setup --uninstall`. Safer: omit `--default-on`, or `touch .jev/bootstrap.disabled`.
2. `~/.mcp.json` stale-jev repair (only fires if an unexpanded `${` entry exists; backup `~/.mcp.json.prejev.<ts>`). Command: run the hook with `CONSTITUTION_JEV_HOOK=off` is NOT enough; step 2b has no switch.
3. Claude plugin registration: `claude plugin marketplace add <abs>/submodules/constitution` and `claude plugin install helix@<marketplace>` (`--check` says helix is already installed and enabled, so likely a no-op).
4. Project `.claude/settings.json` edits: `LABEL_SYNC_AUTOWIRE=apply` (hooks + statusLine) and `wire_prompt_hook` (prefix hook). Risk: changes harness behaviour for every session in this repo; backup is made first.
5. Adapters: `LABEL_SYNC_ADAPTERS=apply` edits other agents' user configs.
6. 10 agent-type files: `AGENT_LABEL_DEFS_AUTOWIRE=apply`.
7. `post-merge` git hook install and `skill_activate.sh session-init` prune (would demote tracked `speckit-*` links).

Recommended invocation if the operator wants the in-repo remainder without any outside effect:
`CONSTITUTION_JEV_HOOK=off LABEL_SYNC_ADAPTERS=off bash submodules/constitution/scripts/post_update_hook.sh --project-root .`
Caveat: this still performs 2b (a read of `~/.mcp.json`, write only with backup), `claude plugin` registration, skill prune and the `post-merge` install.

## 6. Fetch result and applied operator-approved steps (2026-11-23, late)

| Item | Result |
|---|---|
| Fetch | `origin`, `github`, `gitlab`, `gitflic`, `gitverse`, `vasic_digital_github`, `vasic_digital_gitlab` all fetched; `main` of every one is `6d93ccd1` (tag `helixconstitution-v73.4-3-g6d93ccd1`). The pin is already the newest canon; the pin did NOT move. |
| Recursive fetch | `git fetch --recurse-submodules` inside the constitution fails: `no submodule mapping found in .gitmodules for path 'submodules/challenges'` (from nested `claude_toolkit`); canon-internal defect, UNCONFIRMED root cause. |
| Anchors | canon defines 11.4.1..11.4.298; user-scope index stops at 288. Newer than the index: 289-298 (298 = release gate per product; 297 = pull-request delivery). |
| Applied | `install_action_prefix.sh .` (UserPromptSubmit hook), `install_label_sync.sh --apply --project-root .` (hooks only, no statusLine), `.git/hooks/post-merge` installed, `jev/bootstrap.sh --non-interactive` (detect only; RESULT rc 5 "detected, not wired"; agent wiring NOT consented). New untracked: `.claude/settings.json` plus 2 `.bak` files (do not commit the `.bak` files). |
| Gates | `scripts/gates/run_local_gates.sh` 4/4 PASS; 13 hub gates incl. CM-PR-DELIVERY-ANCHOR-WIRED PASS. |
