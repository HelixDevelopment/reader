// isolation: runs-inside-container
// usage: node net_probe.js host:port[,host:port...]   -> prints one JSON line {"host:port":"open|closed"}; 1.5 s timeout each.
const net = require('node:net')
const targets = process.argv[2].split(',').filter(Boolean)
const out = {}
let pending = targets.length
if (!pending) { console.log('{}'); process.exit(0) }
for (const t of targets) {
  const i = t.lastIndexOf(':'); const host = t.slice(0, i); const port = +t.slice(i + 1)
  const s = net.connect({ host, port, timeout: 1500 })
  const done = (r) => { if (out[t]) return; out[t] = r; s.destroy(); if (--pending === 0) { console.log(JSON.stringify(out)); process.exit(0) } }
  s.on('connect', () => done('open')); s.on('timeout', () => done('closed')); s.on('error', () => done('closed'))
}
