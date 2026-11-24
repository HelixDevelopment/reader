// isolation: runs-inside-container
// In-container TCP->UNIX-socket bridge: listens on the container's OWN loopback and forwards to the single socket the
// host handed in. The container has no network namespace connectivity beyond this.
// usage: node sockbridge.js <listen-host> <listen-port> <unix-socket>
const net = require('node:net')
const [host, port, sock] = process.argv.slice(2)
net.createServer((c) => {
  const u = net.connect(sock)
  c.pipe(u); u.pipe(c)
  const end = () => { c.destroy(); u.destroy() }
  c.on('error', end); u.on('error', end)
}).listen(+port, host)
