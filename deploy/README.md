# deploy/

- `docker-compose.yml`: executed so far only by podman-compose (podman 5.7.0). **docker-compose is UNTESTED**: no
  `docker compose` CLI run exists; parity is asserted at file semantics only.
- Container distribution was qualified against a local, unauthenticated, TLS-verification-disabled `registry:2` only;
  signing, attestation and a hosted authenticated registry are unverified.
- Evidence: `evidence/campaign-2026-11-20/containers/`. Registry item: HRD-034.
- `postgres/`: see the files in that directory.
