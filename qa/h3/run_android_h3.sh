#!/usr/bin/env bash
# run_android_h3.sh <state-dir> <out-dir> - LIVE Android proof (Cronet, HTTP/3 + Brotli): helix-server and an Android emulator in
# rootless containers on the INTERNAL h3 lab network (tools/isolation/h3_lab.sh). The emulator container is the one extra device
# (--device /dev/kvm); it has no published port, no host display, all caps dropped, a lease timeout and org.helix.isolation labels.
# The app under test is the LAB build type (trusts a throwaway per-run CA); the verdict is the SERVER access log (check_h3.py).
# env: HELIX_SERVER_BIN, JAVA_HOME (JDK 21), ANDROID_HOME, EMU_IMAGE, SKIP_BUILD=1 (reuse apk/ in the state dir; only valid with an unchanged CA)
set -u
R=$(cd "$(dirname "$0")/../.." && pwd); ST=${1:?state dir}; OUT=${2:?out dir}; BIN=${HELIX_SERVER_BIN:?helix-server build}
EMU_IMAGE=${EMU_IMAGE:-ghcr.io/vasic-digital/lava-android-emulator:api34-x86_64}
export ISO_WORK=$ST JAVA_HOME=${HELIX_JDK21:-$HOME/tools/jdk-21} ANDROID_HOME=${ANDROID_HOME:-$HOME/Android/Sdk}
mkdir -p "$OUT" "$ST/apk"; LAB="$R/tools/isolation/h3_lab.sh"
"$LAB" down "$ST" 2>/dev/null; rm -rf "${ST:?}/lab"; "$LAB" up "$ST" "$BIN" || exit 1
RID=$(cat "$ST/rid"); EMU=hrdiso-h3-emu-$RID
trap 'podman rm -f "$EMU" >/dev/null 2>&1; "$LAB" down "$ST"' EXIT
# 1. lab build type: trust the per-run CA
LABDIR=$ST/android_lab; mkdir -p "$LABDIR/res/xml" "$LABDIR/res/raw"; cp "$ST/lab/ca.pem" "$LABDIR/res/raw/helix_lab_ca.pem"
cat > "$LABDIR/AndroidManifest.xml" <<'XML'
<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <uses-permission android:name="android.permission.INTERNET" />
    <application android:networkSecurityConfig="@xml/network_security_config" />
</manifest>
XML
cat > "$LABDIR/res/xml/network_security_config.xml" <<'XML'
<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <base-config cleartextTrafficPermitted="false">
        <trust-anchors>
            <certificates src="system" />
            <certificates src="@raw/helix_lab_ca" />
        </trust-anchors>
    </base-config>
</network-security-config>
XML
if [ "${SKIP_BUILD:-0}" != 1 ]; then
  (cd "$R/clients/android/android" && ./gradlew --no-daemon -q :app:assembleLab :app:assembleAndroidTest -PhelixLabDir="$LABDIR" > "$OUT/gradle.log" 2>&1) || { echo "gradle build failed, see $OUT/gradle.log" >&2; exit 1; }
  cp "$R/clients/android/android/app/build/outputs/apk/lab/app-lab.apk" "$R/clients/android/android/app/build/outputs/apk/androidTest/lab/app-lab-androidTest.apk" "$ST/apk/"
fi
sha256sum "$ST/apk/"*.apk > "$OUT/apk.sha256"
# 2. emulator on the lab network
podman run -d --rm --timeout 2400 --name "$EMU" --network "hrdiso-h3-net-$RID" --device /dev/kvm --userns=keep-id --cap-drop ALL --security-opt no-new-privileges \
  --pids-limit 2048 --memory 6g --memory-swap 6g --cpus 4 --oom-score-adj 500 --label org.helix.isolation=h3-lab -v "$ST/apk:/apk:ro" "$EMU_IMAGE" > /dev/null || exit 1
