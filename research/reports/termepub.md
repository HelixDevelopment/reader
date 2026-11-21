# Termepub — per-repository research report

- Repo: https://github.com/mattllvw/termepub · Revision: `70a1886f691bcfd87d05829a2839638b4ee24969` — detached checkout at the exact analysis-pinned revision; row(s) `termepub`, `current_checkout_matches_analysis=true`.
- Commit: 2021-11-22 20:35:43 +0100 — "fix release workflow: binary_name". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 8 blobs, 5,299,435 bytes — matches preserved analysis (tracked_files `8`). Top extensions (count): yml 2, (none) 1, md 1, gif 1, mod 1, sum 1.

**Architecture / data flow.** Single-file Go terminal EPUB reader (`main.go`, 8 blobs total): zip→xhtml parse, golang.org/x/term text pagination, keyboard navigation.
**Notable techniques.** Entire EPUB-to-text pipeline in one file; good sanity check for Helix's Go text-extraction boundaries.
**License.** MIT.
**Porting decision — PORT-WORTHY (attributed: Helix research agent, checkout `70a1886f691bcfd87d05829a2839638b4ee24969`).** MIT Go code directly inspectable for Helix's plain-text pagination helpers; any derived function must record this revision and MIT notice in the source header.