# Provenance of the HRD-026 interop captures (frozen files in this directory)

Rule: only facts readable from existing files are stated; everything else is UNCONFIRMED. The captured
outputs below are frozen evidence and are not edited. Class: NOT live client-to-server runtime evidence (see Limits).

## What each file is

| File | What it is | Source of the claim |
|---|---|---|
| `kosync_koreader_client.lua` | probe driving KOReader's `KOSyncClient.lua` + Spore + `api.json` over HTTP against a Helix server; `ui/uimanager`, `logger`, `socketutil` (and `utf8proc` glue) are stubbed | header comment of the file |
| `kosync_koreader_real_client.txt` | output of that probe before a server fix | file content (3+ `RESULT` lines) |
| `kosync_koreader_real_client_after_fix.txt` | output after a fix; no fix name or commit is recorded in the file | file content; fix identity UNCONFIRMED |
| `opds_koreader_parser.lua` | probe feeding saved OPDS 1.2 bytes to KOReader's `plugins/opds.koplugin/opdsparser.lua` | header comment of the file |
| `opds_publications.xml` | the saved feed the parser ran over (5 `<entry>`) | file content |
| `opds_koreader_parser_output.txt` | parser output: `entries= 5` | file content; matches the XML's entry count |
| `pse_page_fetch.txt` | 3 lines: page 1 -> 200 image/png 16, page 3 and 4 -> 404 JSON 84; no URL, no body hash | file content |

## Recorded commands and ports (from the files and `tools/interop/*.sh`)

- KOReader probes run with KOReader's bundled `luajit` from `/usr/lib/koreader`: `./luajit <probe>.lua http://127.0.0.1:PORT [user] [epub-path]` (header comment of `kosync_koreader_client.lua`; `opds_koreader_parser.lua` takes `<feed.xml>`).
- Server start/stop: `tools/interop/hrd026_server.sh` (loopback `127.0.0.1:38090`, Redis container on `127.0.0.1:36379`, both from the script constants). That the captures were taken with this exact script is UNCONFIRMED; the script was written alongside them.
- Provider scan script: `tools/interop/hrd026_providers.sh` (webdav `127.0.0.1:38101`, sftp `127.0.0.1:38102`); its outputs are in `hrd026/` (open PR #10), not in this directory.

## Identity facts that ARE recorded elsewhere

- KOReader version: the PR #10 file `hrd026/kosync_scripted_pull_second_device.txt` line 1 reads `KOReader v2026.07` with timestamp `2026-11-23T21:09:52Z`. That is a later run than the three files cited by the ledger; for the three cited files the KOReader version is UNCONFIRMED (the probe headers say "installed koreader deb", no version).
- Capture times (UTC) of the three cited outputs, read from the PR #9 snapshot provenance table (`ledger-snapshots/PROVENANCE.tsv`, `captured_at`): `kosync_koreader_real_client_after_fix.txt` 2026-11-23T19:27:00Z, `opds_koreader_parser_output.txt` 19:23:41Z, `pse_page_fetch.txt` 19:23:24Z. Source: that table, not these files.
- Fixture: the sample EPUB partialMD5 `af00b1716a8ea20bc7f1108c8358c4a3` appears in the kosync output; the EPUB's sha256 is `9c516119872123fe61ca8311b525b980a08c52fcb1710777fc10f87c33caf3c7` per PR #10 `hrd026/opds_download_pse.txt`.

## UNCONFIRMED (not recorded anywhere I could read)

helix-server commit and binary sha256 at capture; the server environment; KOReader package version for the three cited outputs; the exact command lines and working directory of each capture; the identity of the "fix" in `after_fix`; Redis image digest (the script pins the tag `redis:7.4-alpine` only); whether `pse_page_fetch.txt` came from a live endpoint (no URL recorded).

## Limits (read before citing)

- The probes ran headless with KOReader UI modules stubbed. This is KOReader's real client and parser Lua code exercised against Helix responses, NOT a KOReader application session and NOT an on-device run.
- The OPDS parse ran over a saved XML file, not a live fetch.
- None of the three cited files carries a target identity (server build, endpoint, command, timestamp), so they are not proven live runtime evidence in the sense of constitution 11.4.289(D); they support only: KOReader's sync-client and OPDS-parser code accept Helix responses of an unidentified build.
- `kosync_koreader_real_client_after_fix.txt` line 4 contains an operator home path (`/home/...`). The file is frozen and is not edited; the disclosure is here. A future capture should print a relative or hashed path.
