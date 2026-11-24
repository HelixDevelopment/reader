#!/bin/sh
# isolation: runs-inside-container
# Drives KOReader's REAL GUI on the container-private Xvfb (:99, no window manager; the SDL window sits at
# 337,0, 600x800). Coordinates were calibrated from screenshots of KOReader 2026.07-1.
# usage (via `podman exec`): ko_drive.sh start <P> [path] | tap LX LY [wait] (window-local) | key K | pages N | close
#   | opds_open <P> | opds_download_read | opds_comic_stream | shot NAME | pixel X Y | docpercent <P>
export DISPLAY=:99
cmd=$1; shift
# tap takes WINDOW-LOCAL coordinates; the window origin is read from X (no WM => SDL places it, undecorated).
# The origin is measured once after start (the first KOReader window) and cached; later popups never re-measure.
origin() { if [ ! -s /tmp/ko_origin ]; then W=$(xdotool search --name KOReader | head -1); eval "$(xdotool getwindowgeometry --shell "$W")"; echo "$X $Y" > /tmp/ko_origin; fi; read OX OY < /tmp/ko_origin; }
tap() { origin; xdotool mousemove $((OX+$1)) $((OY+$2)); sleep 0.4; xdotool click 1; sleep "${3:-2.5}"; }
case $cmd in
  start) rm -f /tmp/ko_origin; P=$1; target=${2:-/prof/$P/Books}
    export HOME=/prof/$P XDG_CONFIG_HOME=/prof/$P/.config XDG_DATA_HOME=/prof/$P/.local/share XDG_CACHE_HOME=/prof/$P/.cache KO_MULTIUSER=1 SDL_VIDEODRIVER=x11 LC_ALL=en_US.UTF-8
    cd /opt/koreader && { ./luajit reader.lua "$target" > /prof/$P/ko.log 2>&1 & }
    sleep 15; xdotool mousemove 640 400; sleep 1; xdotool mousemove 650 410; sleep 2; origin ;;
  tap) tap "$@" ;;
  key) xdotool key "$1"; sleep "${2:-2}" ;;
  pages) i=0; while [ $i -lt "$1" ]; do xdotool key Next; sleep 2; i=$((i+1)); done ;;
  close) pkill -TERM luajit; i=0; while pgrep luajit >/dev/null && [ $i -lt 20 ]; do sleep 1; i=$((i+1)); done; pgrep luajit >/dev/null && echo "still-running" || echo "exited" ;;
  opds_open) tap 301 50; tap 258 27; tap 108 537; tap 161 149 4; tap 161 97 3 ;;   # menu > search tab > OPDS catalog > Helix > All publications
  opds_download_read) tap 221 245 3; tap 299 472 5; tap 411 486 7 ;;              # Glass Meridian > epub download > Read now
  opds_comic_stream) tap 161 97 3; tap 166 493 6 ;;                                 # comic > Page stream
  shot) xdotool mousemove 700 790; sleep 1; import -window root /out/$1.png ;;
  pixel) origin; import -window root /tmp/_p.png; convert /tmp/_p.png -crop 1x1+$((OX+$1))+$((OY+$2)) -format '%[pixel:p{0,0}]' info: ;;
  docpercent) grep -rhoE "\[\"percent_finished\"\] = [0-9.]+" /prof/$1/Books 2>/dev/null | head -1 ;;
esac
