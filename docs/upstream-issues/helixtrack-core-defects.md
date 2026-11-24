# DRAFT (not filed): Helix-Track/Core defects found while building the HRD-029 sync

Source (D1-D5 are as reported by the HRD-029 agent; this drafting pass did not re-run them, only checked the evidence paths exist): `docs/helixtrack-sync.md` section "Core quirks found" (all reproduced against a self-hosted Core instance).
Evidence root: `evidence/campaign-2026-11-23/helixtrack/` (`core-instance.txt` identifies the instance and build; traffic
files are in `live/`, with `jwt`, `password`, `token`, `Authorization` redacted). Nothing has been filed.

Severity order is by user impact, not by the numbering in the sync doc.

## D1 (security): raw `/do` request bodies, including passwords, are logged
Reproduce: run `authenticate` against `POST /do` (see `live/00-server-identity.jsonl`, `live/01-dryrun.traffic.jsonl` for the request shape) with Core's logging at its default, then inspect Core's log. The body with the password is present in clear text.
Expected: credentials redacted or the body not logged. Workaround: treat Core logs as secrets. Pointer: `core-instance.txt` for log location.

## D2 (security): default JWT secret is a hard-coded public string
When no secret is configured, tokens are signed with a publicly known constant, so anyone can mint an admin token (HS256).
Reproduce: start Core with no JWT secret configured, `authenticate`, and verify the returned token against the public constant. UNCONFIRMED here which source file holds the constant; the behaviour was inferred from live tokens accepted under the default config (see `live/02-push1.traffic.jsonl` for a token-bearing flow, token redacted).
Expected: refuse to start without a secret, or generate a random one.

## D3: shipped `Application/Database/Definition.sqlite` lacks `ticket.version`, `project.version` and `ticket_history`
Effect: `create project` fails and is mislabelled HTTP 409 "already exists" instead of a schema error. The first-start migrator
over the root `Database/DDL` fails on its own MySQL-style `INDEX (...)` DDL under SQLite.
Reproduce: start Core with the shipped DB, `create project`; compare with the same call after applying
`evidence/campaign-2026-11-23/helixtrack/core-local-schema-patch.sql` (3 statements, used for our local instance). Sink-side DB check: `live/13-sink-side-core-db.txt`.
UNCONFIRMED whether upstream Docker images carry a corrected DB.

## D4: `modify` ignores `type` and silently drops an unknown `status`
`modify` with `type` has no effect; an unknown `status` alongside other fields is dropped without error, but alone returns 400 "No fields to update". A caller cannot tell a no-op from success.
Reproduce: `live/08-push3.traffic.jsonl` and `live/10-push4.traffic.jsonl` (updates), compared with `live/12-pull3.report.json` and `live/13-sink-side-core-db.txt` for the stored values.
Expected: reject unknown fields/values with a 4xx naming the field.

## D5: `read`/`list` omit `version`, and `modified` has 1-second resolution
The optimistic-lock `version` is writable evidence but not readable, so clients must probe (modify with a stale version) to learn it; `modified` (1 s) cannot serve as a change detector. Reproduce: `live/03-compare1.report.json`, `live/09-stale.result.json` (stale-version rejection) and `live/06-remote-edit.result.json` (remote edit detected only via content hash).
Expected: include `version` in `read`/`list`; sub-second or monotonic change stamp.
