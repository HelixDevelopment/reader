HRD-048 round 2 (extra review round 1 of 3 for PR #16). NEW dir; the sealed hrd048 dir is untouched.
mutation.log  - mutate.sh (trap-restored): M01/M02 (request mix 404-blind), M07 (R2 conjunct), M08 (warm-up discard),
                M09 (growth limit), M10 (slope limit), M11 (goroutine slop) all KILLED; restored: yes.
GREEN_hermetic.log - go test -run TestSoak (hermetic, in-process; no containers, no Redis, no live legs). The 3 soak_report*.json
                are written by that run (HELIX_SOAK_REPORT). Remote WebDAV soak slope 9.19 B/iter at R2 0.55 is below the floor
                (long run owed: UNCONFIRMED for longer horizons).
Verify: cd evidence/campaign-2026-11-24/gap-fix/hrd048-r2 && sha256sum -c SHA256SUMS
