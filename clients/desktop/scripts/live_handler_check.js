#!/usr/bin/env node
'use strict'
// Live, non-GUI check: drives the app:// handler (the code main.js registers) with a
// real running helix-server. usage: live_handler_check.js <serverUrl> <user> <passwordFile> <webDist> <outJson>
// No Electron and no display are involved: this proves the proxy/static layer only.
const fs = require('node:fs')
const path = require('node:path')
const { createHandler } = require('../lib/app-handler')
const [server, user, pwFile, webDist, out] = process.argv.slice(2)
const password = fs.readFileSync(pwFile, 'utf8').trim()
const checks = []
const check = (id, ok, evidence) => { checks.push({ id, status: ok ? 'PASS' : 'FAIL', evidence }); console.log(ok ? 'PASS' : 'FAIL', id, JSON.stringify(evidence)) }
const mk = (conn) => createHandler({ getConn: () => conn, webRoot: webDist, shellRoot: path.join(__dirname, '../renderer') })
;(async () => {
  const basic = { Authorization: 'Basic ' + Buffer.from(`${user}:${password}`).toString('base64') }
  const direct = await (await fetch(server + '/v1/works?limit=100', { headers: basic })).json()
  const titles = direct.works.map((w) => w.canonical_title)
  check('server-direct-has-works', titles.length >= 3, { titles })
  const noCors = (await fetch(server + '/v1/works', { headers: basic })).headers.get('access-control-allow-origin')
  check('server-sends-no-cors-headers (why the proxy exists)', noCors === null, { 'access-control-allow-origin': noCors })

  const h = mk({ serverUrl: server, username: user, password })
  const r = await h(new Request('app://helix/v1/works?limit=100'))
  const body = await r.json()
  check('proxy-with-stored-creds-returns-real-titles', r.status === 200 && JSON.stringify(body.works.map((w) => w.canonical_title)) === JSON.stringify(titles), { status: r.status, titles: body.works.map((w) => w.canonical_title) })
  const wid = body.works[0].id
  const d = await h(new Request('app://helix/v1/works/' + encodeURIComponent(wid)))
  check('proxy-work-detail', d.status === 200 && (await d.json()).canonical_title === body.works[0].canonical_title, { status: d.status })
  const hz = await h(new Request('app://helix/healthz'))
  check('proxy-healthz', hz.status === 200, { status: hz.status })

  const bad = await mk({ serverUrl: server, username: user, password: 'wrong-' + password })(new Request('app://helix/v1/works'))
  check('wrong-password-gives-401-and-no-www-authenticate-prompt', bad.status === 401 && bad.headers.get('www-authenticate') === null, { status: bad.status, wwwAuth: bad.headers.get('www-authenticate') })
  const anon = await mk({ serverUrl: server, username: '', password: '' })(new Request('app://helix/v1/works'))
  check('no-creds-gives-401', anon.status === 401, { status: anon.status })
  const rend = await mk({ serverUrl: server, username: '', password: '' })(new Request('app://helix/v1/works', { headers: basic }))
  check('renderer-supplied-authorization-is-forwarded', rend.status === 200, { status: rend.status })

  const idx = await h(new Request('app://helix/index.html'))
  const html = await idx.text()
  check('static-index-from-web-dist-with-csp', idx.status === 200 && /<div id="root"/.test(html) && /default-src 'none'/.test(idx.headers.get('content-security-policy')), { status: idx.status, csp: idx.headers.get('content-security-policy') })
  const js = html.match(/src="\.\/(assets\/[^"]+\.js)"/)
  const jr = js && (await h(new Request('app://helix/' + js[1])))
  check('static-js-bundle-served', !!jr && jr.status === 200 && /javascript/.test(jr.headers.get('content-type')), { asset: js && js[1], status: jr && jr.status })
  for (const p of ['/..%2f..%2f..%2fetc%2fpasswd', '/%2e%2e/%2e%2e/etc/passwd', '/v1/%2e%2e/admin']) {
    const t = await h(new Request('app://helix' + p))
    check('traversal-refused ' + p, t.status === 400 || t.status === 404, { status: t.status })
  }
  const post = await h(new Request('app://helix/v1/sources', { method: 'POST', body: '{}' }))
  check('proxy-refuses-POST', post.status === 405, { status: post.status })
  const dead = await mk({ serverUrl: 'http://127.0.0.1:9', username: user, password })(new Request('app://helix/v1/works'))
  check('unreachable-server-gives-502', dead.status === 502, { status: dead.status })

  const failed = checks.filter((c) => c.status === 'FAIL').length
  fs.writeFileSync(out + '.tmp', JSON.stringify({ gui: false, note: 'handler-level live check; no Electron window', checks, failed }, null, 2)); fs.renameSync(out + '.tmp', out)
  console.log(`\n${checks.length - failed}/${checks.length} PASS`)
  process.exit(failed ? 1 : 0)
})().catch((e) => { console.error('ERROR', e.message); process.exit(2) })
