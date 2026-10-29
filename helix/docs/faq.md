# Helix Reader — FAQ

## Is Helix production-ready?

No production-ready claim is currently made. Core/server tests, race and fuzz runs, responsive browser checks, mutation gates, local live database/transport checks, and daemon-restart persistence checks provide useful evidence. Native applications, real-device and third-party-reader compatibility, external renderers, DRM fulfillment, container deployment, upgrade/rollback, and a complete production security/deployment qualification remain open.

## Does the web prototype save my library?

No. The current web prototype uses sample in-memory data and is explicitly labeled “No backend connected.” Changes are not a durable library and do not synchronize between devices.

## Do I need an account?

The catalog and OPDS endpoints can run without OPDS credentials on a loopback evaluation deployment. Management APIs use the admin Bearer token when configured; a token is required by default for non-loopback listeners. KOReader sync has separate accounts, and public registration is disabled by default. A complete multi-user authorization and per-library permission system is not implemented.

## Can Helix index books without copying or reorganizing them?

Yes, that is the purpose of the read-only connector model. Local files, mounted SMB/NFS directories, HTTP(S), WebDAV, OPDS feeds, and configured rclone remotes are indexed from their existing locations. Source registrations and connector instances are not yet durable across server restarts.

## Which cloud and network protocols are available?

Local, mounted SMB/NFS, HTTP(S), WebDAV, and OPDS connectors are native. FTP/FTPS, SFTP, SMB, S3, Google Drive, Dropbox, OneDrive, Box, pCloud, MEGA, and Nextcloud use rclone. A registered protocol name means a connector factory exists; it does not prove that every provider or credential type has passed live interoperability testing.

## Does Helix support every listed reading format end to end?

No. The registry separates detection, metadata, byte streaming, rendering, conversion, content AI, and DRM fulfillment. Consult `GET /v1/formats` rather than inferring support from a file extension. Native rendering and LCP fulfillment are not currently connected.

## Which conversions actually work?

Only conversion paths marked `direct` in the executable matrix are implemented by the current pure-Go converter. These cover specific EPUB, FB2, MOBI, and TXT inputs producing FB2, TXT, or XHTML. PDF, AZW/AZW3, CBZ, DOCX, and other external conversions are not silently claimed.

## Can AI overwrite a correct ISBN or title?

AI output must remain a proposal with provenance and review state; it must not override stronger deterministic evidence. The review model is designed around that rule. End-to-end provider integrations and their deployment-specific behavior still require separate qualification.

## Does KOReader sync work?

The server now implements the KOReader account and progress HTTP contract with Redis persistence, bcrypt verifiers, bounded requests, and per-user isolation. Automated tests pass. Actual KOReader device/firmware interoperability remains unverified until a live device test is recorded.

## Are reading position conflicts resolved?

The internal progress store merges inherited vector clocks and selects the furthest position according to the core conflict rules. The web prototype demonstrates that behavior with sample data. KOReader’s own protocol uses server-timestamped document progress and is a separate compatibility surface.

## Does Helix send source credentials to other servers?

Configured secret headers must not be forwarded to cross-origin acquisition URLs or redirects. HTTP redirect credential stripping, OPDS same-origin pagination, and dial-time acquisition address checks are covered by implementation tests. Cross-origin private/loopback acquisition requires an explicit trusted-source opt-in; link-local/metadata destinations remain blocked. Credentials must still be kept out of endpoint URLs, logs, screenshots, evidence files, and archives.

## Are OPDS 1.2, OPDS 2.0, and OpenSearch implemented?

Yes, the server has acquisition/navigation feeds, standalone edition documents, acquisition links, pagination, covers, and an OpenSearch description. OPDS-PSE page streaming and Kobo-specific synchronization are not implemented.

## Are annotations, watch folders, smart shelves, and native readers available?

Not yet. These are identified product gaps, not hidden completed features. The current prototype’s passage notes are sample UI state only and are not a durable annotation service.

## Where are architecture, API, testing, and research details documented?

Start with `docs/developer_guide.md`, `docs/architecture.md`, `docs/api.md`, and `docs/testing.md`. Revision-pinned upstream evidence is under `research/`. Evidence under `evidence/` records only checks that actually ran; missing or failed checks must not be interpreted as passes.
