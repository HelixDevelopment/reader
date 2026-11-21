# Tidepage — per-repository research report

- Repo: https://github.com/miipa/tidepage · Revision: `7e9caeb0ce4c7eb9922c5abf59cade2d86617004` — detached checkout at the exact analysis-pinned revision; row `tidepage`, `current_checkout_matches_analysis=true`.
- Commit: 2026-10-27 19:14:58 +0800 — "chore(release): v0.1.33". Full non-shallow clone, `git fsck` clean.
- Ledger at pinned revision: 233 blobs, 6,062,692 bytes — matches analysis tracked_files=233. Top extensions (count): ts 141, svelte 33, rs 26, png 7, jpg 7, json 5.

**Architecture / data flow.** SvelteKit + Tauri (26 rs) reading app: paginated long-form reader, local vault storage, release automation (v0.1.33 cadence).
**Notable techniques.** Tauri-side file vault + Svelte streaming pagination — a lightweight desktop analogue of Helix's reader loop; bilingual docs.
**License.** MIT (2026 miipa).
**Porting decision — LEARN + selective port (attributed: Helix research agent, checkout `7e9caeb0ce4c7eb9922c5abf59cade2d86617004`).** MIT; pagination state machine in TS is directly referenceable for Helix's web reader.