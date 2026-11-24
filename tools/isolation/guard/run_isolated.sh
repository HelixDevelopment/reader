#!/usr/bin/env bash
# run_isolated.sh - run a command with a SANITISED environment (env -i + allowlist).
# usage: run_isolated.sh [--require-display] [--allow VAR]... -- cmd args...
# Refuses (exit 64) if the caller demands a display (--require-display) or the command is a
# host-GUI driver. Host DISPLAY/WAYLAND_DISPLAY/XAUTHORITY/DBUS_*/XDG_RUNTIME_DIR are never forwarded.  # isolation: denylist
set -u
ALLOW="PATH HOME LANG LC_ALL TZ HELIX_* QA_* HQA_* CI_NONE"
while [ $# -gt 0 ]; do
  case $1 in
    --require-display) echo "run_isolated: refusing - a host display is required; GUI tests must run in a container (docs/test-isolation.md)" >&2; exit 64 ;;
    --allow) ALLOW="$ALLOW $2"; shift 2 ;;
    --) shift; break ;;
    *) echo "usage: run_isolated.sh [--require-display] [--allow VAR]... -- cmd..." >&2; exit 2 ;;
  esac
done
[ $# -gt 0 ] || { echo "run_isolated: no command" >&2; exit 2; }
case $(basename "$1") in
  xdotool|ydotool|wtype|scrcpy|grim|gnome-screenshot|xdg-open|wl-copy|xclip|xsel)  # isolation: denylist
    echo "run_isolated: refusing host-GUI command '$1'" >&2; exit 64 ;;
esac
args=()
for pat in $ALLOW; do
  while IFS='=' read -r k v; do
    case $k in
      DISPLAY|WAYLAND_DISPLAY|XAUTHORITY|DBUS_*|XDG_RUNTIME_DIR|XDG_SESSION_*|SSH_AUTH_SOCK) continue ;;  # isolation: denylist
    esac
    # shellcheck disable=SC2254
    case $k in $pat) args+=("$k=$v") ;; esac
  done < <(env)
done
# RUNTIME assertion on the environment actually handed to the child (defence against an allow-list pattern that is too broad)
for a in "${args[@]}"; do
  case ${a%%=*} in
    DISPLAY|WAYLAND*|XAUTHORITY|DBUS_*|XDG_RUNTIME_DIR|XDG_SESSION_*|SSH_AUTH_SOCK|GNOME_*|KDE_*) echo "run_isolated: host session variable ${a%%=*} would leak; refusing" >&2; exit 65 ;;  # isolation: denylist
  esac
done
exec env -i "${args[@]}" "$@"
