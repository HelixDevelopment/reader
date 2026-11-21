# JellyBook — per-repository research report

- Repo: https://github.com/jellybookorg/jellybook · Revision: `505684a96fd2529cf17697a97824064b74671297` — detached checkout at the exact analysis-pinned revision; row `jellybook`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-25 22:33:56 -0400 — "Merge pull request #229 from JellyBookOrg/Kara-Zor-EL/size". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 417 blobs + 2 submodule entries, 76,219,059 bytes — matches analysis tracked_files=419. Top extensions (count): png 72, dart 49, xml 43, arb 39, h 22, plist 15.

**Architecture / data flow.** Flutter (49 dart, 2 submodules) client for Jellyfin book libraries: iOS/Android/desktop shells, arb i18n (39), Makefile dev workflow, docker-compose dev server.
**Notable techniques.** Treating an existing media server as the source of truth while adding a book-first client — same interop posture Helix takes toward Calibre/Kavita; fastlane mobile release path.
**License.** MIT (LICENSE.md, Kara 2022).
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `505684a96fd2529cf17697a97824064b74671297`).** MIT; value is the client-side requirements list for servers exposing book metadata, not code.