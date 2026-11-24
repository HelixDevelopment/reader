// Runs INSIDE the h3 lab client container (private Xvfb from the image entrypoint). Launches the Electron desktop shell against the
// helix-server and issues API reads from the renderer through the app:// forwarder (main process -> server). The negotiated protocol
// is NOT read from the client's own claim: qa/h3/run_desktop_h3.sh reads the SERVER's access log (proto=HTTP/3.0 | HTTP/2.0 | HTTP/1.1).
// usage: node desktop_h3.mjs <electronExecutable> <appDir> <outJson>      env: HELIX_SERVER_URL, HELIX_DESKTOP_* (transport settings)
import { createRequire } from 'node:module'
import { mkdtempSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
const require = createRequire('/work/desktop/package.json')
const { _electron: electron } = require('playwright-core')
const [exe, appDir, out] = process.argv.slice(2)
const env = { ...process.env, HELIX_DESKTOP_USER_DATA: mkdtempSync(join(tmpdir(), 'hrd-h3-')), HELIX_DESKTOP_WEB_ROOT: '/work/h3/fixture_web', HELIX_DESKTOP_NO_CHROMIUM_SANDBOX: '1' }
const app = await electron.launch({ executablePath: exe, args: [appDir], env })
const page = await app.firstWindow()
await page.waitForLoadState('load')
const results = []
for (let i = 0; i < 4; i++) {
  const r = await page.evaluate(async () => { const x = await fetch('/v1/formats'); const j = await x.json(); return { status: x.status, formats: (j.formats || []).length } })
  results.push(r)
  await new Promise((r2) => setTimeout(r2, 700))
}
const version = await app.evaluate(() => ({ electron: process.versions.electron, chrome: process.versions.chrome }))
writeFileSync(out, JSON.stringify({ results, version, url: page.url() }, null, 2))
console.log(JSON.stringify({ results, version }))
await app.close()
process.exit(results.every((r) => r.status === 200 && r.formats > 0) ? 0 : 3)
