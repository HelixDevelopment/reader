#!/bin/sh
# isolation: runs-inside-container
# First thing run in every isolated container (via run_container.sh): RUNTIME assertion that the environment holds no
# host display/session/input/socket, optional unix-socket bridge, then exec the test command.
# usage: launch.sh [--bridge PORT SOCK] -- cmd...   (exit 65 = isolation violation)
set -u
bad() { echo "isolation violation: $*" >&2; exit 65; }
for v in WAYLAND_DISPLAY XAUTHORITY DBUS_SESSION_BUS_ADDRESS DBUS_SYSTEM_BUS_ADDRESS XDG_RUNTIME_DIR XDG_SESSION_TYPE XDG_SESSION_ID SSH_AUTH_SOCK GNOME_DESKTOP_SESSION_ID; do
  eval "val=\${$v:-}"; [ -z "$val" ] || bad "$v is set"
done
[ -z "${DISPLAY:-}" ] || [ "$DISPLAY" = ":99" ] || bad "DISPLAY is not the container-private :99"
if [ -d /tmp/.X11-unix ]; then for s in /tmp/.X11-unix/*; do [ -e "$s" ] || continue; [ "$(basename "$s")" = X99 ] || bad "foreign X socket $s"; done; fi
for p in /var/run/docker.sock /run/podman /dev/input /dev/uinput /dev/dri /run/dbus/system_bus_socket; do [ ! -e "$p" ] || bad "host path present: $p"; done
# /run/user and /run/dbus exist as EMPTY dirs in the image; anything inside would be a host-provided session
[ -z "$(ls -A /run/user 2>/dev/null)" ] || bad "/run/user is not empty"
if grep -E -i 'wayland|X11-unix|dbus|pipewire|pulse|/run/user' /proc/self/mountinfo | grep -v 'X99' >/dev/null 2>&1; then bad "host display/session mount in mountinfo"; fi
BR=
if [ "${1:-}" = "--bridge" ]; then BR=$2; SOCK=$3; shift 3; fi
[ "${1:-}" = "--" ] && shift
# network: only loopback may exist (the container runs with --network none)
for i in /sys/class/net/*; do [ "$(basename "$i")" = lo ] || bad "network interface $(basename "$i") present"; done
if [ -n "$BR" ]; then
  node /opt/isoinside/sockbridge.js 127.0.0.1 "$BR" "$SOCK" &
  i=0; while [ $i -lt 50 ]; do node -e "require('net').connect(+process.argv[1],'127.0.0.1').on('connect',()=>process.exit(0)).on('error',()=>process.exit(1))" "$BR" && break; sleep 0.1; i=$((i+1)); done
fi
exec "$@"
