#!/bin/sh
# Runs INSIDE the hrdiso-* container (--network none: only loopback). Starts the Readium reference
# License Server + Status Server as real processes, encrypts a real EPUB with the reference lcpencrypt,
# then runs Helix's core/pkg/lcp interop test against them. Credentials come from the environment
# (HELIX_LCP_INTEROP_ADMIN_USER / _PASS / _PASSPHRASE, set by the host driver). Nothing secret may reach /out:
# lcpencrypt is NOT run with -verbose (that prints the content-encryption-key), and the end of this script scans /out
# for the key marker and for the credential values and fails the run if any is found (review F1 of PR #33).
set -eu
. /work/in/creds.env 2>/dev/null || true
: "${HELIX_LCP_INTEROP_ADMIN_USER:?}" "${HELIX_LCP_INTEROP_ADMIN_PASS:?}" "${HELIX_LCP_INTEROP_PASSPHRASE:?}"
B=/work/bin; IN=/work/in; OUT=/out; H=/tmp/lcphome
ORACLE=${HELIX_LCP_ORACLE_EPUB:-sample.epub}   # plaintext oracle file name under $IN (the wrong-oracle mutant sets lorem.epub)
mkdir -p "$H/db" "$H/config" "$H/files/storage" "$H/www" "$H/tmp" "$OUT"
LCP=http://127.0.0.1:8989; LSD=http://127.0.0.1:8990; WWW=http://127.0.0.1:8991
# htpasswd ({SHA} scheme, understood by the reference servers' auth library)
python3 - <<PY
import base64,hashlib,os
u=os.environ["HELIX_LCP_INTEROP_ADMIN_USER"];p=os.environ["HELIX_LCP_INTEROP_ADMIN_PASS"]
open("$H/config/htpasswd","w").write(u+":{SHA}"+base64.b64encode(hashlib.sha1(p.encode()).digest()).decode()+"\n")
PY
# Operator-owned test PKI (the EDRLab test CA root is not distributed with the reference server: finding F1)
cd "$H/config"
openssl req -x509 -newkey rsa:2048 -nodes -keyout ca.key -out ca.pem -days 30 -subj "/O=Helix interop test/CN=Helix interop test root" 2>/dev/null
openssl req -newkey rsa:2048 -nodes -keyout privkey.pem -out prov.csr -subj "/O=Helix interop test/CN=Helix interop provider" 2>/dev/null
openssl x509 -req -in prov.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out cert.pem -days 30 2>/dev/null
cp ca.pem "$OUT/root_ca_public.txt"   # public test root only (.txt: the seal allow-list takes no .pem)
# the shipped EDRLab test provider cert (kept to demonstrate the chain gap)
cp "$IN/cert-edrlab-test.pem" "$H/config/cert-edrlab-test.pem"
cat > "$H/config/config.yaml" <<YAML
profile: "basic"
lcp:
    public_base_url: "$LCP"
    port: 8989
    database: "sqlite3://file:$H/db/lcp.sqlite?cache=shared&mode=rwc"
    auth_file: "$H/config/htpasswd"
lcp_update_auth:
    username: "$HELIX_LCP_INTEROP_ADMIN_USER"
    password: "$HELIX_LCP_INTEROP_ADMIN_PASS"
lsd:
    public_base_url: "$LSD"
    port: 8990
    database: "sqlite3://$H/db/lsd.sqlite?cache=shared&mode=rwc"
    auth_file: "$H/config/htpasswd"
    license_link_url: "$LCP/licenses/{license_id}"
lsd_notify_auth:
    username: "$HELIX_LCP_INTEROP_ADMIN_USER"
    password: "$HELIX_LCP_INTEROP_ADMIN_PASS"
license_status:
    register: true
    renew: false
    return: true
certificate:
    cert: "$H/config/cert.pem"
    private_key: "$H/config/privkey.pem"
