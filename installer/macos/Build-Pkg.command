#!/bin/bash
# Run on macOS with Apple's Command Line Tools. Produces a per-user package;
# browser registration and tool setup are completed by Install.command.
set -euo pipefail
cd -- "$(dirname -- "$0")"
bundle="$PWD"
if [ "$(uname -s)" != Darwin ]; then echo 'pkgbuild / productbuild 需要 macOS。'; exit 1; fi
command -v pkgbuild >/dev/null
command -v productbuild >/dev/null
stage="$(mktemp -d "${TMPDIR:-/tmp}/videobridge-pkg.XXXXXX")"
trap 'rm -rf -- "$stage"' EXIT
mkdir -p "$stage/root/Library/Application Support/VideoBridge/Installer"
/usr/bin/ditto "$bundle" "$stage/root/Library/Application Support/VideoBridge/Installer"
pkgbuild --root "$stage/root" --identifier local.videobridge.mac --version __VERSION__ --install-location / "$stage/component.pkg"
cat > "$stage/Distribution.xml" <<'XML'
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
  <title>VideoBridge __VERSION__</title>
  <options customize="never" require-scripts="false" hostArchitectures="arm64,x86_64"/>
  <domains enable_anywhere="false" enable_currentUserHome="true" enable_localSystem="false"/>
  <allowed-os-versions><os-version min="12.0"/></allowed-os-versions>
  <conclusion file="conclusion.html"/>
  <choices-outline><line choice="default"/></choices-outline>
  <choice id="default" title="VideoBridge"><pkg-ref id="local.videobridge.mac"/></choice>
  <pkg-ref id="local.videobridge.mac" version="__VERSION__">component.pkg</pkg-ref>
</installer-gui-script>
XML
cat > "$stage/conclusion.html" <<'HTML'
<!doctype html><meta charset="utf-8"><h2>文件已安装</h2>
<p>请在 Finder 按 Command + Shift + G，打开下面的目录：</p>
<p>~/Library/Application Support/VideoBridge/Installer</p>
<p>双击 Install.command 完成本地桥接注册与依赖安装，再按 README-Mac.md 导入浏览器扩展。</p>
HTML
productbuild --distribution "$stage/Distribution.xml" --package-path "$stage" --resources "$stage" "$bundle/../VideoBridge-__VERSION__-macOS.pkg"
printf '\n生成的是未签名 PKG。公开分发前需 Developer ID Installer 签名、公证和 Mac 实机验收。\n'
