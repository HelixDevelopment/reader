# token_optimizer — CLAUDE.md

## INHERITED FROM Helix Constitution

This module is a submodule of the Helix Constitution. All rules in 
`constitution/CLAUDE.md` and the `constitution/Constitution.md` it 
references apply unconditionally.

Canonical reference: https://github.com/HelixDevelopment/HelixConstitution

## Mandatory credential and secret handling

> "Credentials or any secret and sensitive data MUST NOT leak! This MUST BE added as the mandatory constraint into all Submodules and main project's Constitution, CLAUDE.MD and AGENTS.MD."

Credentials, tokens, passwords, private keys, session material, connection-string passwords, and other sensitive values MUST NOT be committed, printed, logged, or included in test evidence or archives. Use environment-specific secret injection and redacted placeholders; sanitize errors and diagnostics at trust boundaries.