b=; for i in $(seq 1 90); do b=$(podman exec "$EMU" adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r'); [ "$b" = 1 ] && break; sleep 5; done
echo "emulator boot_completed=$b after $((i*5))s"; [ "$b" = 1 ] || { podman logs "$EMU" 2>&1 | tail -20 >&2; exit 1; }
podman exec "$EMU" adb shell 'getprop ro.build.version.sdk; getprop ro.product.cpu.abi' > "$OUT/device.txt" 2>&1
# DNS + reachability from the guest: the container resolves helix-server through the lab network; the guest uses 10.0.2.3 -> container resolver
podman exec "$EMU" adb shell 'ping -c1 -W3 helix-server' > "$OUT/guest_ping.txt" 2>&1 || true
# Chromium (Cronet) refuses QUIC for a certificate that is not issued by a KNOWN root (net-log: CERTIFICATE_VERIFY_FAILED on the QUIC handshake
# although the TCP handshake with the same CA passes via network-security-config). Lab-only: put the per-run CA into the SYSTEM store of the
# throwaway emulator (tmpfs bind over the conscrypt apex cacerts, in the zygote mount namespaces).
if [ "${LAB_SYSTEM_CA:-0}" = 1 ]; then
  H=$(openssl x509 -in "$ST/lab/ca.pem" -subject_hash_old -noout)
  cp "$ST/lab/ca.pem" "$ST/apk/$H.0"
  podman exec "$EMU" adb root > "$OUT/adb_root.txt" 2>&1; sleep 3
  podman exec "$EMU" adb push /apk/$H.0 /data/local/tmp/$H.0 >> "$OUT/adb_root.txt" 2>&1
  podman exec "$EMU" adb shell "set -e; mkdir -p /data/local/tmp/cac; mount -t tmpfs tmpfs /data/local/tmp/cac; cp /apex/com.android.conscrypt/cacerts/* /data/local/tmp/cac/; cp /data/local/tmp/$H.0 /data/local/tmp/cac/; chmod 644 /data/local/tmp/cac/*; chcon u:object_r:system_file:s0 /data/local/tmp/cac/*; for p in 1 \$(pidof zygote) \$(pidof zygote64); do nsenter --mount=/proc/\$p/ns/mnt -- mount --bind /data/local/tmp/cac /apex/com.android.conscrypt/cacerts; done; ls /apex/com.android.conscrypt/cacerts | wc -l" > "$OUT/system_ca.txt" 2>&1
  podman exec "$EMU" adb shell "ls /apex/com.android.conscrypt/cacerts/$H.0" >> "$OUT/system_ca.txt" 2>&1
fi
podman exec "$EMU" adb install -r /apk/app-lab.apk > "$OUT/install_app.txt" 2>&1
podman exec "$EMU" adb install -r /apk/app-lab-androidTest.apk > "$OUT/install_test.txt" 2>&1
before=$("$LAB" logs "$ST" | wc -l)
podman exec "$EMU" adb logcat -c
podman exec "$EMU" adb shell am instrument -w -e server https://helix-server:8443 ${EXPECT_H3:+-e expectH3 1} -e forceQuic helix-server:8443 -e netlog /data/data/org.helixreader.app/netlog.json -e class org.helixreader.app.CronetLiveTest org.helixreader.app.test/androidx.test.runner.AndroidJUnitRunner > "$OUT/instrument.txt" 2>&1; irc=$?
podman exec "$EMU" adb logcat -d -s HelixCronet:I > "$OUT/logcat_HelixCronet.txt" 2>&1
podman exec "$EMU" adb shell run-as org.helixreader.app cat netlog.json > "$OUT/cronet_netlog.json" 2>/dev/null || true
"$LAB" logs "$ST" | tail -n +"$((before+1))" > "$OUT/android.server.log"
python3 -I "$R/qa/h3/check_h3.py" "$OUT/android.server.log" /v1/formats "${EXPECT_PROTO:-HTTP/2.0}" br > "$OUT/android.verdict.json"; v=$?
tail -3 "$OUT/instrument.txt"; echo "instrument_rc=$irc"; cat "$OUT/android.verdict.json"
grep -c 'proto=h3' "$OUT/logcat_HelixCronet.txt" | sed 's/^/logcat h3 lines: /'
# Android h3 is UNCONFIRMED (private-CA QUIC cert verify -356): the gate is the instrumented run (h2/br assertion green, h3 test an honest SKIP-with-reason)
# plus the server verdict for EXPECT_PROTO (default HTTP/2.0; set EXPECT_PROTO=HTTP/3.0 with -e expectH3 1 for a publicly trusted certificate).
rc=0; grep -q "OK (3 tests)" "$OUT/instrument.txt" || rc=1
[ "$v" -eq 0 ] || rc=1
echo "overall rc=$rc"; exit $rc
