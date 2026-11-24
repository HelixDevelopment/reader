#!/usr/bin/env bash
# reap.sh - stale-resource reaper, LIMITED to containers named hrdiso-* / hrdisorev-* that carry our label.
# A container is stale when its owner is gone: label org.helix.owner-pid (runner process dead) or org.helix.owner-file
# (a pid file that is missing or names a dead process). Never touches any other name.
# N4 startup grace: a container younger than REAP_GRACE seconds (default 120; label org.helix.lease-ts, else the creation time)
# is NEVER reaped - its starter may not have written/refreshed the owner file yet. Starters also write the owner file first.
set -u
GRACE=${REAP_GRACE:-120}
now=$(date +%s); n=0
lab() { podman inspect "$1" --format "{{index .Config.Labels \"$2\"}}" 2>/dev/null; }
while read -r name; do
  case $name in hrdiso-*|hrdisorev-*) ;; *) continue ;; esac
  pid=$(lab "$name" org.helix.owner-pid); file=$(lab "$name" org.helix.owner-file); lease=$(lab "$name" org.helix.lease-ts)
  case $lease in ''|*[!0-9]*) lease=$(podman inspect "$name" --format '{{.Created.Unix}}' 2>/dev/null) ;; esac
  case $lease in ''|*[!0-9]*) continue ;; esac            # unknown age: fail safe, leave it alone
  [ $((now - lease)) -lt "$GRACE" ] && continue
  stale=0
  # dead = kill -0 fails with ESRCH ("No such process"). EPERM means the pid exists (another user's process): NOT dead, fail safe.
  dead() { local e; case $1 in ''|*[!0-9]*) return 1;; esac
    e=$(LC_ALL=C kill -0 "$1" 2>&1) && return 1; case $e in *"No such process"*) return 0;; *) return 1;; esac; }
  if [ -n "$pid" ] && [ "$pid" != "<no value>" ]; then dead "$pid" && stale=1; fi
  if [ -n "$file" ] && [ "$file" != "<no value>" ]; then if [ -f "$file" ]; then dead "$(cat "$file")" && stale=1; else stale=1; fi; fi
  if [ "$stale" = 1 ]; then podman rm -f "$name" >/dev/null 2>&1 && { echo "reaped $name"; n=$((n+1)); }; fi
done < <(podman ps -a --filter label=org.helix.isolation --format '{{.Names}}')
echo "reaper: $n stale container(s) removed"
