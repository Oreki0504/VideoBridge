# VideoBridge

[English](README.md) | [简体中文](README.zh-CN.md)

**v0.8.0 新增仅字幕下载。** 使用此功能需要同时更新扩展和本地 Host。Windows 构建产物为 `VideoBridge-0.8.0-Setup.exe`；macOS 为 `VideoBridge-0.8.0-macOS.zip`，仍为尚未 Mac 实机验证的预览版。

Chrome / Edge 浏览器扩展 → Go Native Messaging Host → 本地 `yt-dlp` 与 `FFmpeg` 下载桥接工具。

当用户点击下载并启用当前站点会话模式时，扩展仅读取当前标签页对应站点的 Cookie 并通过本机 Native Messaging 传给本地 `yt-dlp`。无需关闭浏览器、不复制或解密浏览器 Cookie 数据库，也不依赖远程调试端口（CDP）或自动化控制。

- **Windows 版（v0.8.0）**：提供单文件安装器（`Setup.exe`）与源码构建脚本，支持自动检测并下载校验 `yt-dlp`、`FFmpeg`、`FFprobe`，以及在插件内检查、更新或回滚 `yt-dlp`（稳定版 / 每日构建）。
- **macOS 预览版（v0.8.0）**：支持构建 Apple Silicon / Intel 双架构压缩包（解压后运行 `Install.command`，详见 [installer/macos/README-Mac.md](installer/macos/README-Mac.md)）。目前已通过交叉编译、依赖校验与包结构测试，尚未进行 Mac 实机验收、Developer ID 签名或公证。

> **免责声明**：本项目仅作为本地浏览器与本地 `yt-dlp` / `FFmpeg` 的调用桥接工具，仅供下载用户本人拥有合法权限或站点允许保存的媒体内容。项目不提供任何 DRM 绕过或付费鉴权破解功能；所有 Cookie 仅在用户显式按站点授权后于本机单次任务中使用，任务结束立即清理，绝不上传至任何外部服务器。

## 功能特性

- **WXT + TypeScript + Manifest V3 扩展**：支持 Chrome 与 Edge（及同内核 Chromium 浏览器），弹窗右上角内置 **中文 / English** 一键切换（EN / 中文）。
- **多站点通用支持**：公共 HTTP / HTTPS 媒体页面交由 `yt-dlp` 提取，保留分 P、查询参数、片段定位等 URL 信息，默认只下载单个条目。支持 YouTube、Bilibili、Niconico、X / Twitter、Vimeo、Twitch 等 [yt-dlp 支持的站点](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md)（实际可用性受提取器版本、登录权限、地区与 DRM 限制影响，不支持 DRM 加密流）。
- **Windows 单文件安装器**：统一安装到 `%LOCALAPPDATA%\VideoBridge`，支持升级与卸载。内置固定扩展公钥以保持稳定扩展 ID，安装后只需在浏览器手动加载一次固定目录中的扩展。
- **YouTube 与 X 专项优化**：YouTube 自动剥离播放列表附加参数（支持普通视频、Shorts、直播与短链接）；X / Twitter 支持按 `/video/2` 等路径选择多视频帖子中的对应视频。
- **视频直转 GIF**：可选起点（0–86400 秒）、时长（5 / 10 / 15 / 30 / 60 秒）、宽度上限（320 / 480 / 640 px）与帧率（8 / 12 / 15 / 20 fps），默认从 0 秒截取 15 秒、480 px、12 fps，转换完成后自动清理临时源视频。
- **依赖自动补齐**：缺失 `yt-dlp` / `FFmpeg` / `FFprobe` 时自动从官方发布源下载、校验 SHA256 并安装到 Host 的 `tools` 子目录；若本机已有可用工具则直接复用，不修改系统 `PATH`。
- **yt-dlp 更新管理**：在扩展弹窗内支持稳定版（Stable）与每日构建（Nightly）通道检查、更新与离线回滚上一版本。
- **丰富的画质与音频选项**：
  - 视频分辨率上限：最佳（不限）、4320p (8K)、2160p (4K)、1440p、1080p、720p、480p、360p、240p、144p。
  - 仅音频模式：原始音频（默认，不重编码）、MP3（最佳或 128 / 192 / 256 / 320 kbps）、M4A / AAC、FLAC、WAV。
- **干净的输出文件名**：下载完成后自动移除末尾的 `[视频ID]` 串，保留视频标题与格式后缀；同名文件自动按 `(2)`、`(3)` 递增编号，避免覆盖。
- **分片并发下载**：支持 1 / 4 / 8 / 16 并发分片（默认 4），对应 `yt-dlp --concurrent-fragments`。
- **严格的会话与安全边界**：
  - 站点 Cookie 授权与“使用当前站点会话”开关相互独立；开启后每次任务实时读取，仅包含当前可注册域及子域的 Cookie，HTTP 页面强制匿名。
  - 任务级 Netscape Cookie 临时文件采用受保护权限（Windows 下仅当前用户与 SYSTEM 可访问），任务完成、失败或取消后立即删除，启动时自动清理异常残留。
  - 使用参数数组调用子进程（防命令注入），Windows 下通过 Job Object 托管进程树，取消任务或 Host 退出时干净终止子进程。
  - Cookie 内容、完整请求与 `yt-dlp` 原始输出不写日志、不进扩展 `storage`、不回传弹窗。

