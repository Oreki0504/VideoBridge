#!/bin/bash
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
trap 'status=$?; if [ "$status" -ne 0 ]; then if [ "${VIDEOBRIDGE_LANG:-en}" = zh ]; then printf "\n安装未完成。请保留上方错误信息，查看 README-Mac.md。\n"; else printf "\nInstallation did not finish. Keep the error message above and see README-Mac.md.\n"; fi; fi; if [ -t 0 ]; then if [ "${VIDEOBRIDGE_LANG:-en}" = zh ]; then read -r -p "按回车关闭…" answer; else read -r -p "Press Return to close…" answer; fi; fi' EXIT
cd -- "$(dirname -- "$0")"
bundle="$PWD"
if [ "$(uname -s)" != Darwin ]; then
  if [ "$VIDEOBRIDGE_LANG" = zh ]; then echo '此安装程序仅适用于 macOS。'; else echo 'This installer only runs on macOS.'; fi
  exit 1
fi
if [ "$(id -u)" = 0 ]; then
  if [ "$VIDEOBRIDGE_LANG" = zh ]; then echo '请使用自己的普通账户双击安装，不要使用 sudo。'; else echo 'Run this installer under your normal user account without sudo.'; fi
  exit 1
fi
case "$(uname -m)" in
  arm64) arch=arm64 ;;
  x86_64) arch=amd64 ;;
  *)
    if [ "$VIDEOBRIDGE_LANG" = zh ]; then echo '不支持此 Mac 架构。'; else echo 'Unsupported Mac architecture.'; fi
    exit 1
    ;;
esac
# Rosetta terminals may report x86_64 on an Apple Silicon Mac.
if [ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then arch=arm64; fi
/usr/bin/shasum -a 256 -c SHA256SUMS
"$bundle/host/darwin-$arch/videobridge" --install-macos "$bundle"
/usr/bin/open "$HOME/Library/Application Support/VideoBridge/extension"
