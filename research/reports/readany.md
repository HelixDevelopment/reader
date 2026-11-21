# ReadAny — per-repository research report

- Repo: https://github.com/codedogqby/readany · Revision: `fce44689e1377df1ae99a35e8e71d5e8b3a639d1` — detached checkout at the exact analysis-pinned revision; row `readany`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-07 22:50:19 +0800 — "Merge pull request #758 from k6G52m4Dz75W/fix/expo-cli-rel". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 1414 blobs, 33,471,848 bytes — matches analysis tracked_files=1414. Top extensions (count): ts 481, tsx 226, bcmap 168, json 113, md 94, png 86.

**Architecture / data flow.** Expo/React Native reader (481 ts/226 tsx, 168 bcmap = pdf.js assets): multi-format rendering (epub/pdf/mobi via pdf.js + js engines), translation integrations, patches/ dir for dependency surgery, pnpm monorepo `packages/`.
**Notable techniques.** EPUB_CONVERSION_PLAN.md documents format-normalization strategy as a plan-of-record; patch-directory discipline for pinned upstream fixes.
**License.** GPL-3.0.
**Porting decision — LEARN ONLY (attributed: Helix research agent, checkout `fce44689e1377df1ae99a35e8e71d5e8b3a639d1`).** GPL bars reuse; conversion-plan methodology and RN/pdf.js packaging notes are the harvest.