// Live GUI check for the packaged/unpacked app via Playwright _electron.
// RUN ONLY inside the isolated rootless container (qa/isolation/run_desktop_live.sh: private Xvfb, --network none, no host display);
// the operator forbids windows on the host display.
// usage: node e2e/live.mjs <electronExecutable> <serverUrl> <user> <passwordFile> <outDir> [appDirOrEmpty]
// Pass the unpacked helix-reader binary (out/helix-reader-linux-x64/helix-reader) as <electronExecutable>
// with no appDir, or node_modules/electron/dist/electron with appDir=clients/desktop.
import { _electron as electron } from 'playwright-core'
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync, renameSync, readdirSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const [exe, server, user, pwFile, out, appDir] = process.argv.slice(2)
mkdirSync(out, { recursive: true })
const password = readFileSync(pwFile, 'utf8').trim()
const auth = { Authorization: 'Basic ' + Buffer.from(`${user}:${password}`).toString('base64') }
const expected = (await (await fetch(server + '/v1/works?limit=100', { headers: auth })).json()).works.map((w) => w.canonical_title)
import { sandboxStatus } from './sandbox_status.mjs'
const checks = []
const check = (id, ok, evidence) => { checks.push({ id, status: ok ? 'PASS' : 'FAIL', evidence }); console.log(ok ? 'PASS' : 'FAIL', id, JSON.stringify(evidence)) }
check('server-has-works', expected.length >= 3, { expected })

const userData = mkdtempSync(join(tmpdir(), 'hrd-desktop-'))
const env = { ...process.env, HELIX_DESKTOP_USER_DATA: userData, HELIX_SERVER_URL: server, HELIX_USERNAME: user, HELIX_PASSWORD_FILE: pwFile, HELIX_DESKTOP_NO_CHROMIUM_SANDBOX: process.env.HELIX_DESKTOP_NO_CHROMIUM_SANDBOX ?? '1' }
const app = await electron.launch({ executablePath: exe, args: appDir ? [appDir] : [], env, chromiumSandbox: process.env.HELIX_EXPECT_SANDBOX === '1' })
const page = await app.firstWindow()
const errors = []
page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
page.on('pageerror', (e) => errors.push(String(e)))
try {
  check('window-title', (await page.title()) !== undefined, { title: await page.title() })
  check('origin-is-app-scheme', page.url().startsWith('app://helix/'), { url: page.url() })
  await page.getByRole('heading', { level: 2, name: expected[0] }).waitFor({ timeout: 15000 })
  const shown = await page.locator('.card h2').allInnerTexts()
  check('real-titles-in-dom', expected.every((t) => shown.includes(t)) && shown.length === expected.length, { shown })
  await page.screenshot({ path: join(out, '01_library.png') })
  // Chromium sandbox state, read from /proc (the test runs in the same container as the app): renderer processes with
  // seccomp filter mode 2 and no --no-sandbox on any Electron command line = the renderer sandbox is really on.
  const procs = readdirSync('/proc').filter((d) => /^\d+$/.test(d)).map((pid) => {
    try { return { pid, cmd: readFileSync(`/proc/${pid}/cmdline`, 'utf8').split('\0').join(' '), status: readFileSync(`/proc/${pid}/status`, 'utf8') } } catch { return null }
  }).filter(Boolean)
  const electronProcs = procs.filter((p) => p.cmd.includes('helix-reader'))
  const renderers = electronProcs.filter((p) => p.cmd.includes('--type=renderer'))
  const sb = { electron_processes: electronProcs.length, renderers: renderers.map((r) => ({ pid: r.pid, seccomp: /Seccomp:\s*(\d)/.exec(r.status)?.[1] })),
               any_no_sandbox_flag: electronProcs.some((p) => p.cmd.includes('--no-sandbox')) }
  const sandboxed = renderers.length > 0 && !sb.any_no_sandbox_flag && renderers.every((r) => r.seccomp === '2')
  const sbStatus = sandboxStatus(sandboxed, process.env.HELIX_EXPECT_SANDBOX === '1')   // PASS only when really ON; OFF is WARN (recorded, NOT verified)
  checks.push({ id: 'chromium-sandbox-state', status: sbStatus, evidence: { ...sb, sandboxed, expected: process.env.HELIX_EXPECT_SANDBOX === '1' } })
  console.log(sbStatus, 'chromium-sandbox-state', JSON.stringify({ ...sb, sandboxed }))
  // renderer hardening, observed from inside the page
  const probe = await page.evaluate(() => ({ require: typeof require, process: typeof process, shell: typeof window.helixShell }))
  check('no-node-in-renderer', probe.require === 'undefined' && probe.process === 'undefined' && probe.shell === 'undefined', probe)
  const net = await page.evaluate(async () => { try { await fetch('https://example.com/', { mode: 'no-cors' }); return 'allowed' } catch { return 'blocked' } })
  check('csp-blocks-external-fetch', net === 'blocked', { net })
  const cookie = await page.evaluate(() => document.cookie)
  check('no-cookies', cookie === '', { cookie })
  await page.getByRole('heading', { level: 2, name: expected[0] }).click()
  await page.screenshot({ path: join(out, '02_detail.png') })
  // The deliberate CSP probe above logs its own (expected) blocked-fetch errors; every OTHER console error is a failure.
  const unexpected = errors.filter((e) => !/example\.com/.test(e))
  check('no-console-errors', unexpected.length === 0, { unexpected, expected_csp_probe_errors: errors.length - unexpected.length })
  const raw = readdirSync(userData).map((f) => f)
  const files = raw.filter((f) => f.endsWith('.json')).map((f) => readFileSync(join(userData, f), 'utf8')).join('\n')
  check('password-never-in-userdata-json', !files.includes(password), { files: raw })
  // Runs LAST: the app (correctly) cancels the cross-origin navigation in the main process, but Playwright then keeps
  // waiting for that never-committed navigation and every later actionability wait stalls (observed in the container run).
  const nav = await page.evaluate(() => { try { location.href = 'https://example.com/' } catch (e) { return String(e) } return 'attempted' })
  await page.waitForTimeout(500)
  check('navigation-locked-to-app-origin', page.url().startsWith('app://helix/'), { nav, url: page.url() })
} finally {
  await app.close()
}
const failed = checks.filter((c) => c.status === 'FAIL').length
const warned = checks.filter((c) => c.status === 'WARN').length
writeFileSync(join(out, 'results.json.tmp'), JSON.stringify({ gui: true, checks, failed }, null, 2)); renameSync(join(out, 'results.json.tmp'), join(out, 'results.json'))
console.log(`${checks.filter((c) => c.status === 'PASS').length}/${checks.length} PASS, ${warned} WARN (recorded, not verified), ${failed} FAIL`); process.exit(failed ? 1 : 0)
