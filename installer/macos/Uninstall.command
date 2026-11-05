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
trap 'if [ -t 0 ]; then if [ "${VIDEOBRIDGE_LANG:-en}" = zh ]; then read -r -p "按回车关闭…" answer; else read -r -p "Press Return to close…" answer; fi; fi' EXIT
cd -- "$(dirname -- "$0")"
if [ "$(uname -s)" != Darwin ]; then
  if [ "$VIDEOBRIDGE_LANG" = zh ]; then echo '此程序仅适用于 macOS。'; else echo 'This uninstaller only runs on macOS.'; fi
  exit 1
fi
if [ "$(id -u)" = 0 ]; then
  if [ "$VIDEOBRIDGE_LANG" = zh ]; then echo '请使用普通账户运行，不要使用 sudo。'; else echo 'Run this uninstaller under your normal user account without sudo.'; fi
  exit 1
fi
case "$(uname -m)" in arm64) arch=arm64 ;; x86_64) arch=amd64 ;; *) exit 1 ;; esac
if [ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then arch=arm64; fi
/usr/bin/shasum -a 256 -c SHA256SUMS
"$PWD/host/darwin-$arch/videobridge" --unregister-macos
if [ "$VIDEOBRIDGE_LANG" = zh ]; then
  printf '\n程序文件保留于 ~/Library/Application Support/VideoBridge，可确认后手动删除。\n'
else
  printf '\nProgram files are kept in ~/Library/Application Support/VideoBridge and can be removed manually if no longer needed.\n'
fi
