# Helix Server — HTTP API Reference

## Conventions

- JSON request and response bodies unless an OPDS or file media type is specified.
- Errors have the form `{"error":"public message","request_id":"..."}`. Internal errors and credentials are not returned.
- Management routes use `Authorization: Bearer <HELIX_ADMIN_TOKEN>`. If the token is empty, the current middleware permits the request; non-loopback startup rejects an empty token unless an explicit insecure override is set.
- Catalog read, content, cover, and OPDS routes use optional OPDS Basic authentication. When `HELIX_OPDS_USERNAME` and `HELIX_OPDS_PASSWORD` are configured, all of those routes require the same credentials.
- Pagination uses `offset` and `limit`; the default limit is 20 and the maximum is 100. Negative offsets are normalized to zero.
- Source and AI request bodies must use the documented content types. Credentials are accepted only as configuration fields and are never returned in source responses.

## Health and capabilities

| Method | Path | Auth | Result |
| --- | --- | --- | --- |
| GET | `/healthz` | None | Process health and current UTC time |
| GET | `/readyz` | None | `503` when no catalog is configured; otherwise ready |
| GET | `/v1/formats` | None | Executable format descriptors and capability flags |
| GET | `/v1/protocols` | None | Sorted registered connector names |

A registered format or protocol is not a claim that every external engine, provider, renderer, or live service has been verified.

## Catalog and files

| Method | Path | Auth | Result |
| --- | --- | --- | --- |
| GET | `/v1/works?q=&author=&state=&offset=&limit=` | OPDS Basic when configured | Deterministic work page |
| GET | `/v1/works/{id}` | OPDS Basic when configured | Work or `404` |
| GET | `/v1/editions/{id}` | OPDS Basic when configured | Edition or `404` |
| GET, HEAD | `/v1/covers/{id}` | OPDS Basic when configured | Validated cached cover or metadata without a body |
| GET, HEAD | `/v1/files/{id}/content` | OPDS Basic when configured | Original bytes with ETag and single-range support |
| DELETE | `/v1/works/{id}` | Admin | `204` or `404` |

Content responses use safe attachment disposition. Local GET responses open the file first and read size/modification time from that same descriptor, then stream or seek through it, preventing path replacement between validation and reading; HEAD uses a fresh metadata stat. Post-scan growth or shrinkage therefore cannot make Helix emit a stale catalogued length. While live size and modification time still match the scan record (compared at PostgreSQL microsecond precision), responses include `Accept-Ranges: bytes` and a SHA-256-derived strong ETag. Changed files instead receive a full `200` without the stale ETag or ranges, preventing an old `If-Range` validator from splicing different bytes. Connectors without live revalidation use a full `200` without `Content-Length`, ETag, or ranges. Multi-range requests and mismatched `If-Range` validators also receive the complete representation. Invalid or unsatisfiable single-range requests against an unchanged file return `416` with `Content-Range: bytes */{live-size}`. HEAD returns the corresponding headers without opening the file. Attachments and already-compressed media remain identity-encoded. A disconnected source returns `409`; a file deleted at the connected source returns `404`.

Identification states are `pending`, `identified`, `partial`, `needs_review`, and `failed`.

## Sources and scans

| Method | Path | Auth | Result |
| --- | --- | --- | --- |
| POST | `/v1/sources` | Admin | Create a runtime connector; `201` with source ID and capabilities |
| DELETE | `/v1/sources/{id}` | Admin | Close and remove runtime connector; `204` |
| POST | `/v1/scans` | Admin | Queue an in-place scan; `202` with job |
| GET | `/v1/scans` | Admin | List process-local jobs |
| GET | `/v1/scans/{id}` | Admin | Read job state/report |
| DELETE | `/v1/scans/{id}` | Admin | Request cancellation; `202` |

Example local source request:

```json
{
  "id": "library-main",
  "protocol": "local",
  "root": "/library"
}
```

The optional `id` is a stable administrative identifier: 1–128 ASCII letters, digits, internal hyphens/underscores/dots, beginning with a letter or digit. If omitted, the server generates a random ID. Reuse the same explicit ID when re-registering a source after restart so existing source/path file IDs reattach and full-scan reconciliation remains effective; registering under a different ID creates a different source identity and can duplicate catalog files. A duplicate active ID returns `409`, and an invalid ID returns `400`.

Remote source requests may additionally contain `endpoint`, `username`, `password`, `headers`, `rclone_path`, `rclone_config`, or `allow_private_network`. Do not place credentials in endpoint URLs. Secret headers are not forwarded to cross-origin acquisition URLs or redirects. For OPDS sources, cross-origin loopback/private/CGNAT acquisition destinations are rejected by default and validated at dial time, including redirect hops. Set `allow_private_network: true` only for a trusted feed that must reference another LAN/loopback origin. Link-local, unspecified, multicast, and broadcast destinations remain blocked even with that opt-in; the explicitly configured source origin is not restricted.

Example scan request:

```json
{
  "source_id": "source-returned-by-registration",
  "prefix": "optional/subdirectory"
}
```

