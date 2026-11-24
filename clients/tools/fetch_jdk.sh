#!/usr/bin/env bash
# Download + sha256-verify a portable Temurin JDK 21 into ~/tools/jdk-21 (user dir, no root). Aborts on mismatch.
set -euo pipefail
DEST="${1:-$HOME/tools/jdk-21}"; WORK="$HOME/tools/dl"; mkdir -p "$WORK"
META=$(curl -fsS "https://api.adoptium.net/v3/assets/latest/21/hotspot?architecture=x64&image_type=jdk&os=linux&vendor=eclipse")
read -r NAME SUM < <(python3 -I -c 'import json,sys;p=json.load(sys.stdin)[0]["binary"]["package"];print(p["name"],p["checksum"])' <<<"$META")
curl -fSL -C - -o "$WORK/$NAME" "https://api.adoptium.net/v3/binary/latest/21/ga/linux/x64/jdk/hotspot/normal/eclipse"
echo "$SUM  $WORK/$NAME" | sha256sum -c - || { echo "SHA256 MISMATCH - aborting"; exit 1; }
mkdir -p "$DEST" && tar -xzf "$WORK/$NAME" -C "$DEST" --strip-components=1
"$DEST/bin/javac" -version
