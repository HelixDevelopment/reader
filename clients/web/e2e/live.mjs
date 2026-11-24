// Live end-to-end check: the built web client (vite preview, proxied) against a REAL helix-server.
// usage: node e2e/live.mjs <clientUrl> <serverUrl> <user> <pass> <outDir> [chromePath]
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'

const [client, server, user, pass, out, chrome] = process.argv.slice(2)
mkdirSync(out, { recursive: true })
const auth = { Authorization: 'Basic ' + Buffer.from(`${user}:${pass}`).toString('base64') }
const apiWorks = await (await fetch(server + '/v1/works?limit=100', { headers: auth })).json()
const expected = apiWorks.works.map((w) => w.canonical_title)
const checks = []
const check = (id, ok, evidence) => { checks.push({ id, status: ok ? 'PASS' : 'FAIL', evidence }); console.log(ok ? 'PASS' : 'FAIL', id, JSON.stringify(evidence)) }

check('server-has-works', expected.length >= 3, { expected })

const browser = await chromium.launch({ executablePath: chrome || undefined, headless: true })
const views = [['phone', 375, 812], ['tablet', 768, 1024], ['desktop', 1280, 900]]

async function signIn(page) {
  await page.getByRole('button', { name: 'Settings' }).click()
  await page.getByLabel('Username (optional)').fill(user)
  await page.getByLabel(/^Password/).fill(pass)
  await page.getByRole('button', { name: 'Save' }).click()
}

for (const scheme of ['light', 'dark']) {
  for (const [name, width, height] of views) {
    const ctx = await browser.newContext({ viewport: { width, height }, colorScheme: scheme, reducedMotion: 'reduce' })
    const page = await ctx.newPage()
    const errors = []
    page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()) })
    const tag = `${name}-${scheme}`
    await page.goto(client)
    // Without credentials the live server answers 401; the client must say so.
    await page.getByRole('alert').waitFor()
    check(`${tag}:401-error-state`, /rejected the credentials/.test(await page.getByRole('alert').innerText()), await page.getByRole('alert').innerText())
    await signIn(page)
    await page.getByRole('heading', { level: 2, name: expected[0] }).waitFor()
    const shown = await page.locator('.card h2').allInnerTexts()
    check(`${tag}:real-titles-in-dom`, expected.every((t) => shown.includes(t)) && shown.length === expected.length, { shown })
    const m = await page.evaluate(() => ({ sw: document.documentElement.scrollWidth, cw: document.documentElement.clientWidth }))
    check(`${tag}:no-horizontal-overflow`, m.sw <= m.cw, m)
    const overlap = await page.evaluate(() => {
      const r = [...document.querySelectorAll('.card, .top, .toolbar')].map((e) => e.getBoundingClientRect())
      const bad = []
      for (let i = 0; i < r.length; i++) for (let j = i + 1; j < r.length; j++) {
        const a = r[i], b = r[j]
        if (a.left < b.right - 1 && b.left < a.right - 1 && a.top < b.bottom - 1 && b.top < a.bottom - 1) bad.push([i, j])
      }
      return bad
    })
    check(`${tag}:no-overlap`, overlap.length === 0, overlap)
    const small = await page.evaluate(() => [...document.querySelectorAll('button, input, a.button')].filter((e) => { const b = e.getBoundingClientRect(); return b.width > 0 && b.height < 40 }).length)
    check(`${tag}:touch-targets>=40px`, small === 0, { smaller: small })
    const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
    check(`${tag}:theme-applied`, scheme === 'dark' ? bg === 'rgb(20, 19, 15)' : bg === 'rgb(250, 248, 244)', bg)
    await page.screenshot({ path: `${out}/${tag}-library.png`, fullPage: true })

    if (scheme === 'light') {
      await page.getByRole('searchbox', { name: 'Search library' }).fill('Glass')
      await page.waitForFunction(() => document.querySelectorAll('.card').length === 1)
      check(`${tag}:search-filters-server-side`, (await page.locator('.card h2').innerText()) === 'The Glass Meridian', await page.locator('.card h2').innerText())
      await page.getByRole('searchbox', { name: 'Search library' }).fill('zzz-no-such-book')
      await page.getByText(/No works match/).waitFor()
      check(`${tag}:empty-state`, true, 'No works match shown')
      await page.getByRole('searchbox', { name: 'Search library' }).fill('')
      await page.locator('.card').nth(2).waitFor()

      // EPUB: honest limitation + the real bytes downloadable.
      await page.getByRole('button', { name: 'Open The Glass Meridian' }).click()
      await page.getByRole('heading', { level: 1, name: 'The Glass Meridian' }).waitFor()
      check(`${tag}:detail-author`, (await page.locator('.detail').innerText()).includes('Elena Voss'), 'author from EPUB metadata')
      await page.screenshot({ path: `${out}/${tag}-detail-epub.png`, fullPage: true })
      await page.getByRole('button', { name: 'Read EPUB' }).click()
      await page.getByText(/does not render EPUB/).waitFor()
      const dl = page.getByRole('link', { name: /Download original/ })
      const [download] = await Promise.all([page.waitForEvent('download'), dl.click()])
      const path = await download.path()
      const { statSync, readFileSync } = await import('node:fs')
      check(`${tag}:epub-download-is-real-zip`, readFileSync(path).subarray(0, 2).toString() === 'PK' && statSync(path).size > 100, { suggested: download.suggestedFilename(), bytes: statSync(path).size })
      await page.screenshot({ path: `${out}/${tag}-reader-epub-unsupported.png`, fullPage: true })

      // TXT: real text from the content route.
      await page.goto(client + '#/')
      await page.getByRole('button', { name: 'Open field-notes' }).click()
      await page.getByRole('button', { name: 'Read TXT' }).click()
      await page.getByText('A second paragraph proves paragraph splitting.').waitFor()
      const paras = await page.locator('article.reading p').count()
      check(`${tag}:txt-reader-renders-live-content`, paras >= 3, { paragraphs: paras })
      const before = await page.locator('article.reading').evaluate((e) => getComputedStyle(e).fontSize)
      await page.getByRole('button', { name: 'Increase text size' }).click()
      const after = await page.locator('article.reading').evaluate((e) => getComputedStyle(e).fontSize)
      check(`${tag}:text-size-delta`, before !== after, { before, after })
      await page.screenshot({ path: `${out}/${tag}-reader-txt.png`, fullPage: true })

      // Unreachable base URL: error + retry, not a blank page.
      await page.goto(client + '#/settings')
      await page.getByLabel('Server address').fill('http://127.0.0.1:9')
      await page.getByRole('button', { name: 'Save' }).click()
      await page.getByRole('alert').waitFor()
      check(`${tag}:unreachable-error-state`, /Cannot reach the server/.test(await page.getByRole('alert').innerText()), await page.getByRole('alert').innerText())
      check(`${tag}:retry-button`, await page.getByRole('button', { name: 'Retry' }).isVisible(), 'visible')
      await page.screenshot({ path: `${out}/${tag}-error.png`, fullPage: true })
    }
    check(`${tag}:no-unexpected-console-errors`, errors.every((e) => /status of 401|ERR_UNSAFE_PORT/.test(e)), errors)
    await ctx.close()
  }
}
await browser.close()
const failed = checks.filter((c) => c.status !== 'PASS')
writeFileSync(`${out}/results.json`, JSON.stringify({ client, server_titles: expected, chrome: chrome || 'playwright default', passed: checks.length - failed.length, failed: failed.length, checks }, null, 2) + '\n')
console.log(`\n${checks.length - failed.length}/${checks.length} checks passed`)
process.exit(failed.length ? 1 : 0)
