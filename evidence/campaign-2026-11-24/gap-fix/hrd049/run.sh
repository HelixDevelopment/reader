#!/usr/bin/env bash
# HRD-049 race re-run at f9d14dc5e2aea6b21676b3cb4674e0b52be50d73
set -u
D=$(cd "$(dirname "$0")" && pwd)
for m in core server; do
  ( cd $D/../../../../$m
    { echo "# HEAD=f9d14dc5e2aea6b21676b3cb4674e0b52be50d73"; echo "# date=$(date -u +%FT%TZ)"; echo "# cmd: cd $m && GIN_MODE=release go test -race -p 2 ./... -count=1 -v"; go version; } > $D/go-race-$m.log
    GIN_MODE=release GOMAXPROCS=4 go test -race -p 2 ./... -count=1 -v >> $D/go-race-$m.log 2>&1; echo "EXIT=$?" | tee -a $D/go-race-$m.log
  )
done
