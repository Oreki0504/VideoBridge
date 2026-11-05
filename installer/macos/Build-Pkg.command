#!/bin/bash
# Run on macOS with Apple's Command Line Tools. Produces a per-user package;
# browser registration and tool setup are completed by Install.command.
set -euo pipefail
detect_lang() {
  local raw="${VIDEOBRIDGE_LANG:-}"
  if [ -z "$raw" ]; then
    for v in "${LC_ALL:-}" "${LC_MESSAGES:-}" "${LANG:-}"; do
      case "$v" in
        ""|C|C.*|POSIX|UTF-8) ;;
        *) raw="$v"; break ;;
      esac
    done
  fi
  if [ -z "$raw" ] && [ "$(uname -s)" = Darwin ]; then
    raw="$(/usr/bin/defaults read -g AppleLanguages 2>/dev/null | /usr/bin/awk 'NR > 1 { gsub(/[[:space:]",]/, ""); if ($0 ~ /^[A-Za-z][A-Za-z0-9_-]*$/) { print; exit } }' || true)"
  fi
  case "$(printf '%s' "$raw" | /usr/bin/tr '[:upper:]' '[:lower:]')" in
    zh*) printf 'zh' ;;
    *) printf 'en' ;;
  esac
}
for arg in "$@"; do
  case "$arg" in
    --lang=zh|zh) VIDEOBRIDGE_LANG=zh ;;
    --lang=en|en) VIDEOBRIDGE_LANG=en ;;
  esac
done
export VIDEOBRIDGE_LANG="$(detect_lang)"
cd -- "$(dirname -- "$0")"
bundle="$PWD"
if [ "$(uname -s)" != Darwin ]; then
  if [ "$VIDEOBRIDGE_LANG" = zh ]; then echo 'pkgbuild / productbuild 需要 macOS。'; else echo 'pkgbuild / productbuild requires macOS.'; fi
  exit 1
fi
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
<!doctype html><meta charset="utf-8">
<h2>Files Installed / 文件已安装</h2>
<p>In Finder, press Command + Shift + G and open:</p>
<p><code>~/Library/Application Support/VideoBridge/Installer</code></p>
<p>Double-click <code>Install.command</code> to register the Native Host and install dependencies, then load the browser extension as described in <code>README-Mac.md</code>.</p>
<hr>
<p>请在 Finder 按 Command + Shift + G，打开上面的目录，双击 <code>Install.command</code> 完成本地桥接注册与依赖安装，再按 <code>README-Mac.md</code> 导入浏览器扩展。</p>
HTML
productbuild --distribution "$stage/Distribution.xml" --package-path "$stage" --resources "$stage" "$bundle/../VideoBridge-__VERSION__-macOS.pkg"
if [ "$VIDEOBRIDGE_LANG" = zh ]; then
  printf '\n生成的是未签名 PKG。公开分发前需 Developer ID Installer 签名、公证和 Mac 实机验收。\n'
else
  printf '\nBuilt an unsigned PKG. Sign with Developer ID Installer, notarize, and verify on a Mac before public distribution.\n'
fi
