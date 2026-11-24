// isolation: runs-inside-container
// HRD-076 capture of the Linux DESKTOP client (packaged Electron app) on the container-private Xvfb :99. Playwright _electron drives the app.
// usage: node capture_desktop.mjs <electronExecutable> <serverUrl> <user> <passwordFile> <outDir>
import { mkdirSync, mkdtempSync, writeFileSync, readFileSync, readdirSync, renameSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { execFileSync } from 'node:child_process'
const { _electron: electron } = await import('/work/node_modules/playwright-core/index.mjs')
const [exe, server, user, pwFile, out] = process.argv.slice(2)
mkdirSync(out, { recursive: true })
const scenes = []
const framesDir = mkdtempSync(join(tmpdir(), 'hrd076-frames-')); let frameNo = 0
const frame = () => { try { execFileSync('import', ['-display', ':99', '-window', 'root', '-quality', '92', join(framesDir, `f${String(frameNo++).padStart(3, '0')}.jpg`)], { stdio: 'ignore' }) } catch (e) { console.log('frame failed', String(e).slice(0, 200)) } }
const rec = (file, scenario, size, note) => scenes.push({ file, scenario, viewport: size, color_scheme: 'app', note })
const userData = mkdtempSync(join(tmpdir(), 'hrd076-desktop-'))
const env = { ...process.env, HELIX_DESKTOP_USER_DATA: userData, HELIX_SERVER_URL: server, HELIX_USERNAME: user, HELIX_PASSWORD_FILE: pwFile, HELIX_DESKTOP_NO_CHROMIUM_SANDBOX: '1' }
const W = 1280, H = 860
const app = await electron.launch({ executablePath: exe, args: [], env })
const page = await app.firstWindow()
await app.evaluate(({ BrowserWindow }, [w, h]) => { const b = BrowserWindow.getAllWindows()[0]; b.setSize(w, h); b.center() }, [W, H])
await page.waitForTimeout(800)
const version = await app.evaluate(({ app: a }) => `${a.getName()} ${a.getVersion()} (Electron ${process.versions.electron}, Chromium ${process.versions.chrome})`)
const size = async () => { const s = await page.evaluate(() => `${window.innerWidth}x${window.innerHeight}`); return s }
async function shot(file, scenario, note, full = false) { frame(); await page.screenshot({ path: `${out}/${file}`, fullPage: full }); rec(file, scenario, await size(), note) }
try { await page.locator('.card').first().waitFor({ timeout: 20000 }) } catch (e) { await page.screenshot({ path: `${out}/debug-fail.png` }); console.log('BODY', (await page.locator('body').innerText()).slice(0, 400)); throw e }
await shot('desktop-library.png', 'Linux desktop client signed in at launch from its stored settings: library of 27 works', 'page content, window ' + W + 'x' + H)
// the real X screen of the private Xvfb :99 (window frame included), taken by ImageMagick inside the container
try { execFileSync('import', ['-display', ':99', '-window', 'root', `${out}/desktop-xvfb-root-library.png`], { stdio: 'ignore' }); rec('desktop-xvfb-root-library.png', 'Whole container-private X screen (Xvfb :99) with the app window, via ImageMagick import inside the container', 'root', 'not the operator display') } catch (e) { console.log('root shot failed', String(e).slice(0, 100)) }
const box = page.getByRole('searchbox', { name: 'Search library' })
await box.fill('Glass'); await page.waitForFunction(() => document.querySelectorAll('.card').length === 1)
await shot('desktop-search.png', 'Server-side search "Glass" narrows 27 works to 1', 'query Glass')
await box.fill('field-notes'); await page.waitForFunction(() => document.querySelectorAll('.card').length === 1)
await page.getByRole('button', { name: 'Open field-notes' }).click()
await page.getByRole('button', { name: 'Read TXT' }).waitFor()
await shot('desktop-work-detail.png', 'Work detail for a plain-text work', 'field-notes')
await page.getByRole('button', { name: 'Read TXT' }).click()
await page.getByText('A second paragraph proves paragraph splitting in the reader.').waitFor()
for (let i = 0; i < 3; i++) await page.getByRole('button', { name: 'Increase text size' }).click()
await shot('desktop-reader-large-text.png', 'Built-in reader with text size raised three steps (A+)', 'font-size ' + await page.locator('article.reading').evaluate((e) => getComputedStyle(e).fontSize))
const themeBtn = page.getByRole('button', { name: /^Theme:/ })
await themeBtn.click(); await themeBtn.click()
await shot('desktop-reader-dark-theme.png', 'Dark theme in the desktop reader (header theme switch clicked twice from system)', 'label ' + await themeBtn.innerText())
frame(); await app.close()
// step video: one frame of the real Xvfb screen per step (ImageMagick import), encoded with the ffmpeg shipped with Playwright (no x11grab in that build)
try {
  const ff = readdirSync('/ms-playwright').filter((d) => d.startsWith('ffmpeg-')).sort().pop()
  execFileSync(`/ms-playwright/${ff}/ffmpeg-linux`, ['-y', '-loglevel', 'error', '-f', 'image2pipe', '-framerate', '1', '-c:v', 'mjpeg', '-i', 'pipe:0', '-c:v', 'libvpx-vp9', '-b:v', '1M', '-pix_fmt', 'yuv420p', `${out}/desktop-walkthrough-steps.webm`], { input: Buffer.concat(readdirSync(framesDir).sort().map((f) => readFileSync(join(framesDir, f)))), stdio: ['pipe', 'inherit', 'inherit'] })
  rec('desktop-walkthrough-steps.webm', 'Step video of the desktop session: one real Xvfb screen frame per step, 1 frame/s (a slideshow of real frames, NOT a continuous screen recording)', `${W}x${H}`, 'video')
} catch (e) { console.log('video encode failed (UNCONFIRMED: no video):', String(e).slice(0, 120)) }
writeFileSync(`${out}/scenes.json`, JSON.stringify({ client: 'desktop', app: version, scenes }, null, 2) + '\n')
console.log('scenes', scenes.length)
