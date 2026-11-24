#!/usr/bin/env bash
# run_android_emulator.sh - canonical containerised Android emulator (headless; never a host window).
# Recipe taken from evidence/clients/android/run-20261123T211340Z/RUN.txt (HRD-022 run by another agent). UNVERIFIED here:
# this script itself was not executed by the isolation work (the image's entrypoint env knobs were not re-probed).
# usage: run_android_emulator.sh start|stop|adb [ADB_PORT=5590]
# Own container only (name hrdiso-android); never touch other projects' containers.
set -eu
N=hrdiso-android; PORT=${2:-5590}; IMG=ghcr.io/vasic-digital/lava-android-emulator:api34-x86_64
case $1 in
  start)
    # --device /dev/kvm (ACL'd to the user) + --userns=keep-id: KVM acceleration without root, no display of any kind.
    # AVD 'default' (API 34 google_apis x86_64) runs with: emulator -avd default -no-window -no-audio -no-boot-anim
    exec podman run -d --name $N --memory 5g --memory-swap 5g --cpus 4 --pids-limit 4096 --oom-score-adj 500 \
      --device /dev/kvm --userns=keep-id --cap-drop ALL --security-opt no-new-privileges \
      -p 127.0.0.1:$PORT:5555 "$IMG" ;;
  adb) adb connect 127.0.0.1:$PORT ;;   # then: adb -s 127.0.0.1:$PORT install -r <apk>; use a logging proxy for server access logs
  stop) podman rm -f $N ;;
esac
