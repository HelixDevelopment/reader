#!/usr/bin/env bash
# Mutation check for the security-critical policy modules: each mutation must make
# `node --test` fail; the original file is always restored. Exit 1 if any mutant survives.
set -u
cd "$(dirname "$0")/.."
survived=0
run() { # file sed-expr label [equivalent]
  cp "$1" "$1.orig"
  sed -i -E "$2" "$1"
  if cmp -s "$1" "$1.orig"; then echo "MUTATION-NOOP  $3 (sed did not change the file)"; survived=1
  elif node --test test/ >/dev/null 2>&1; then
    if [ "${4:-}" = equivalent ]; then echo "EQUIVALENT     $3 (unreachable: '..' segments are rejected earlier; containment kept as defence in depth)"
    else echo "SURVIVED       $3"; survived=1; fi
  else echo "KILLED         $3"; fi
  mv "$1.orig" "$1"
}
run lib/allowlist.js "s/u.protocol === 'app:' && u.host === 'helix'/u.protocol === 'app:'/" "navigation: drop host check"
run lib/allowlist.js "0,/u.protocol === 'app:' && u.host === 'helix'/s//(u.protocol === 'app:' || u.protocol === 'https:')/" "navigation: allow https"
run lib/allowlist.js "s/if \(u.protocol === 'blob:' \|\| u.protocol === 'data:'\) return true/if (u.protocol === 'blob:' || u.protocol === 'data:' || u.protocol === 'https:') return true/" "request: allow https"
run lib/allowlist.js "s/if \(u.username \|\| u.password\)//; s/ \|\| u.username \|\| u.password//" "server url: allow embedded credentials"
run lib/proxy.js "s/method !== 'GET' && method !== 'HEAD'/false/" "proxy: allow all methods"
run lib/proxy.js "s/'set-cookie', //" "proxy: leak set-cookie"
run lib/proxy.js "s/'accept-language', /'accept-language', 'cookie', /" "proxy: forward cookie"
run lib/proxy.js "s/'accept-language', /'accept-language', 'host', /" "proxy: forward host"
run lib/proxy.js "s/'accept-language', /'accept-language', 'x-http-method-override', /" "proxy: forward x-http-method-override"
run lib/proxy.js "s/'accept-language', /'accept-language', 'x-forwarded-for', 'x-forwarded-host', /" "proxy: forward x-forwarded-*"
run main.js "s/\(sameOrigin\(conn.serverUrl, serverUrl\) \? conn.password : ''\)/conn.password/" "main: carry stored password to a different server origin"
run lib/proxy.js "s/\|\| \/%2e\|%2f\|%5c\|\\\\\\\\\/i.test\(u.pathname\)//" "proxy: allow encoded dot/slash"
run lib/static.js "s/if \(p.split\('\/'\).includes\('..'\)\) return null//" "static: allow .. segments"
run lib/static.js "s/abs !== rootAbs && !abs.startsWith\(rootAbs \+ path.sep\)/false/" "static: no root containment" equivalent
run lib/csp.js "s/'script-src': \[\"'self'\"\]/'script-src': [\"'self'\", \"'unsafe-inline'\"]/" "csp: script unsafe-inline"
run lib/secrets.js "s/if \(persisted && password\)/if (password)/" "secrets: persist without encryption"
exit $survived
