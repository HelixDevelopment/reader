HRD-049 / GAP-014: go test -race on core and server at github/main f9d14dc (clean worktree, no uncommitted changes), 2026-11-24.
Commands are the header lines of each log (run.sh): cd <module> && GIN_MODE=release GOMAXPROCS=4 go test -race -p 2 ./... -count=1 -v
Result: both modules exit 0, zero DATA RACE lines (counts in README below are from grep over the logs).
NOT covered (UNCONFIRMED): paths gated on live PG/Redis/HELIX_* environment are SKIPped in the logs (see SKIP lines); no live PG/Redis was available
to this run, so the original proposed fix "with live PG/Redis" is only partly met. No emulator/GUI involved.
core ok-packages=25 DATA_RACE=0 SKIP-lines=0 FAIL-lines=0
server ok-packages=13 DATA_RACE=0 SKIP-lines=49 FAIL-lines=0
