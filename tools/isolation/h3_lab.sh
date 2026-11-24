#!/usr/bin/env bash
# h3_lab.sh - rootless HTTP/3 (QUIC) lab: helix-server and ONE client container on a private INTERNAL podman network.
# HTTP/3 is UDP: the unix-socket bridge of run_container.sh cannot carry it, so this is the sanctioned second topology
# (docs/test-isolation.md section 8). Guarantees: network is `podman network create --internal` (no route to the host, the LAN or the
# internet; nothing published, no host port, no --net=host); all caps dropped; no-new-privileges; memory/pids caps; names hrdiso-h3-*;
# bounded lifetime (podman --timeout); no host display/session/env; no secret of the operator. Test CA + cert are generated per run.
# usage: h3_lab.sh up   <state-dir> <helix-server-bin>
#        h3_lab.sh run  <state-dir> <name> [-r SRC:DST]... [-w SRC:DST]... [-e K=V]... -- cmd...    (client container, attached, output to stdout)
#        h3_lab.sh logs <state-dir>      (server container logs)
#        h3_lab.sh down <state-dir>
# The server is reachable from clients ONLY as https://helix-server:8443 (TCP h2/h1.1 + UDP h3). CA pem: /lab/ca.pem inside clients.
set -eu
HERE=$(cd "$(dirname "$0")" && pwd); IMG=${H3_LAB_IMAGE:-localhost/helix-gui-test:1}; LIFE=${H3_LAB_LIFE:-1800}
cmd=${1:?}; ST=${2:?}; case $ST in /*) ;; *) echo "state dir must be absolute" >&2; exit 64;; esac
RID_FILE=$ST/rid
flags() { echo --rm --timeout "$LIFE" --cap-drop ALL --security-opt no-new-privileges --userns=keep-id --pids-limit 512 --oom-score-adj 500 --ulimit fsize=2147483648:2147483648 --label org.helix.isolation=h3-lab; }
case $cmd in
up)
  BIN=${3:?}; mkdir -p "$ST"; chmod 700 "$ST"; podman image exists "$IMG" || { echo "image $IMG missing" >&2; exit 66; }
  [ -f "$BIN" ] || { echo "no server binary $BIN" >&2; exit 64; }
  RID=$(head -c4 /dev/urandom | od -An -tx1 | tr -d ' \n'); echo "$RID" > "$RID_FILE"
  mkdir -p "$ST/lab"; cd "$ST/lab"
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -keyout ca.key -out ca.pem -days 2 -subj "/CN=helix-h3-lab-ca" -addext "basicConstraints=critical,CA:TRUE" >/dev/null 2>&1
  openssl req -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 -nodes -keyout srv.key -out srv.csr -subj "/CN=helix-server" >/dev/null 2>&1
  printf 'subjectAltName=DNS:helix-server\nbasicConstraints=CA:FALSE\nextendedKeyUsage=serverAuth\n' > ext.cnf
  openssl x509 -req -in srv.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out srv.pem -days 2 -extfile ext.cnf >/dev/null 2>&1
  rm -f ca.key srv.csr ca.srl; chmod 644 ca.pem srv.pem; chmod 600 srv.key
  cp "$BIN" "$ST/lab/helix-server"; chmod 755 "$ST/lab/helix-server"
  podman network create --internal "hrdiso-h3-net-$RID" >/dev/null
  podman run -d $(flags) --name "hrdiso-h3-srv-$RID" --network "hrdiso-h3-net-$RID" --network-alias helix-server --memory 1g --memory-swap 1g --cpus 2 \
    -v "$ST/lab:/lab:ro" -e HELIX_ADDRESS=0.0.0.0:8443 -e HELIX_TLS_CERT=/lab/srv.pem -e HELIX_TLS_KEY=/lab/srv.key -e HELIX_ALLOW_INSECURE_ADMIN=true \
    -e HELIX_HTTP3=true -e HELIX_ACCESS_LOG=true -e HELIX_COVER_CACHE=/tmp/covers --tmpfs /tmp:rw,size=64m --entrypoint /lab/helix-server "$IMG" >/dev/null
  for _ in $(seq 1 50); do podman logs "hrdiso-h3-srv-$RID" 2>&1 | grep -q "HTTP/3 listener starting" && break; sleep 0.2; done
  podman logs "hrdiso-h3-srv-$RID" 2>&1 | grep -q "HTTP/3 listener starting" || { echo "server did not start" >&2; podman logs "hrdiso-h3-srv-$RID" >&2; exit 1; }
  echo "lab up rid=$RID"
  ;;
run)
  NAME=${3:?}; shift 3; RID=$(cat "$RID_FILE"); B=(); E=(); 
  while getopts "r:w:e:" o; do case $o in
    r) IFS=: read -r s d <<<"$OPTARG"; real=$(python3 -I "$HERE/bindcheck.py" ro "$s" "$d") || exit 64; B+=(-v "$real:$d:ro");;
    w) IFS=: read -r s d <<<"$OPTARG"; real=$(python3 -I "$HERE/bindcheck.py" rw "$s" "$d") || exit 64; B+=(-v "$real:$d:rw");;
    e) b=$OPTARG; n=${b%%=*}; N=${n^^}
       case $b in *=*) ;; *) echo "h3_lab: env must be explicit NAME=VALUE: $b" >&2; exit 64;; esac
       case $n in ''|[0-9]*|*[!A-Za-z0-9_]*) echo "h3_lab: invalid env name: $n" >&2; exit 64;; esac
       case $N in DISPLAY*|*_DISPLAY|WAYLAND*|XAUTHORITY*|ICEAUTHORITY*|*DBUS*|XDG_*|SSH_AUTH_SOCK|SSH_*|PULSE*|PIPEWIRE*|GNOME_*|KDE_*|GTK_*|QT_QPA*|DESKTOP_SESSION|SESSION_MANAGER|GPG_AGENT*|GDK_*) echo "h3_lab: refusing host display/session env: $n" >&2; exit 64;; esac  # same denylist as run_container.sh
       E+=(-e "$b");; *) exit 2;; esac; done; shift $((OPTIND-1)); [ "${1:-}" = -- ] && shift
  case $NAME in hrdiso-h3-*) ;; *) echo "name must start hrdiso-h3-" >&2; exit 64;; esac
  exec podman run $(flags) --name "$NAME-$RID" --network "hrdiso-h3-net-$RID" --memory 3g --memory-swap 3g --cpus 2 --shm-size 512m --tmpfs /tmp:rw,size=512m \
    -v "$ST/lab/ca.pem:/lab/ca.pem:ro" "${B[@]}" "${E[@]}" --entrypoint "" "$IMG" "$@"
  ;;
logs) podman logs "hrdiso-h3-srv-$(cat "$RID_FILE")" 2>&1 ;;
down)
  [ -f "$RID_FILE" ] || exit 0; RID=$(cat "$RID_FILE")
  podman rm -f "hrdiso-h3-srv-$RID" >/dev/null 2>&1 || true
  for c in $(podman ps -aq --filter "name=-$RID\$" --filter label=org.helix.isolation=h3-lab); do podman rm -f "$c" >/dev/null 2>&1 || true; done
  podman network rm "hrdiso-h3-net-$RID" >/dev/null 2>&1 || true
  ;;
*) echo "usage: see header" >&2; exit 2;;
esac
