#!/usr/bin/env bash
# hrd026_providers.sh - loopback rclone-served providers registered as helix sources; scan + sha256 + Range proof.
# usage: hrd026_providers.sh <state-dir> <fixture-dir> <out-dir>   (server must be up: hrd026_server.sh start)
# Providers: native WebDAV (rclone serve webdav), rclone-connector->SFTP (rclone serve sftp), native OPDS-less control: local.
# s3: rclone v1.60.1 has no `serve s3` (added in 1.65) -> reported as not testable here.
set -u
# Script hygiene (constitution 11.4.287(C)): every positional argument and every path that is later deleted is guarded.
ST=${1:?state-dir}; FIX=${2:?fixture-dir}; OUT=${3:?out-dir}
case $ST in /*) ;; *) echo "state-dir must be absolute" >&2; exit 2;; esac
case $FIX in /*) ;; *) echo "fixture-dir must be absolute" >&2; exit 2;; esac
case $OUT in /*) ;; *) echo "out-dir must be absolute" >&2; exit 2;; esac
[ -f "$ST/env" ] || { echo "missing $ST/env (run hrd026_server.sh start first)" >&2; exit 2; }
. "$ST/env"; B=http://127.0.0.1:$PORT
mkdir -p "$OUT"
TMPROOT=${TMPDIR:-/tmp}
W=$(mktemp -d) || exit 2
case $W in "$TMPROOT"/?*) ;; *) echo "scratch dir $W is not under $TMPROOT" >&2; exit 2;; esac
PIDS=()
# the admin header lives in a 0600 file, not in argv
umask 077; printf 'Authorization: Bearer %s\n' "$ADMIN" > "${W:?}/hdr"
cleanup() { for p in "${PIDS[@]}"; do kill "$p" 2>/dev/null; done; rm -rf -- "${W:?}"; }
trap cleanup EXIT
# HOME is NOT reassigned: rclone gets an explicit config file below.
PU=prov$RANDOM; PP=$(head -c12 /dev/urandom | base64 | tr -dc A-Za-z0-9)
: "${PP:?password generation failed}"
export RCLONE_CONFIG="$W/rclone.conf"; : > "$RCLONE_CONFIG"; chmod 600 "$RCLONE_CONFIG"
rclone serve webdav --addr 127.0.0.1:38101 --user "$PU" --pass "$PP" "$FIX" >"$W/webdav.log" 2>&1 & PIDS+=($!)
rclone serve sftp   --addr 127.0.0.1:38102 --user "$PU" --pass "$PP" "$FIX" >"$W/sftp.log" 2>&1 & PIDS+=($!)
sleep 2
OBS=$(rclone obscure "$PP")
printf '[sftpremote]\ntype = sftp\nhost = 127.0.0.1\nport = 38102\nuser = %s\npass = %s\n' "$PU" "$OBS" > "$W/rclone.conf"; chmod 600 "$W/rclone.conf"
reg() { printf '%s' "$1" > "$W/reg.json"; curl -s -o "$W/reg.out" -w '%{http_code}' -XPOST "$B/v1/sources" -H "@$W/hdr" -d "@$W/reg.json"; }
scan() { id=$(curl -s -XPOST "$B/v1/scans" -H "@$W/hdr" -d "{\"source_id\":\"$1\",\"prefix\":\"\"}" | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
  for _ in $(seq 1 60); do s=$(curl -s "$B/v1/scans/$id" -H "@$W/hdr"); echo "$s" | grep -qE '"state":"(succeeded|failed)"' && break; sleep 0.5; done; echo "$s"; }
{
echo "# HRD-026 loopback providers $(date -u +%FT%TZ); rclone $(rclone version | head -1)"
echo "# fixture sha256 (origin):"; (cd "$FIX" && sha256sum -- *)
echo "# REG webdav: http=$(reg "{\"id\":\"p-webdav\",\"protocol\":\"webdav\",\"endpoint\":\"http://127.0.0.1:38101\",\"username\":\"$PU\",\"password\":\"$PP\",\"allow_private_network\":true}")"
echo "# SCAN webdav: $(scan p-webdav)"
echo "# REG sftp(rclone): http=$(reg "{\"id\":\"p-sftp\",\"protocol\":\"sftp\",\"endpoint\":\"sftpremote:\",\"rclone_config\":\"$W/rclone.conf\"}")"
echo "# SCAN sftp: $(scan p-sftp)"
} > "$OUT/providers_scan.txt" 2>&1
sed -i "s/$PP/<redacted>/g; s/$PU/<provuser>/g" "$OUT/providers_scan.txt"
# Verify each file per source: byte identity + Range
HRD026_U=$OPDS_USER HRD026_P=$OPDS_PASS python3 -I - "$B" "$FIX" "$OUT/providers_verify.txt" <<'PY'
import sys,json,hashlib,base64,urllib.request,os
B,fix,out=sys.argv[1:4]
u,p=os.environ["HRD026_U"],os.environ["HRD026_P"]
hdr={"Authorization":"Basic "+base64.b64encode(f"{u}:{p}".encode()).decode()}
def get(path,extra=None):
    r=urllib.request.Request(B+path,headers={**hdr,**(extra or {})})
    with urllib.request.urlopen(r) as f: return f.status,dict(f.headers),f.read()
lines=[]
works=json.loads(get("/v1/works?limit=200")[2])
items=works.get("items") or works.get("works") or works
files=[]
for w in items:
    for e in w.get("editions",[]) or []:
        for f in e.get("files",[]) or []: files.append(f)
if not files:
    lines.append("works json keys: "+str(list(works)[:10])); lines.append(json.dumps(works)[:600])
origin={hashlib.sha256(open(os.path.join(fix,n),"rb").read()).hexdigest():n for n in os.listdir(fix)}
for f in sorted(files,key=lambda f:(f["source_id"],f["sha256"])):
    sid=f["source_id"]; fid=f["id"]
    path=origin.get(f["sha256"],"?"); ref=open(os.path.join(fix,path),"rb").read()
    st,h,body=get(f"/v1/files/{fid}/content")
    ok=hashlib.sha256(body).hexdigest()==hashlib.sha256(ref).hexdigest()
    a,b=1000,2047
    st2,h2,part=get(f"/v1/files/{fid}/content",{"Range":f"bytes={a}-{b}"})
    rok= st2==206 and part==ref[a:b+1]
    fb = (st2==200 and part==ref)  # documented fallback: remote connectors have no revalidation -> full 200, no Accept-Ranges
    lines.append(f"{sid:10} {path:22} size={len(body):8} sha256={hashlib.sha256(body).hexdigest()[:16]} full_identical={ok} range_status={st2} range206_bytes_identical={rok} full200_fallback_identical={fb} accept-ranges={h.get('Accept-Ranges')} etag={h.get('ETag') is not None} content-range={h2.get('Content-Range')}")
open(out,"w").write("\n".join(lines)+"\n"); print("\n".join(lines))
PY
