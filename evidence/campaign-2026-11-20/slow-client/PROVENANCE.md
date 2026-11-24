# Provenance (added in HRD-067 review round 2)

These result files were recorded by EARLIER versions of the slow-client test code. The SLOWCLIENTJSON line numbers in
raw.log / raw_catalog.log (where present) do not match any committed version of slow_client_live_test.go, and the
assertions in the current test code were strengthened afterwards (deadline discrimination, pending-bound and
running-bound proofs). They are therefore NOT evidence for the code at this head: UNCONFIRMED for this head.
The measurement_path field also differs between files (catalog 2026-11-23: host_hrd027_isolated_container; jobs: no
field; 2026-11-20 copies: host_forwarded); the field was not reconciled. A fresh live run (isolation runner, Redis +
PostgreSQL) writing to a new directory is owed. Integrity only: SHA256SUMS lists the files as committed.
