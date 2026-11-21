# Yomikiru — per-repository research report

- Repo: https://github.com/mienaiyami/yomikiru · Revision: `3087618628734ef7e986fcd26b331a80dd4f3b57` — detached checkout at the exact analysis-pinned revision; row(s) `yomikiru`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-22 20:16:54 +0530 — "fix(keybindings): run side-button shortcuts once per click". Full non-shallow clone, `git fsck` clean.
- Ledger recomputed at pinned revision: 508 blobs, 4,424,019 bytes — matches preserved analysis (tracked_files `508`). Top extensions (count): ts 272, tsx 133, md 37, json 17, scss 16, yml 9.

**Architecture / data flow.** Electron comic reader (272 ts/133 tsx): local library, drizzle ORM for settings/history, side-button/mouse-key ergonomics for trackpad-free reading.
**Notable techniques.** Input-mapping layer for media remotes — relevant to Helix's e-ink key event model; compact Electron patterns.
**License.** MIT.
**Porting decision — LEARN + selective port (attributed: Helix research agent, checkout `3087618628734ef7e986fcd26b331a80dd4f3b57`).** MIT-compatible; borrow the keybinding schema, not the shell.