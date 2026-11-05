#!/bin/bash
set -euo pipefail
trap 'if [ -t 0 ]; then read -r -p "按回车关闭…" answer; fi' EXIT
cd -- "$(dirname -- "$0")"
if [ "$(uname -s)" != Darwin ]; then echo '此程序仅适用于 macOS。'; exit 1; fi
if [ "$(id -u)" = 0 ]; then echo '请使用普通账户运行，不要使用 sudo。'; exit 1; fi
case "$(uname -m)" in arm64) arch=arm64 ;; x86_64) arch=amd64 ;; *) exit 1 ;; esac
if [ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then arch=arm64; fi
/usr/bin/shasum -a 256 -c SHA256SUMS
"$PWD/host/darwin-$arch/videobridge" --unregister-macos
printf '\n程序文件保留于 ~/Library/Application Support/VideoBridge，可确认后手动删除。\n'
