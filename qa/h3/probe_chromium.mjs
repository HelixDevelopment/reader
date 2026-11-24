// Runs INSIDE the h3 lab client container. Drives a Chromium (Playwright chromium, or any chrome binary) at the helix-server
// and records, per request, the NEGOTIATED protocol (CDP Network.responseReceived.response.protocol: "h3" / "h2" / "http/1.1")
// and the Content-Encoding. Pass criterion (strict): after Alt-Svc discovery the repeat request negotiates h3 AND content-encoding br.
// usage: node probe_chromium.mjs <chrome-binary> <url> <out.json> [extra chrome flags...]
import { createRequire } from 'node:module'
import { writeFileSync } from 'node:fs'
const require = createRequire(process.env.PW_REQUIRE_FROM || '/work/desktop/package.json')
const { chromium } = require('playwright-core')
const [bin, url, out, ...flags] = process.argv.slice(2)
const browser = await chromium.launch({ executablePath: bin, headless: true, args: ['--no-sandbox', ...flags] })
const ctx = await browser.newContext({ extraHTTPHeaders: {} })
const page = await ctx.newPage()
const cdp = await ctx.newCDPSession(page)
await cdp.send('Network.enable')
const seen = []
cdp.on('Network.responseReceived', (e) => seen.push({ url: e.response.url, status: e.response.status, protocol: e.response.protocol, encoding: e.response.headers['content-encoding'] || e.response.headers['Content-Encoding'] || null, altsvc: e.response.headers['alt-svc'] || null }))
const rounds = []
for (let i = 0; i < 4; i++) {
  const before = seen.length
  await page.goto(url, { waitUntil: 'load' })
  if (i === 0) await new Promise((r) => setTimeout(r, 1500)) // let Chromium learn Alt-Svc and race a QUIC session
  rounds.push(seen.slice(before).filter((s) => s.url.startsWith(url.split('/').slice(0, 3).join('/'))))
  if (i < 3) await new Promise((r) => setTimeout(r, 800))
}
const last = rounds[rounds.length - 1][0] || {}
const result = { url, flags, rounds, finalProtocol: last.protocol || null, finalEncoding: last.encoding || null, version: browser.version(), pass: last.protocol === 'h3' && last.encoding === 'br' }
writeFileSync(out, JSON.stringify(result, null, 2))
console.log(JSON.stringify({ finalProtocol: result.finalProtocol, finalEncoding: result.finalEncoding, protocols: rounds.map((r) => r.map((x) => x.protocol)), pass: result.pass, version: result.version }))
await browser.close()
process.exit(result.pass ? 0 : 3)
