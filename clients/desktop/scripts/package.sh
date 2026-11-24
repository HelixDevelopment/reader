#!/usr/bin/env bash
# Builds the unpacked Linux x64 app, a tar.gz and a .deb into clients/desktop/out.
# Needs: clients/web built (npm run build), npm ci in clients/desktop (electron + @electron/packager),
# electron zip in ~/.cache/electron (node node_modules/electron/install.js), dpkg-deb. No network at package time.
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION=$(node -p "require('./package.json').version")
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
mkdir -p "$STAGE/app"
cp -r main.js preload.js package.json lib renderer "$STAGE/app/"
cp -r ../web/dist "$STAGE/app/web"
# the staged package.json carries no dependencies so nothing from node_modules is bundled
node -e "const p=require('$STAGE/app/package.json');delete p.devDependencies;delete p.scripts;require('fs').writeFileSync('$STAGE/app/package.json',JSON.stringify(p,null,2))"
rm -rf out && mkdir out
npx --no-install electron-packager "$STAGE/app" helix-reader --platform=linux --arch=x64 --out=out --overwrite \
  --electron-version="$(node -p "require('electron/package.json').version")" --asar --no-prune >/dev/null
APPDIR=out/helix-reader-linux-x64
test -x "$APPDIR/helix-reader"
tar -C out -czf "out/helix-reader-$VERSION-linux-x64.tar.gz" helix-reader-linux-x64
# .deb: /opt/helix-reader + /usr/bin symlink + desktop entry
D=$(mktemp -d); trap 'rm -rf "$STAGE" "$D"' EXIT
mkdir -p "$D/opt" "$D/usr/bin" "$D/usr/share/applications" "$D/DEBIAN"
cp -r "$APPDIR" "$D/opt/helix-reader"
ln -s /opt/helix-reader/helix-reader "$D/usr/bin/helix-reader"
cat > "$D/usr/share/applications/helix-reader.desktop" <<EOF
[Desktop Entry]
Name=Helix Reader
Exec=/opt/helix-reader/helix-reader %U
Terminal=false
Type=Application
Categories=Office;Viewer;
EOF
SIZE=$(du -sk "$D/opt" | cut -f1)
cat > "$D/DEBIAN/control" <<EOF
Package: helix-reader
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Installed-Size: $SIZE
Depends: libgtk-3-0, libnss3, libxss1, libasound2t64 | libasound2, libgbm1
Maintainer: Helix Reader <noreply@invalid>
Description: Helix Reader desktop client
 Electron shell around the Helix Reader web client.
EOF
chmod 0755 "$D/DEBIAN"
dpkg-deb --root-owner-group -Zxz -b "$D" "out/helix-reader_${VERSION}_amd64.deb" >/dev/null
( cd out && sha256sum helix-reader_${VERSION}_amd64.deb helix-reader-$VERSION-linux-x64.tar.gz )
