# HelixTrack synchronization (HRD-029)

Status: proven against a **self-hosted local Helix-Track/Core instance only** (rounds 1 and 2, 2026-11-23). Not proven
against any production HelixTrack deployment. Tool: `tools/helixtrack-sync/helixtrack_sync.py` (stdlib Python, never writes
the registry). Evidence: `evidence/campaign-2026-11-23/helixtrack/` (round 1 in `live/`, round 2 in `runs/20261123T212401Z/`,
`latest` points at the newest run; `SHA256SUMS.top` and each run's `SHA256SUMS`).

## Target API (read from Helix-Track/Core @ c85221282d8b, Application/internal)

Single endpoint `POST /do` with `{action, jwt, object, data}`; success is `errorCode == -1`. Used actions only:
`authenticate`, `list`/`read`/`create`/`modify` with `object` = `project` | `ticket` (`remove` only by the live test helper acting as
a second user). Auth: with the external auth service disabled, `authenticate` (local `users` table, bcrypt) returns an HS256 `token`,
accepted as `jwt` body field or `Authorization: Bearer`; users are created by `POST /api/auth/register`. Optimistic locking:
`modify` takes `data.version`; mismatch yields HTTP 409, `errorCode` 3006, current version in `errorMessageLocalised`.

## Mapping (registry outbox record -> Core ticket)

| Registry payload field | Core | Notes |
|---|---|---|
| `client_key` / `atm_id` (HRD-nnn) | title prefix `[HRD-nnn] ` plus trailing marker line `helix-reader-sync/v1 hrd=HRD-nnn` | idempotency key; Core tickets have no external-id/custom-field on create |
| `title` | title | |
| `type` Feature/Task/Bug | `type` feature/task/bug | settable on create only; `modify` ignores it |
| `status` | `status` | Queued->open, In progress->in_progress, Operator-blocked->blocked, Completed/Implemented/Fixed->done; applied by modify after create |
| `description`, `severity`, `reopens_count`, `affected_scope`, `reproduction`, `acceptance`, `evidence_path`, history (read-only from DB with `--db`) | description body | Core persists no severity/priority (create echoes `priority` but does not store it) |
| `closure_criteria`, `composes_with`, `forensic_anchor`, `assigned_to`, `created_by`, `created_at`, `last_modified`, `current_location` | NOT PUSHED (intentional) | Core's ticket row stores only title, description, type, status, assignee user id, estimation, story points. `assigned_to`/`created_by` are free text (e.g. "Operator", "AI") while Core's assignee/creator must be a Core user; `created_at`/`last_modified` are server-controlled and would make every registry touch a remote update; `closure_criteria` duplicates `acceptance` in the current data; `composes_with`, `forensic_anchor` and `current_location` have no field and would only enlarge the free-text description. Enforced by tests (`PUSHED_PAYLOAD_FIELDS` / `UNPUSHED_PAYLOAD_FIELDS` must cover every key of the real outbox) |

Owned fields = title, description, type, status. `fields_hash` (sha256 of those four) is stored as `base_hash` in the state file
after every successful write; it is the baseline for remote-edit detection.

## Matching remote tickets to registry items

1. the `ticket_id` recorded in the state file; 2. the marker line in the description; 3. the `[HRD-nnn]` title prefix.
A known ticket whose title lost its key is therefore a **conflict**, never a new ticket. Several tickets claiming one key (including a
forged `[HRD-nnn]` title next to the known ticket) are reported `duplicate` (conflict, nothing written). Tickets claiming a key that is not in the
registry are listed as `orphans`. A ticket recorded in the state file that is gone remotely (absent from `list` and `read` returns 404) is
`remote-deleted`: a conflict (exit 2), never silently re-created; `--overwrite-remote` re-creates it explicitly (`recreate`).

## Conflict policy

- The registry is the source of truth for all four owned fields.
- A remote edit is never silently overwritten: if the remote hash differs from both the desired hash and the last pushed `base_hash`
  (or no baseline exists), push reports `conflict`, writes nothing, exits 2. After review the operator re-runs with `--overwrite-remote` (`overwrite`).
- Registry-only change (remote still equals baseline): `update` with a version read by the probe below, one retry on 409 if remote content still
  equals the baseline; a content change during the race becomes `conflict`.
- Pull is read-only toward the registry and the tickets' content. It reports `in-sync`, `registry-ahead`, `remote-edit`, `conflict`,
  `diverged-no-baseline`, `missing-remote`, `remote-deleted`, `duplicate`, plus orphans. Remote status changes appear as proposals mapped back to registry
  vocabulary; `done`, `review`, `testing`, `closed` have no unambiguous registry status so the proposal is null. The registry is only changed by an operator
  (`scripts/workable_items.py`).
- Type changes in the registry cannot be pushed (`unpushable`). Every write is verified by a read-back (`ReadbackMismatch` aborts the run and is reported).

## Version probe and the fact that pull issues modify requests

`read`/`list` omit `version`. To read the optimistic-lock version the tool reads the ticket, then sends `modify` with the ticket's **current**
title/description/status and the impossible version 2147483647. A conforming Core rejects it with 409 before writing and reports the current version, so
nothing changes. If a server ignored `version` the request would be applied as a no-op write (identical content, version counter bumps) and the tool aborts with
`ServerError` ("locking not enforced"). `pull` therefore does issue `modify` requests for drifted items; `pull --no-probe` issues none (remote_version is then null).
Live evidence: 07b (no probe requests) and the sink check "no ticket titled probe".

## Resumability, atomic state, exit codes

State (`--state`) is saved atomically (0600 temp file in the same directory, fsync, rename) after every item, and right after a create **before** the follow-up status
modify, with the as-created baseline; a failure in between therefore resumes as `update`, not as a false conflict. If the state file is lost as well, a ticket equal to the
as-created form is reconciled the same way. A failure (HTTP 5xx, transport, read-back mismatch) stops the batch, is recorded as an `error` action, the partial report is written
and the exit code is 3. `--dry-run` never writes the state file. Exit codes: 0 clean, 2 conflict/drift (push conflict, `unpushable`, `duplicate`, `remote-deleted`; pull drift;
compare mismatch), 3 error or aborted run (including every item below).

**Single-instance lock.** Every invocation (push, pull, compare, dry-run) takes an exclusive, non-blocking `flock` on `<state>.lock` (a 0600 file beside the state file; POSIX only) before
reading any input. A second concurrent run exits **3** at once with `LockError: another helixtrack-sync instance holds the lock ...`, makes no request, and deliberately does **not**
overwrite `--report` (that file belongs to the running instance). The kernel drops the lock when the process ends or dies, so there is no stale-lock cleanup; the `.lock` file
itself is left in place and is safe to ignore or delete while no run is active.

**Input validation before login.** The outbox, the state file and (with `--db`) the registry history DB are validated before the first request. Truncated/empty/NUL/non-UTF-8 files,
a state that is not an object, `items` that is not an object, an item that is `null`/a string/a list, a bad `ticket_id`/`base_hash`, an unknown `schema`, an outbox without a non-empty
`records` list, a record without a `client_key` (`HRD-nnn`), without a `payload` object, with duplicate keys or without string `title`/`type`/`status`, and an unreadable history DB
all give `SyncError`, a message on stderr, an `aborted` report and exit 3, with zero requests made.

**File modes.** The state file, the report (including the abort report and the compare report), the temp files and the lock file are created 0600 regardless of umask (`run_live.sh` additionally runs under `umask 077`). A missing `--db` registry history DB is refused (exit 3) and never created.

**History DB and WAL.** `docs/workable_items.db` is WAL-mode and `immutable=1` ignores the `-wal`, so the tool picks the open per run: when a non-empty `<db>-wal` and its `-shm` both exist it opens `mode=ro` (honours un-checkpointed writer content; creates and modifies no file, as both side files already exist). Otherwise, or if that open fails, it falls back to `mode=ro&immutable=1`; if a non-empty `-wal` exists in that case the history may be **stale**, and a WARNING is printed to stderr and listed in the report's `warnings` array (push/pull/compare). With no `-wal` the main file is the whole truth and no warning is raised.

**Env-file parser limits.** The file is read as at most 64 KiB (`ENV_MAX_BYTES`; one byte more is refused unread, exit 3, no matter how large the file is). It is a minimal `KEY=VALUE` parser: `#` comment lines and blank lines are skipped; there is no BOM handling (a UTF-8 BOM breaks the first key), no `export ` prefix, no quote or escape processing (quotes would become part of the value), no variable expansion; the first occurrence of a key wins and variables already in the process environment take precedence. A symlink is followed and judged by the TARGET's mode and type (it must be a regular file with no group/other bits).

**Baseline ordering.** A ticket's state baseline is advanced to the desired content only after the read-back verification succeeded; a failed read-back leaves the previous (or as-created) baseline, so the next run treats the item as `update`, not as a conflict.

**Transport failures.** Anything the HTTP stack throws (refused/reset/timeout, `BadStatusLine`, `IncompleteRead`, a 200 response whose body is not JSON) becomes `TransportError`
(reported as an `error` action, partial report written, exit 3), never a traceback. Since fix round 1 this also holds for malformed server responses (a ticket without `id`, `data` of the wrong type, deeply nested or over-size bodies, non-text fields): every parsed server field is shape-validated (`ShapeError`), bodies are capped at 8 MiB, `--report` is replaced by an `InProgress`/aborted marker as soon as the run starts so a stale report never survives, and an unexpected exception is reported as an aborted run (exit 3). All server- or tree-derived text printed to stderr or written to the report is escaped by `safe_text` (every Unicode C* character, ESC/OSC/CSI/C1, bidi, NUL, CR, line separators) and capped. A registry record whose status or type has no Core mapping is reported as `unmapped` (exit 2) and the batch continues.

## Transport safety

Credentials are sent only over https, or http to a loopback host; the scheme is parsed and compared case-insensitively (`HTTP://10.x` is refused before any connection).
Redirects are refused (the opener's redirect handler returns nothing; any 3xx is `RedirectRefused`), so `Authorization` is never forwarded to another host or scheme.
The base URL must carry no userinfo (`user@`), backslash, `#`, `?`, whitespace/control or non-ASCII character and a valid port: `urlparse` and the HTTP client disagree on such inputs
(`http://evil@127.0.0.1`, `http://evil.com\@127.0.0.1`), so they are refused outright rather than parsed. Loopback means exactly `localhost` or a loopback IP (`evillocalhost` is refused).
The env file is opened first (non-blocking, so a FIFO cannot hang the tool) and the opened descriptor is checked before anything is read: it must be a regular file (a FIFO, directory or missing path is `SyncError`, exit 3) and the process environment is only populated after the whole file parsed. Env files must be regular files with no group/other access bit at all (0600/0400/0700 pass; 0640, 0604, 0660, ... are refused). Requests/responses are recorded with `jwt`, `password`, `token`, `Authorization` redacted.

## Core quirks found (all reproduced live)

1. `read`/`list` do not return `version`. 2. `modify` ignores `type` and silently drops an unknown `status` when other fields are present. 3. `modified` has 1-second resolution.
4. Shipped `Application/Database/Definition.sqlite` lacks `ticket.version`, `project.version`, `ticket_history`; with it `create project` fails and is mislabelled HTTP 409 "already exists". The first-start
   migrator over the root `Database/DDL` fails on its own MySQL-style `INDEX (...)` DDL. The local instance used the shipped DB plus `core-local-schema-patch.sql`. UNCONFIRMED whether upstream Docker images carry a fixed DB.
5. The server logs raw `/do` bodies including passwords. 6. The default JWT secret is a hard-coded public string when none is configured.

## Usage

```
HT_BASE_URL=... HT_USER=... HT_PASSWORD=...   # or --env-file (mode 0600)
python3 tools/helixtrack-sync/helixtrack_sync.py push|pull|compare --state S.json [--dry-run] [--overwrite-remote] [--no-probe] \
  [--outbox docs/tracking/helixtrack-outbox.json] [--db docs/workable_items.db] [--record traffic.jsonl] [--report r.json]
```
Tests: `cd tools/helixtrack-sync && python3 -I -m unittest discover -s . -p 'test_*.py'` (126 tests at fix round 1; run with `PYTHONPATH=.` because `-I` hides the test module). Mutation checks: `evidence/campaign-2026-11-23/helixtrack/run_mutants.py` (rounds 1-3 mutants) and `evidence/campaign-2026-11-23/helixtrack/fix-round-1/run_mutants_r1.py` (fix-round-1 mutants); counts and results are in the logs next to them, not restated here.

## Recommended HRD-029 wording

"Push (create/update, idempotent by state ticket_id, marker line and `[HRD-nnn]` key), pull with remote-edit/retitle/delete detection by content hash and Core optimistic-lock version,
conflict reporting, resumable batches and redirect/plain-http refusal are implemented and proven against a self-hosted Helix-Track/Core (75/75 items field-by-field, idempotent re-push,
remote edit, retitle and delete by a second local user detected and not overwritten, stale-version write rejected, 503 mid-batch resumed with no duplicates). Registry write-back is intentionally
report-only. NOT proven: a production HelixTrack instance, external auth/permission services, multi-user permission scoping, attachments/comments, scale beyond 75 items, TLS to a real host.
Status stays In progress until a production-instance run is recorded."
