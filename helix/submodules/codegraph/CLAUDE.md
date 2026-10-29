# CLAUDE.md

Claude Code project guidance for this repository.

Primary instructions live in the canonical agent guide — import it:

@AGENTS.md

## Claude-only notes

- Prefer the repo-root `AGENTS.md` as the source of truth for build/test/architecture/house rules. Edit that file (not this wrapper) when guidance changes.
- Claude Code can `@`-import nested guides too (e.g. `@docs/AGENTS.md`) when working on design/eval docs; Codex loads nested `AGENTS.md` automatically when the session cwd is under that directory.
- Do not reintroduce a duplicated `## CodeGraph` MCP tool-guidance block here — `src/mcp/server-instructions.ts` is the single source of truth (issue #529); the installer strips legacy marker blocks on upgrade.

## Mandatory credential and secret handling

> "Credentials or any secret and sensitive data MUST NOT leak! This MUST BE added as the mandatory constraint into all Submodules and main project's Constitution, CLAUDE.MD and AGENTS.MD."

Credentials, tokens, passwords, private keys, session material, connection-string passwords, and other sensitive values MUST NOT be committed, printed, logged, or included in test evidence or archives. Use environment-specific secret injection and redacted placeholders; sanitize errors and diagnostics at trust boundaries.