license:
    links:
        status: "$LSD/licenses/{license_id}/status"
        hint: "$WWW/hint.html"
storage:
    filesystem:
        directory: "$H/files/storage"
YAML
echo "<html><body>interop hint</body></html>" > "$H/www/hint.html"
export READIUM_LCPSERVER_CONFIG="$H/config/config.yaml" READIUM_LSDSERVER_CONFIG="$H/config/config.yaml"
(cd "$H/www" && exec python3 -m http.server 8991 --bind 127.0.0.1 >"$OUT/www.log" 2>&1) &
(cd "$H" && exec "$B/lcpserver" >"$OUT/lcpserver.log" 2>&1) &
(cd "$H" && exec "$B/lsdserver" >"$OUT/lsdserver.log" 2>&1) &
for _ in $(seq 1 60); do
  curl -s -o /dev/null "$LCP/" && curl -s -o /dev/null "$LSD/" && break; sleep 1
done
{ echo "lcpserver: $(curl -s -o /dev/null -w '%{http_code}' $LCP/)"; echo "lsdserver: $(curl -s -o /dev/null -w '%{http_code}' $LSD/)"; } | tee "$OUT/provider_up.txt"
# Encrypt a real EPUB with the reference tool; it registers the content with the License Server.
CID=$(python3 -c 'import uuid;print(uuid.uuid4())')
"$B/lcpencrypt" -input "$IN/sample.epub" -contentid "$CID" -storage "$H/www" -url "$WWW" -filename interop.epub \
  -lcpsv "$LCP" -login "$HELIX_LCP_INTEROP_ADMIN_USER" -password "$HELIX_LCP_INTEROP_ADMIN_PASS" >"$OUT/lcpencrypt.log" 2>&1 || { echo "lcpencrypt FAILED"; tail -20 "$OUT/lcpencrypt.log"; }
ls -l "$H/www" | tee "$OUT/www_listing.txt"
export HELIX_LCP_INTEROP_LCP_URL="$LCP" HELIX_LCP_INTEROP_CONTENT_ID="$CID" \
  HELIX_LCP_INTEROP_ROOT_PEM="$OUT/root_ca_public.txt" HELIX_LCP_INTEROP_SOURCE_EPUB="$IN/$ORACLE" \
  HELIX_LCP_INTEROP_FOREIGN_PEM="$IN/cert-edrlab-test.pem" \
  HELIX_LCP_INTEROP_EVIDENCE="$OUT/interop_steps.json"
set +e
"$B/lcp.test" -test.run 'TestInteropReadiumLCPServer' -test.v >"$OUT/helix_interop.log" 2>&1
rc=$?
echo "helix_interop_exit=$rc" | tee "$OUT/helix_interop_exit.txt"
sed -e "s/$HELIX_LCP_INTEROP_ADMIN_PASS/<redacted>/g" "$OUT/helix_interop.log"
# Secret scan of everything that will be published (F1 guard). grep -F with the values from a pattern file, never printed.
SCAN=$(mktemp); printf '%s\n' "$HELIX_LCP_INTEROP_ADMIN_PASS" "$HELIX_LCP_INTEROP_PASSPHRASE" > "$SCAN"
hits=0
if grep -rIil 'content-encryption-key' "$OUT" >/dev/null 2>&1; then echo "SECRET-SCAN FAIL: content-encryption-key marker present in /out"; hits=1; fi
if grep -rIl -F -f "$SCAN" "$OUT" >/dev/null 2>&1; then echo "SECRET-SCAN FAIL: a credential value is present in /out"; hits=1; fi
rm -f "$SCAN"
if [ "$hits" = 1 ]; then echo "secret_scan=FAIL" > "$OUT/secret_scan.txt"; [ "$rc" -eq 0 ] && rc=3; else echo "secret_scan=PASS (key marker absent, credential values absent)" > "$OUT/secret_scan.txt"; fi
exit $rc
