# Helix Reader — Current User Guide

## Status and scope

Helix is under active development. This guide distinguishes the **interactive product prototype**, the **Go server**, and features that are still integration targets. It does not describe planned functionality as working software.

- The responsive web prototype is interactive but uses sample in-memory data. It is not connected to the Go server. Reloading or clearing browser state resets its changes.
- Native desktop and mobile applications are not delivered.
- The server is not yet production-hardened. Local live PostgreSQL, Redis, pgvector, daemon-restart, and transport checks have passed, but external-device, third-party-reader, public-deployment, upgrade/rollback, and container-runtime qualification remain explicit open tasks.
- No renderer, LCP license fulfillment, or external conversion engine is currently connected.

## Interactive web prototype

Open the saved website version to inspect the shared visual language and interaction model. The prototype includes:

- catalog search and filters;
- book and edition details;
- chapter navigation, typography controls, themes, and sample passage notes;
- per-field metadata proposal acceptance or rejection;
- sample source validation and scan-state transitions;
- a deterministic furthest-position conflict demonstration.

The interface is labeled **“Interactive product prototype”**, **“No backend connected”**, and **“Sample data only.”** These disclosures are intentional. The prototype must not be treated as a persistent library.

## Server quickstart

### Requirements

- Go 1.27.3 for a source build;
- PostgreSQL for a durable catalog;
- Redis for durable reading progress and KOReader sync;
- optionally pgvector for AI memory;
- optionally `rclone` and a separately configured rclone remote for supported non-native connectors.

### Configuration

Start from `deploy/.env.example`. Keep the populated `.env` file out of source control and out of logs, screenshots, test reports, and archives.

Important variables:

| Variable | Purpose | Default / constraint |
| --- | --- | --- |
| `HELIX_ADDRESS` | TCP and optional QUIC listen address | `127.0.0.1:8080` |
| `HELIX_ADMIN_TOKEN` | Bearer credential for management APIs | Required for non-loopback listeners unless an explicit insecure override is used |
| `HELIX_POSTGRES_DSN` | Durable catalog and optional AI memory | Without it, the catalog is volatile memory |
| `HELIX_REDIS_URL` | Progress and KOReader sync persistence | Without it, those APIs report that they are not configured |
| `HELIX_OPDS_USERNAME`, `HELIX_OPDS_PASSWORD` | Optional OPDS Basic authentication | Must be configured together |
| `HELIX_TLS_CERT`, `HELIX_TLS_KEY` | Native TLS for HTTP/2 and optional HTTP/3 | Must be configured together |
| `HELIX_HTTP3` | HTTP/3 when native TLS is configured | `true` |
| `HELIX_EXTERNAL_URL` | Absolute public base URL used in generated links | No trailing slash |
| `HELIX_KOSYNC_REGISTRATION` | Allows KOReader account creation | `false`; disable again after provisioning accounts |
| `HELIX_AI_MEMORY` | Enables pgvector-backed AI memory | Requires PostgreSQL |

The Compose defaults publish only on loopback. They do not by themselves provide public TLS. Use a correctly configured TLS reverse proxy or extend the deployment with certificate mounts and the corresponding Helix variables.

### Build

```bash
cd server
go test ./... -count=1
go build ./cmd/helix-server
```

See the Developer Guide for the complete validation matrix. A successful local build is not evidence of production deployment or external-device interoperability. An isolated backup/restore rehearsal has verified PostgreSQL catalog/vector data and Redis progress through a fresh server; version-upgrade and rollback rehearsal remains open.

## Catalog and sources

The catalog preserves **work → edition → file** relationships. ISBNs and other edition identifiers remain edition-specific; multiple files can belong to one edition. Metadata evidence and review states are retained.

Source connectors are read-only. Indexing does not copy books into a Calibre-style managed directory or reorganize originals. Native connectors cover local directories, mounted SMB/NFS paths, HTTP(S), WebDAV, and OPDS acquisition feeds. Additional protocols use configured rclone remotes. A successful full scan of a recursive source removes stale catalog references to files that have disappeared; prefix scans and non-recursive feeds do not prune. Deleting catalog references never deletes source bytes. OPDS feeds may reference another origin only without configured credential forwarding; private/loopback cross-origin destinations require the explicit `allow_private_network` source option, and cloud-metadata/link-local destinations remain blocked.

Current management limitation: source registrations, connector instances, and scan-job state are process-local. They must be recreated after a restart; durable source configuration and scheduling are unfinished. Set an explicit stable source `id` when registering and reuse that exact ID after restart so existing file identities reattach and rescans do not create a second source’s worth of file entries. Signed OPDS feeds must also be rescanned to rebuild their process-local acquisition map.

## Reading progress and KOReader sync

When Redis is configured, Helix exposes its internal progress API and KOReader-compatible endpoints:

- `POST /users/create` — registration only when explicitly enabled;
- `GET /users/auth` — validates `x-auth-user` and `x-auth-key`;
- `PUT /syncs/progress` — stores a document position;
- `GET /syncs/progress/{document}` — retrieves the caller’s position.

KOReader sends a password digest as `x-auth-key`. Helix treats that digest as a password and stores only a bcrypt verifier. Network transport must use TLS. Progress timestamps are assigned by the server, request bodies are bounded, and each account has an isolated progress namespace.

Actual KOReader firmware interoperability has not yet been live-tested; unit, HTTP, and Redis-backed tests do not substitute for that device check.

## Formats and conversion

Use `GET /v1/formats` for the executable capability registry. Detection and byte streaming do not imply metadata parsing, rendering, conversion, or DRM fulfillment.

- Native metadata/parser paths exist for EPUB, MOBI, FB2, TXT, and CBZ at the levels indicated by the registry.
- Pure-Go conversion is limited to the implemented EPUB/FB2/MOBI/TXT → FB2/TXT/XHTML paths in the conversion matrix.
- PDF, AZW/AZW3, comic archives, Readium packages, DAISY, and other detected formats may be streamable without having a renderer or converter.
- `.lcpl` is detected as a license document. Helix does not fulfill LCP licenses or decrypt protected publications.

## AI and metadata review

Deterministic identifiers and metadata evidence take precedence over AI proposals. AI memory accepts bounded content/metadata and fixed-dimension finite vectors when pgvector is configured. A strategy or provider name alone never establishes a working integration. Production LLM provider configuration, native content renderers, and end-to-end content-AI qualification remain incomplete.
