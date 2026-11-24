# Helix Reader desktop (Linux)

Electron 45.4.1 shell around `clients/web`. Windows and macOS stay BLOCKED (see `clients/supported-platforms.json`).

- Page origin is the privileged `app://helix` scheme (no TCP port is opened). `contextIsolation` on, `nodeIntegration` off, `sandbox` on, no `webview`, no remote module, all permission requests denied, navigation and window.open locked to the app origin, every session request outside `app:`/`blob:`/`data:` cancelled.
- Strict CSP on every app response (`default-src 'none'`, `script-src 'self'`, `connect-src 'self'`, no frames/objects/forms).
- The Helix server sends no CORS headers, so the main process proxies `GET/HEAD` on `/v1/*`, `/healthz`, `/readyz` to the configured server (`lib/proxy.js`). Cookies are never forwarded, `Set-Cookie`/`WWW-Authenticate` are stripped, redirects are not followed, dot/encoded-slash paths are refused. Leave the web client's "Server address" empty.
- Connection (server URL, username, password) is edited via menu Helix Reader > Connection. The password is encrypted with Electron `safeStorage` (OS keystore); if only the `basic_text` backend or none exists it is kept in memory for the session and never written. It is write-only to the renderer and never logged. Env overrides for QA: `HELIX_SERVER_URL`, `HELIX_USERNAME`, `HELIX_PASSWORD_FILE`, `HELIX_DESKTOP_USER_DATA`.
- `HELIX_DESKTOP_NO_CHROMIUM_SANDBOX=1` adds `--no-sandbox` (process sandbox only; renderer `sandbox:true` stays). Needed in containers without userns/suid sandbox; never default.

Commands: `npm ci` then `node node_modules/electron/install.js` (fetches the Electron zip), `npm test`, `bash scripts/mutation_check.sh`, `(cd ../web && npm run build) && bash scripts/package.sh` (tar.gz + .deb into `out/`, gitignored). No AppImage: electron-builder needs network downloads of appimagetool.
