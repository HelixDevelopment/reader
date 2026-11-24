HRD-084 pin bump: submodules/constitution 74c415d3 -> 7ac2926f (upstream main at 2026-11-24; 3 commits, ancestor-checked:
git merge-base --is-ancestor 74c415d3 7ac2926 => yes; commits 17db5ed3, 002f54b3, 7ac2926f all about the section 11.4.299 parity verifier).
post_pull_validation.log: constitution validators run in a worktree where ONLY submodules/constitution is initialised.
  Verdicts are BLIND/FAIL because nested own-org submodules (claude_toolkit, containers, llmctl, hub engines) are not checked out
  in this worktree. These are NOT PASS. Recursive init + post_update_hook steps outside the repo are UNCONFIRMED / operator-gated
  (docs/constitution-update-plan-2026-11-23.md section 5).
local_gates.log: scripts/gates/run_local_gates.sh at the bumped pin, 7/7 PASS.
