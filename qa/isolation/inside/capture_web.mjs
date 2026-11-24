// isolation: runs-inside-container
// HRD-076 capture of the WEB client (vite preview of the built clients/web) in headless Chromium INSIDE the container.
// usage: node capture_web.mjs <clientUrl> <serverUrl> <user> <pass> <outDir> <chromePath>
// Writes PNGs + one WEBM walkthrough + scenes.json (what each image shows). sha256/size are added HOST-side by make_capture_manifest.py.
import { mkdirSync, writeFileSync } from 'node:fs'
const { chromium } = await import('/work/web/node_modules/playwright-core/index.mjs')
const [client, server, user, pass, out, chrome] = process.argv.slice(2)
mkdirSync(out, { recursive: true })
const scenes = []
const browser = await chromium.launch({ executablePath: chrome, headless: true })
const version = browser.version()
const rec = (file, scenario, w, h, scheme, note) => scenes.push({ file, scenario, viewport: `${w}x${h}`, color_scheme: scheme, note })
async function signIn(page) {
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.getByLabel('Username (optional)').fill(user)
  await page.getByLabel(/^Password/).fill(pass)
  await page.getByRole('button', { name: 'Save' }).click()
  await page.locator('.card').first().waitFor()
}
async function shot(page, file, scenario, w, h, scheme, note, full = false) {
  await page.screenshot({ path: `${out}/${file}`, fullPage: full }); rec(file, scenario, w, h, scheme, note)
}
// 1. responsive + theme matrix of the library
for (const scheme of ['light', 'dark']) for (const [n, w, h] of [['phone', 390, 844], ['tablet', 820, 1100], ['desktop', 1366, 900]]) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, colorScheme: scheme, reducedMotion: 'reduce' }); const page = await ctx.newPage()
  await page.goto(client); await signIn(page)
  await shot(page, `web-library-${n}-${scheme}.png`, `Library grid, ${n} layout, ${scheme} theme (colour scheme follows the OS setting)`, w, h, scheme, 'first page of 24 works of 27')
  await ctx.close()
}
// 2. desktop walkthrough with a video: search, detail, reader, text size, paging, unreachable-server error
const W = 1366, H = 900
const ctx = await browser.newContext({ viewport: { width: W, height: H }, colorScheme: 'light', reducedMotion: 'reduce', recordVideo: { dir: `${out}/video-tmp`, size: { width: W, height: H } } })
const page = await ctx.newPage()
await page.goto(client)
await page.getByRole('alert').waitFor()
await shot(page, 'web-error-401-before-sign-in.png', 'First start before credentials: the client says the server rejected them instead of showing a blank page', W, H, 'light', 'alert text: ' + (await page.getByRole('alert').innerText()).slice(0, 80))
await page.getByRole('button', { name: 'Settings' }).click()
await shot(page, 'web-settings.png', 'Connection settings: server address, username, session-only password', W, H, 'light', 'password field is not shown or stored in localStorage')
await page.getByLabel('Username (optional)').fill(user); await page.getByLabel(/^Password/).fill(pass); await page.getByRole('button', { name: 'Save' }).click()
await page.locator('.card').first().waitFor()
await page.getByRole('button', { name: 'Next' }).scrollIntoViewIfNeeded()
await shot(page, 'web-library-paging.png', 'Library paging: 24 works per page, Next enabled', W, H, 'light', 'pager at the bottom of page 1', true)
await page.getByRole('button', { name: 'Next' }).click(); await page.waitForTimeout(600)
await shot(page, 'web-library-page-2.png', 'Library page 2: the remaining works', W, H, 'light', 'second page after Next', true)
await page.getByRole('button', { name: 'Previous' }).click(); await page.locator('.card').first().waitFor()
const box = page.getByRole('searchbox', { name: 'Search library' })
await box.fill('Glass'); await page.waitForFunction(() => document.querySelectorAll('.card').length === 1)
await shot(page, 'web-search-server-side.png', 'Server-side search: typing "Glass" narrows 27 works to 1', W, H, 'light', 'query Glass')
await box.fill('zzz-no-such-book'); await page.getByText(/No works match/).waitFor()
await shot(page, 'web-search-empty-state.png', 'Search with no match shows an explicit empty state', W, H, 'light', 'query zzz-no-such-book')
await box.fill('field-notes'); await page.waitForFunction(() => document.querySelectorAll('.card').length === 1)
await page.getByRole('button', { name: 'Open field-notes' }).click()
await page.getByRole('button', { name: 'Read TXT' }).waitFor()
await shot(page, 'web-work-detail.png', 'Work detail: title, formats and the Read action for a plain-text work', W, H, 'light', 'field-notes')
await page.getByRole('button', { name: 'Read TXT' }).click()
await page.getByText('A second paragraph proves paragraph splitting in the reader.').waitFor()
await shot(page, 'web-reader-txt.png', 'Built-in text reader at the default size', W, H, 'light', 'default font size')
for (let i = 0; i < 4; i++) await page.getByRole('button', { name: 'Increase text size' }).click()
const fs = await page.locator('article.reading').evaluate((e) => getComputedStyle(e).fontSize)
await shot(page, 'web-reader-large-text.png', 'Reader text size raised with A+ four times', W, H, 'light', 'computed font-size ' + fs)
await page.goto(client + '#/'); await page.locator('.card').first().waitFor()
await page.getByRole('button', { name: 'Open The Glass Meridian' }).click()
await page.getByRole('heading', { level: 1, name: 'The Glass Meridian' }).waitFor()
await shot(page, 'web-detail-epub-metadata.png', 'EPUB work: author and title come from the file metadata; original is downloadable', W, H, 'light', 'The Glass Meridian / Elena Voss')
await page.getByRole('button', { name: 'Read EPUB' }).click(); await page.getByText(/does not render EPUB/).waitFor()
await shot(page, 'web-epub-honest-limitation.png', 'EPUB in the web client: honest "does not render EPUB" notice with a Download original link (the web reader supports TXT only)', W, H, 'light', 'limitation, not a bug in the capture')
await page.goto(client + '#/'); await page.locator('.card').first().waitFor()
const themeBtn = page.getByRole('button', { name: /^Theme:/ })
const t0 = await themeBtn.innerText(); await themeBtn.click(); const t1 = await themeBtn.innerText()
await shot(page, 'web-theme-toggle-1.png', `Theme switch in the header: "${t0}" -> "${t1}" after one click`, W, H, 'light', 'label before: ' + t0 + ', after: ' + t1)
await themeBtn.click(); const t2 = await themeBtn.innerText()
await shot(page, 'web-theme-toggle-2.png', `Theme switch, second click: "${t2}"`, W, H, 'light', 'label: ' + t2)
await themeBtn.click()
await page.goto(client + '#/settings'); await page.getByLabel('Server address').fill('http://127.0.0.1:9'); await page.getByRole('button', { name: 'Save' }).click()
await page.getByRole('alert').waitFor()
await shot(page, 'web-error-unreachable-retry.png', 'Unreachable server: error with a Retry button', W, H, 'light', 'server address http://127.0.0.1:9 (nothing listens there)')
await ctx.close()   // flushes the video
const { readdirSync, renameSync, rmSync } = await import('node:fs')
const vids = readdirSync(`${out}/video-tmp`); renameSync(`${out}/video-tmp/${vids[0]}`, `${out}/web-walkthrough-desktop.webm`); rmSync(`${out}/video-tmp`, { recursive: true })
rec('web-walkthrough-desktop.webm', 'Recorded walkthrough of the desktop-size session above (Playwright recordVideo, no host capture)', W, H, 'light', 'video')
await browser.close()
writeFileSync(`${out}/scenes.json`, JSON.stringify({ client: 'web', browser: `Chromium ${version} (headless, Playwright ${'1.65.1'})`, scenes }, null, 2) + '\n')
console.log('scenes', scenes.length)
