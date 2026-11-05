#!/bin/bash
set -euo pipefail
trap 'status=$?; if [ "$status" -ne 0 ]; then printf "\n安装未完成。请保留上方错误信息，查看 README-Mac.md。\n"; fi; if [ -t 0 ]; then read -r -p "按回车关闭…" answer; fi' EXIT
cd -- "$(dirname -- "$0")"
bundle="$PWD"
if [ "$(uname -s)" != Darwin ]; then echo '此安装程序仅适用于 macOS。'; exit 1; fi
if [ "$(id -u)" = 0 ]; then echo '请使用自己的普通账户双击安装，不要使用 sudo。'; exit 1; fi
case "$(uname -m)" in
  arm64) arch=arm64 ;;
  x86_64) arch=amd64 ;;
  *) echo '不支持此 Mac 架构。'; exit 1 ;;
esac
# Rosetta terminals may report x86_64 on an Apple Silicon Mac.
if [ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then arch=arm64; fi
/usr/bin/shasum -a 256 -c SHA256SUMS
"$bundle/host/darwin-$arch/videobridge" --install-macos "$bundle"
/usr/bin/open "$HOME/Library/Application Support/VideoBridge/extension"
