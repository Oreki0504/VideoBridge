# VideoBridge __VERSION__ for macOS (Preview / 预览版)

[English](#english) | [简体中文](#简体中文)

---

## English

**Target**: macOS 12 or newer, Apple Silicon (`arm64`) and Intel (`x86_64`), Google Chrome and Microsoft Edge.
Cross-compilation, checksums, and installer file operations have been verified on Windows; real-Mac installation, Gatekeeper prompts, and live download verification are still pending.

### Installation

1. Extract the entire ZIP archive while keeping its directory structure, then double-click `Install.command` (scripts follow your system language automatically, or run `./Install.command --lang=en` / `--lang=zh` in Terminal). Your Mac architecture is selected automatically; `sudo`, Go, Node.js, Python, and Homebrew are not required.
2. Wait for dependency verification and setup to finish. Files are installed to `~/Library/Application Support/VideoBridge`. Existing working Homebrew tools (`/opt/homebrew/bin` and `/usr/local/bin`) are reused automatically.
3. Open `chrome://extensions` in Chrome (or `edge://extensions` in Edge), enable **Developer mode**, and click **Load unpacked**. Press **Command + Shift + G**, enter `~/Library/Application Support/VideoBridge/extension`, and select that folder.
4. Open a media page and click the VideoBridge icon to download video, audio, GIFs, or subtitles. For content requiring sign-in, click **Grant** and enable **Use current site session**. The pinned extension ID is `knllhbbekgpmakfppghioeknafdonoao`, and downloads default to `~/Downloads/VideoBridge`. The popup follows your browser language on first use and includes an **EN / 中文** toggle.

This preview package is not signed with an Apple Developer ID or notarized. If macOS blocks opening the script or binary, verify the download source and SHA256 checksum, then use the per-file **Open Anyway** option under **System Settings → Privacy & Security**. Do not disable Gatekeeper globally or recursively strip quarantine attributes.

After installation, you can delete the extracted archive folder; your browser loads the extension from the installed directory under `~/Library/Application Support/VideoBridge/extension`. To upgrade, run `Install.command` from the new release archive and reload the extension in your browser (existing download folders and tool settings are preserved). Do not reload the extension while a download is active.

### Dependencies & Updates

- `yt-dlp` uses the official `yt-dlp_macos` universal binary verified against `SHA2-256SUMS` (no system Python required). Stable / Nightly channel checks, updates, and offline rollback are available directly inside the extension popup.
- Missing `FFmpeg` / `FFprobe` binaries are downloaded on demand from OSXExperts static builds and verified against pinned executable SHA256 hashes (Apple Silicon 9.0.2, Intel 8.0; see `DEPENDENCIES.md`).
- If network or checksum verification fails, existing settings are preserved and you can retry via **Local Bridge → Check & Install Dependencies** in the popup, or set custom absolute paths in `~/Library/Application Support/VideoBridge/host/config.json` (`ffprobe` must be in the same folder as `ffmpeg`). `PATH` and browser profiles are never modified.
- Temporary per-job cookie files are created under `~/Library/Caches/VideoBridge/cookie-tasks` (`0700` directory, `0600` files) and deleted immediately when the task ends. Safari and Firefox are not supported in this release.

### Uninstalling

Stop any active downloads, run `Uninstall.command` to remove the Chrome / Edge Native Messaging registrations owned by this installation, and remove the extension from your browser. Program files, managed tools, settings, and downloaded media are kept; once you confirm they are no longer needed, you can manually delete `~/Library/Application Support/VideoBridge`.

### PKG Builder & Verification

This archive delivers runnable `.command` scripts rather than a signed PKG. On a Mac with Apple Command Line Tools installed, you can run `Build-Pkg.command` to build an unsigned per-user `.pkg` that stages the installer into `~/Library/Application Support/VideoBridge/Installer`. In a macOS development environment, run `scripts/Test-Mac.sh` from the repository to execute Go tests and builds on macOS.

---

## 简体中文

支持目标：macOS 12 或更新版本，Apple Silicon / Intel，Chrome / Edge。
当前已在 Windows 上交叉编译与检查包结构，尚未完成 Mac 实机安装、Gatekeeper 和下载验收。安装与卸载脚本已支持中英文自动识别（也可通过 `--lang=zh` 或 `--lang=en` 指定）。

### 安装

1. 将整个压缩包解压，保留目录结构，双击 `Install.command`。程序自动选择 Mac 架构并按系统语言显示提示；不需要 sudo、Go、Node.js、Python 或 Homebrew。
2. 等待依赖安装完成。文件集中保存到 `~/Library/Application Support/VideoBridge`。已有可运行的 Homebrew 工具会优先复用；GUI 浏览器也能识别 `/opt/homebrew/bin` 和 `/usr/local/bin`。
3. Chrome 打开 `chrome://extensions`（Edge 为 `edge://extensions`），开启开发者模式，点击“加载已解压的扩展程序”。按 Command + Shift + G，输入 `~/Library/Application Support/VideoBridge/extension`，选择此目录。
4. 打开媒体页，可选择下载视频、仅音频、GIF 或仅字幕；需要登录的媒体请通过插件授权当前站点并开启“使用当前站点会话”。安装版扩展 ID 为 `knllhbbekgpmakfppghioeknafdonoao`，默认下载到 `~/Downloads/VideoBridge`。扩展首次打开跟随浏览器语言，支持右上角 **EN / 中文** 切换。

本预览版没有 Apple Developer ID 签名或公证。如果 macOS 阻止打开，请核对下载来源与 SHA256，使用系统“隐私与安全性”提供的针对本文件的打开选项；如果系统没有允许入口，请保留提示并反馈。不要关闭 Gatekeeper，也不要批量移除隔离属性。安装程序本身不会修改这些保护设置。

完成安装后可以删除解压目录；浏览器必须加载固定安装目录。升级时运行新包的 `Install.command`，再重新加载扩展；下载目录和工具配置保留。不要在下载进行时重新加载扩展。

### 依赖与更新

`yt-dlp` 使用官方 `yt-dlp_macos` 通用二进制，SHA256 校验后安装；无需系统 Python。支持插件内稳定版 / 每日构建检查、更新和恢复上一版本。`FFmpeg` / `FFprobe` 缺失时使用 OSXExperts 第三方静态构建，按发布者公布的可执行文件 SHA256 固定校验：Apple Silicon 9.0.2，Intel 8.0。来源及限制见 `DEPENDENCIES.md`。这些工具按需下载，不随压缩包打包。

网络或校验失败时保留原配置，已注册的插件可以通过“本地桥接 → 检测并补齐依赖”重试。已有工具路径可在 `~/Library/Application Support/VideoBridge/host/config.json` 配置，使用绝对路径；`FFprobe` 应与 `FFmpeg` 在同一目录。工具安装不改 `PATH`、不写浏览器 Profile。

Cookie 临时文件位于 `~/Library/Caches/VideoBridge/cookie-tasks`，目录权限 `0700`、文件 `0600`，任务结束后删除。请只授予需要的网站会话权限。Safari 和 Firefox 不在此版本支持范围内。

### 卸载

先结束下载，再运行 `Uninstall.command` 解除属于本安装的 Chrome / Edge 本地连接，并从浏览器移除扩展。程序目录、依赖、配置和下载文件会保留。确认不再需要后，可手动移除 `~/Library/Application Support/VideoBridge`；你的下载目录不会被脚本删除。

### PKG 与验证

当前交付的是可运行安装脚本的 ZIP，**不是已签名或已实测的 PKG**。有 Mac 与 Apple Command Line Tools 时，可运行 `Build-Pkg.command` 生成当前用户范围的 PKG。该 PKG 安装后仍需运行固定目录中的 `Install.command` 完成注册与依赖设置。公开分发前需完成签名、公证、干净机器安装及 Chrome / Edge 实机下载验收。

在 Mac 开发环境可运行 `scripts/Test-Mac.sh`（源码仓库中）检查 Go 测试和构建。运行 `Install.command` 后应检查插件内 Host 版本、三项依赖、真实下载、取消、临时 Cookie 清理以及更新 / 恢复；Windows 上的通过结果不代表这些 Mac 行为已验证。