Deleting a source returns `409` while any queued, cancel-requested, or running scan references it; scan admission and source deletion are serialized so a connector cannot be closed between lookup and job registration. Job states are `queued`, `cancel_requested`, `running`, `succeeded`, `failed`, and `canceled`. At most two scans run concurrently and 64 unfinished scans may be pending; excess requests receive `429`. Up to 256 terminal/recent jobs are retained, with terminal jobs older than 24 hours pruned first. Panics become failed jobs, cancellation is observed while queued or running, and shutdown rejects new work (`503`), cancels workers, and drains them before closing connectors. A successful unprefixed scan of a recursive connector reconciles the complete listing: catalog files no longer present at that source are removed, and editions/works left empty are deleted. The report’s `removed` field counts these stale file references. Same-source full scans are serialized so an older listing cannot prune newer results. Prefix scans and non-recursive connectors never prune; a file still present in the listing is retained even if its ingest fails. OPDS listings that exceed pagination or entry limits fail instead of being treated as complete snapshots. Administrator-facing job errors and per-file report locations strip URL userinfo, query strings, and fragments, including presigned acquisition URLs; malformed HTTP URLs fail closed to redaction. OPDS catalog file paths likewise exclude acquisition userinfo, queries, and origin fragments. When distinct raw acquisitions share a credential-free public path, Helix assigns deterministic `helix-x-{sha256}` synthetic fragments instead of collapsing them. New collision groups are keyed independently of feed order, and previous-generation raw→key assignments are reused when colliding siblings arrive or disappear. These fragments are never sent to the origin and are removed from download filenames. Signed acquisition URLs exist only in the current connector’s process-local memory. A complete listing atomically replaces and prunes that mapping; failed listings do not partially replace it, and closing the connector clears it. Source registrations, connector instances, signed acquisition mappings, and jobs are currently process-local: after restart, re-register with the same explicit source ID and rescan signed remote sources before requesting their content.

## OPDS

All OPDS routes use OPDS Basic authentication when configured.

| Method | Path | Media type / result |
| --- | --- | --- |
| GET | `/opds/1.2` | OPDS 1.2 navigation Atom feed |
| GET | `/opds/1.2/publications?q=&offset=&limit=` | OPDS 1.2 acquisition feed |
| GET | `/opds/1.2/editions/{id}` | Standalone Atom edition entry |
| GET | `/opds/1.2/opensearch.xml` | OpenSearch description |
| GET | `/opds/2.0` | OPDS 2.0 navigation feed |
| GET | `/opds/2.0/publications?q=&offset=&limit=` | OPDS 2.0 publication feed |
| GET | `/opds/2.0/editions/{id}` | Standalone RWPM publication |

Feeds include acquisition and cover links where available. OPDS-PSE page streaming is not implemented.

## Internal reading progress

These routes require admin Bearer authentication and Redis. Without Redis they return `503`.

| Method | Path | Result |
| --- | --- | --- |
| GET | `/v1/sync/progress/{user}/{book}` | Structured progress or `404` |
| PUT | `/v1/sync/progress/{user}/{book}` | Merge and return structured progress |
| DELETE | `/v1/sync/progress/{user}/{book}` | Delete progress; `204` |

The PUT body is a core `ProgressRecord`; the path supplies the book ID. A record requires a device ID and vector clock. Position components must be non-negative. Redis optimistic transactions merge inherited clocks and select the furthest valid position deterministically.

## KOReader sync

These routes are available only when Redis is configured. They use KOReader’s `x-auth-user` and `x-auth-key` headers, not the admin Bearer token. Use TLS in transit.

| Method | Path | Auth | Result |
| --- | --- | --- | --- |
| POST | `/users/create` | Public only while registration is enabled | `201` and username; `403` when disabled; `409` on duplicate |
| GET | `/users/auth` | KOSync headers | `{"authorized":"OK"}` or `401` |
| PUT | `/syncs/progress` | KOSync headers | Server document/timestamp response |
| GET | `/syncs/progress/{document}` | KOSync headers | Caller’s document progress or `404` |

Registration body:

```json
{
  "username": "reader",
  "password": "password-digest-supplied-by-koreader"
}
```

Usernames contain 3–32 ASCII letters, digits, underscores, or hyphens. Password digests are 8–72 bytes. Only bcrypt verifiers are persisted.

Progress body:

```json
{
  "document": "document-hash",
  "progress": "opaque-koreader-position",
  "percentage": 0.47,
  "device": "device-name",
  "device_id": "optional-device-id",
  "timestamp": 0
}
```

`document`, `progress`, and `device` are required. String fields are limited to 255 bytes, percentage must be between 0 and 1, and the request body is limited to 16 KiB. The server ignores the supplied timestamp and assigns its own. Registration and authentication bodies have smaller limits. Invalid credentials receive the same public error for unknown users and incorrect keys.

## AI memory

These routes require admin Bearer authentication, `HELIX_AI_MEMORY=true`, PostgreSQL, and pgvector. Without the configured store they return `503`.

| Method | Path | Result |
| --- | --- | --- |
| PUT | `/v1/ai/memories` | Store bounded content/metadata and a fixed-dimension finite vector |
| POST | `/v1/ai/memories/search` | Scoped cosine-distance search with optional limit |
| DELETE | `/v1/ai/memories/{scope}/{id}` | Delete memory; `204` |

The configured embedding dimension defaults to 1536 and must be between 1 and 2000. AI records and proposals do not override deterministic metadata evidence.