## 环境要求

- **操作系统与浏览器**：Windows 11（或 macOS 预览版），Chrome / Edge 120+。
- **源码构建依赖**：Node.js 22.12+（或更新 LTS）、npm、Go 1.26+。（直接运行安装包无需安装开发工具）
- **运行时媒体工具**：近期版本的 [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation)、[FFmpeg 与 FFprobe](https://ffmpeg.org/download.html)（`ffprobe` 需与 `ffmpeg` 位于同一目录，也可由插件自动下载补齐）。
- **附加说明**：YouTube 近期部分提取策略可能还需要 `yt-dlp` 官方支持的 JavaScript runtime / PO Token 组件，相关组件不由本项目自动安装。

## 源码构建

### Windows 构建

在 PowerShell 中运行：

```powershell
cd <repo-root>
.\scripts\Build.ps1
```

脚本依次执行锁文件依赖安装、TypeScript 检查、扩展测试、WXT 构建、Go 测试、`go vet` 与本地程序编译。若已安装前端依赖，可加 `-SkipInstall`。如需进一步打包 Windows 单文件安装器（需安装 Inno Setup 6/7）：

```powershell
.\scripts\Build-Installer.ps1
```

主要构建产物：

```text
extension/.output/chrome-mv3/        可加载的浏览器扩展目录（Chrome / Edge 通用）
dist/host/videobridge.exe            Windows 本地 Native Messaging Host
dist/VideoBridge-0.8.0-Setup.exe     Windows 单文件安装程序
```

### macOS 预览包构建

可通过以下命令交叉编译并生成 macOS 双架构安装包（`dist/VideoBridge-0.8.0-macOS.zip`）：

```powershell
.\scripts\Build-Mac.ps1
```

macOS 端的详细安装与卸载步骤见 [installer/macos/README-Mac.md](installer/macos/README-Mac.md)。

## 安装与使用（Windows）

若使用打包好的 `Setup.exe`，双击运行后按向导提示在浏览器中加载一次 `%LOCALAPPDATA%\VideoBridge\extension` 即可。若从源码安装，请按以下步骤操作：

1. 在 Chrome 打开 `chrome://extensions`（Edge 打开 `edge://extensions`），开启右上角“开发者模式”，点击“加载已解压的扩展程序”，选择 `extension\.output\chrome-mv3`。
2. 复制扩展卡片上的 32 位扩展 ID。
3. 在 PowerShell 中为当前用户注册本地 Host（无需管理员权限）：

```powershell
.\scripts\Install-Host.ps1 `
  -ExtensionId '替换为扩展卡片上的32位ID' `
  -Browser Chrome `
  -OutputDir 'D:\Downloads\VideoBridge'
```

- 如不指定 `-OutputDir`，默认下载目录为系统下载文件夹下的 `Downloads\VideoBridge`（即 `C:\Users\<用户名>\Downloads\VideoBridge`）。
- 工具路径可省略，或通过 `-YtDlpPath` / `-FFmpegPath` 指定绝对路径；配置保存在 `%LOCALAPPDATA%\VideoBridge\host\config.json`。
- `-Browser Edge` 可注册 Edge；当两个浏览器的扩展 ID 一致时可使用 `-Browser Both`。

4. 打开目标视频页面，点击浏览器工具栏中的 VideoBridge 图标。
5. 展开“本地桥接”，点击“检测并补齐依赖”，确认三个本地工具状态正常。
6. 如需下载需要登录或会员权限的视频，先点击“授权”授予当前站点 Cookie 读取权限，并勾选“使用当前站点会话”。
7. 选择模式（视频 / 仅音频 / GIF）、分辨率上限、格式与分片并发数，点击开始下载。下载在后台 Service Worker 中保持，关闭弹窗不会中断任务。

## 下载字幕

在“下载内容”中选择“仅字幕”，选择语言和 **SRT / WebVTT**，点击“下载字幕”即可。默认选择中文＋英文、SRT，并包含站点自动字幕；也可选择中文、英文、日文或全部可用语言。

通过 yt-dlp 的字幕能力支持 YouTube、Bilibili 等平台，只保存独立字幕文件，不下载视频内容。同一语言优先人工字幕，可按开关包含站点提供的自动字幕；本功能不进行语音识别或生成翻译。语言预设兼容 Bilibili 的 AI 字幕语言标识，弹幕以及烧录在画面里的字幕不属于可下载字幕轨道。

需要登录的字幕，须先授权站点并开启“使用当前站点会话”。是否可获取取决于视频、账号权限及提取器版本。没有匹配字幕时明确提示 `SUBTITLES_UNAVAILABLE`，可尝试其他语言或开启自动字幕。选择全部语言可能下载大量字幕并触发站点限流。

文件保存到已配置的下载目录，命名如 `标题.en.srt`、`标题.zh-Hans.srt`，同名文件递增编号。扩展与 Host 均需要 v0.8.0 或更新版本，仅更新 yt-dlp 不会升级 VideoBridge。

## yt-dlp 更新管理

在插件弹窗展开“本地桥接”，可选择“稳定版”或“每日构建”通道，点击“检查更新”查看最新版本，并可一键“更新 yt-dlp”。

- 更新包从 `yt-dlp` 官方 GitHub Release 下载，通过 SHA256 校验与 `--version` 运行检查后才切换本地配置。
- 不会覆盖用户通过 Scoop 或手动安装在外部目录的原工具，也不修改系统 `PATH`。
- 更新成功后支持一键离线“恢复上一版本”。

## 权限与会话边界

| 权限 | 用途 |
| --- | --- |
| `activeTab` | 点击扩展时获取当前标签页的视频地址与标题 |
| `storage` | 保存各站点会话开关与下载偏好设置；绝不保存任何 Cookie 内容 |
| `nativeMessaging` | 连接本机 `com.videobridge.host` |
| 可选 `cookies` + 当前站点 HTTP/HTTPS 域 | 运行时按需申请当前站点的域名权限（同时申请 `http://` 与 `https://` 以确保能读取未标记 `Secure` 的登录 Cookie，如 Niconico 的 `user_session`）；仅在 HTTPS 页面发起会话下载 |

Cookie 临时目录位于 `%LOCALAPPDATA%\VideoBridge\cookie-tasks\job-*`（macOS 位于 `~/Library/Application Support/VideoBridge/cookie-tasks/job-*`），单次任务结束后立即删除。

## 常见错误代码

| 代码 | 含义与处理方式 |
| --- | --- |
| `HOST_DISCONNECTED` | 检查扩展 ID、Native Host 注册表项、配置文件与可执行文件路径；重载扩展后重试 |
| `YTDLP_MISSING` / `FFMPEG_MISSING` | 点击“检测并补齐依赖”自动安装，或手动配置工具路径（`ffprobe` 需与 `ffmpeg` 同目录） |
| `SETUP_FAILED` / `SETUP_BUSY` | 检查网络连接或等待正在进行的工具安装/更新完成后重试 |
| `HOST_UPDATE_REQUIRED` | 完成新版 Host 安装后，在浏览器扩展管理页点击“重新加载” |
| `GIF_EMPTY` / `GIF_CONVERSION_FAILED` | 检查截取起点是否超出视频总时长、磁盘空间及 FFmpeg 状态 |
| `UNSUPPORTED_MEDIA` | `yt-dlp` 未识别到可下载媒体；请打开具体播放页或尝试更新 `yt-dlp` |
| `DRM_PROTECTED` | 受 DRM 加密保护的流媒体不支持下载 |
| `COOKIE_PERMISSION_REQUIRED` | 已开启会话模式但尚未授权该站点 Cookie，点击“授权”后再试 |
| `COOKIE_UNAVAILABLE` | 当前浏览器会话中未检测到该站点的有效登录 Cookie，请先在网页登录或关闭会话模式 |
| `PARTITIONED_COOKIES_UNSUPPORTED` | 当前会话依赖无法无损转为 Netscape 格式的分区 Cookie |
| `COOKIE_REJECTED` | 站点拒绝了提交的 Cookie（可能已过期），请重新登录网页后重试 |
| `MEMBERSHIP_REQUIRED` / `AUTH_REQUIRED` | 站点要求登录、会员资格、年龄确认或风控验证；请确认已授权并勾选“使用当前站点会话”，且当前账号在网页端可完整播放 |
| `EXTRACTOR_TOKEN_REQUIRED` | 请更新 `yt-dlp` 并参考其官方 PO Token / JavaScript runtime 说明 |
| `NETWORK_ERROR` | 检查网络连接、系统代理与目标站点可用性 |

## 测试

```powershell
npm run check
npm test
npm run build
npm run test:ui
cd host
go test ./...
go vet ./...
```

如需运行真实浏览器与 Native Messaging 联调测试（使用独立测试 Profile，结束后自动恢复当前用户的注册表配置）：

```powershell
.\scripts\Test-BrowserNative.ps1 -Browser chrome
.\scripts\Test-BrowserNative.ps1 -Browser msedge
```

## 卸载

先停止正在进行的下载任务并移除浏览器扩展，随后运行：

```powershell
.\scripts\Uninstall-Host.ps1 -Browser Both
```

若使用 Windows 安装包安装，也可直接通过系统“已安装的应用”或开始菜单中的“卸载 VideoBridge”完成卸载（保留已下载的媒体文件）。

## 致谢 (Acknowledgements)

VideoBridge 使用 [yt-dlp](https://github.com/yt-dlp/yt-dlp) 作为视频解析与下载引擎，并使用 [FFmpeg](https://ffmpeg.org/) 进行音视频与 GIF 媒体处理。

VideoBridge 是独立开源项目，与 yt-dlp 或 FFmpeg 开发者没有隶属或官方背书关系。

感谢 yt-dlp 与 FFmpeg 贡献者们的出色工作。
