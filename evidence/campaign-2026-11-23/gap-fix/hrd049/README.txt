HRD-049 / GAP-014: go test -race on core and server, current worktree (HEAD 652c03f + uncommitted), 2026-11-23.
Exact commands are the header lines of each log:
  core:   cd core && go test -race ./... -count=1
  server: cd server && GIN_MODE=release go test -race ./... -count=1 -v   (hrd049_server_race_verbose.log, exit 0)
core: all packages ok. server: all packages ok, exit 0 (re-run after the review fixes, including the soak and remote tests).
47 SKIP lines remain in the server run (live PG/Redis/HELIX_*-gated tests): race evidence does NOT cover those paths (GAP-015).
hrd049_core_race.log is the earlier core run (no header); re-run below for the header form.
