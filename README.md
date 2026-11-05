# VideoBridge

[English](README.md) | [简体中文](README.zh-CN.md)

**v0.8.0 adds subtitle-only downloads.** Update both the extension and the Native Host to use this feature. Windows builds produce `VideoBridge-0.8.0-Setup.exe`; macOS builds produce `VideoBridge-0.8.0-macOS.zip` and remain a preview without real-Mac runtime verification.

A local bridge connecting Chromium-based browser extensions (Chrome / Edge) to a Go Native Messaging Host and local `yt-dlp` / `FFmpeg`.

When you start a download with site session mode enabled, VideoBridge reads cookies only for the current tab's registrable domain via the browser's `chrome.cookies` API and passes them over Native Messaging to your local `yt-dlp` process. You do not need to close your browser, copy or decrypt SQLite cookie databases, or run your browser with remote debugging ports (CDP).

- **Windows (v0.8.0)**: Includes a single-file installer (`Setup.exe`) and source build scripts, automatic detection and SHA256-verified installation of `yt-dlp`, `FFmpeg`, and `FFprobe`, plus built-in `yt-dlp` channel switching (Stable / Nightly), updates, and offline rollback.
- **macOS Preview (v0.8.0)**: Supports building a universal distribution archive (`Install.command` for Apple Silicon and Intel Macs; see [installer/macos/README-Mac.md](installer/macos/README-Mac.md)). Cross-compilation, dependency checksums, and installer file operations are verified; real-Mac runtime testing, Developer ID signing, and Apple notarization are not yet included.

> **Disclaimer**: VideoBridge is solely a local bridge between your browser and your local `yt-dlp` / `FFmpeg` installation. Only download content that you own or are authorized to save. This project does not bypass DRM (such as Widevine) or circumvent paywalls. Cookies are read only after explicit per-site user authorization, written to an isolated per-job temporary file on your local machine, and deleted immediately when the job finishes.

## Features

- **Manifest V3 Extension (WXT + TypeScript)**: Works on Chrome, Microsoft Edge, and other Chromium-based browsers, with a one-click **English / 简体中文** UI toggle (EN / 中文) in the popup header.
- **Broad Multi-Site Support**: Delegates HTTP/HTTPS media extraction to `yt-dlp`, preserving query parameters, multi-part indices, and timestamps while downloading a single entry by default. Supports YouTube, Bilibili, Niconico, X / Twitter, Vimeo, Twitch, and other [yt-dlp supported sites](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md) (subject to extractor support, account access, region, and DRM restrictions; DRM-protected streams are unsupported).
- **Single-File Windows Installer**: Installs to `%LOCALAPPDATA%\VideoBridge` with upgrade and uninstall support, using a pinned extension public key so the unpacked extension ID remains stable across installs.
- **YouTube & X Refinements**: Automatically strips playlist query parameters on YouTube watch/Shorts/live URLs, and honors `/video/<index>` paths on multi-video X / Twitter posts.
- **Direct Video-to-GIF Conversion**: Configurable start offset (`0–86400s`), duration (`5 / 10 / 15 / 30 / 60s`), max width (`320 / 480 / 640px`), and frame rate (`8 / 12 / 15 / 20 fps`), defaulting to `0s`, `15s`, `480px`, `12 fps` with automatic cleanup of temporary source media.
- **Automatic Tool Provisioning**: Automatically downloads and verifies `yt-dlp`, `FFmpeg`, and `FFprobe` via SHA256 into the Host's `tools/` directory if missing, or reuses existing working binaries on your system without modifying `PATH`.
- **Built-in `yt-dlp` Updater**: Check for updates on the Stable or Nightly channel, upgrade with SHA256 and runtime verification, and roll back to the previous binary offline directly from the extension popup.
- **Flexible Video & Audio Options**:
  - Video resolution caps: Best (unlimited), 4320p (8K), 2160p (4K), 1440p, 1080p, 720p, 480p, 360p, 240p, and 144p.
  - Audio-only formats: Original audio stream (default, no re-encoding), MP3 (best or 128 / 192 / 256 / 320 kbps), M4A / AAC, FLAC, and WAV.
- **Clean Output Filenames**: Automatically strips trailing `[<video_id>]` suffixes upon completion while preserving media titles and format tags, numbering duplicate filenames as `(2)`, `(3)` without overwriting existing files.
- **Concurrent Fragment Downloads**: Configurable fragment concurrency (`1 / 4 / 8 / 16`, default `4`) mapped to `yt-dlp --concurrent-fragments`.
- **Strict Security & Session Boundaries**:
  - Per-site cookie permission grants and the "Use current site session" toggle are stored separately; cookies are snapshotted fresh per job and scoped strictly to the active tab's registrable domain (HTTPS only).
  - Per-job Netscape cookie files are created with restricted permissions (current user + `SYSTEM` DACL on Windows, `0600` on macOS) and deleted immediately on completion, failure, or cancellation, with stale-job cleanup on Host startup.
  - Subprocesses are spawned via argument arrays (never shell strings) and managed by a Windows Job Object (or process group on macOS) so cancelling a job or closing the Host terminates the entire process tree.
  - Cookie values, raw requests, and raw `yt-dlp` stderr are never written to logs, extension storage, or popup messages.

## Requirements

- **OS & Browser**: Windows 11 (or macOS preview), Google Chrome or Microsoft Edge 120+.
- **Build from Source**: Node.js 22.12+ (or newer LTS), npm, Go 1.26+. (Not required when installing from a prebuilt release package.)
- **Runtime Tools**: Recent [yt-dlp](https://github.com/yt-dlp/yt-dlp#installation), [FFmpeg, and FFprobe](https://ffmpeg.org/download.html) (`ffprobe` must reside in the same directory as `ffmpeg`; missing tools can be downloaded automatically by the Host).
- **Note on YouTube**: Recent YouTube extraction rules may additionally require a `yt-dlp`-supported JavaScript runtime or PO Token provider, which are outside the scope of VideoBridge's automatic installer.

## Building from Source

### Windows Build

Run in PowerShell:

```powershell
cd <repo-root>
.\scripts\Build.ps1
```

This script runs `npm ci`, TypeScript type-checking, extension unit tests, the WXT production build, Go tests, `go vet`, and compiles the Native Host executable. Pass `-SkipInstall` if `node_modules` is already installed. To build the single-file Windows installer (requires Inno Setup 6/7):

```powershell
.\scripts\Build-Installer.ps1
```

Build outputs:

```text
extension/.output/chrome-mv3/        Unpacked MV3 extension directory (Chrome & Edge)
dist/host/videobridge.exe            Windows Native Messaging Host binary
dist/VideoBridge-0.8.0-Setup.exe     Windows single-file installer
```

### macOS Preview Package Build

Cross-compile and package the universal macOS archive (`dist/VideoBridge-0.8.0-macOS.zip`):

```powershell
.\scripts\Build-Mac.ps1
```

For macOS installation and usage instructions, see [installer/macos/README-Mac.md](installer/macos/README-Mac.md).

## Installation & Usage (Windows)

If you have the prebuilt `Setup.exe`, run it and follow the welcome guide to load the unpacked extension from `%LOCALAPPDATA%\VideoBridge\extension` once. To install from a source build:

1. Open `chrome://extensions` in Chrome (or `edge://extensions` in Edge), enable **Developer mode**, click **Load unpacked**, and select `extension\.output\chrome-mv3`.
2. Copy the 32-character Extension ID shown on the extension card.
3. Register the Native Messaging Host for your current user in PowerShell (no Administrator privileges required):

```powershell
.\scripts\Install-Host.ps1 `
  -ExtensionId 'YOUR_32_CHAR_EXTENSION_ID' `
  -Browser Chrome `
  -OutputDir 'D:\Downloads\VideoBridge'
```

- If `-OutputDir` is omitted, downloads default to `Downloads\VideoBridge` inside your user profile (`C:\Users\<user>\Downloads\VideoBridge`).
- You can optionally pass `-YtDlpPath` and `-FFmpegPath`; configuration is saved to `%LOCALAPPDATA%\VideoBridge\host\config.json`.
- Use `-Browser Edge` for Edge, or `-Browser Both` when both browsers share the same Extension ID.

4. Open a video page and click the VideoBridge toolbar icon.
5. Expand **本地桥接 (Local Bridge)** and click **检测并补齐依赖 (Check & Install Dependencies)** to verify or auto-install `yt-dlp`, `FFmpeg`, and `FFprobe`.
6. For videos that require sign-in or membership access, click **授权 (Grant)** for the current site and enable the **使用当前…会话 (Use current site session)** toggle.
7. Choose your output mode (Video / Audio Only / GIF), quality cap, audio format, and fragment concurrency, then start the download. Downloads continue in the background Service Worker even if the popup is closed.

## Downloading Subtitles

Select **Subtitles Only** in **Download Mode**, choose a language and **SRT** or **WebVTT**, then click **Download Subtitles**. The default is Chinese + English, SRT, with site-generated captions enabled. Chinese, English, Japanese, and all available languages are also selectable.

VideoBridge uses yt-dlp's subtitle support for YouTube, Bilibili, and other supported extractors. It downloads separate subtitle files without fetching the video body, preferring human subtitles for each language and optionally including site-generated captions. This does not perform speech recognition or create translations. Bilibili AI subtitle language codes are included in the language presets; danmaku and subtitles burned into video are not downloadable subtitle tracks.

For subtitles requiring login, grant site access and enable **Use current site session**. Availability depends on the video, account, and extractor version. If no matching subtitles are available, the task reports `SUBTITLES_UNAVAILABLE`; try another language or enable automatic captions. Selecting all languages can download many tracks and trigger site rate limits.

Files use names such as `Title.en.srt` and `Title.zh-Hans.srt` in your configured download folder. Existing files are preserved with numbered duplicates. Both the extension and Host must be v0.8.0 or newer; updating yt-dlp alone does not update VideoBridge.

## `yt-dlp` Update Management

Inside the **本地桥接 (Local Bridge)** section of the popup, select either the **Stable** or **Nightly** channel, click **检查更新 (Check Update)**, and click **更新 yt-dlp (Update yt-dlp)**.

- Updates are downloaded directly from official `yt-dlp` GitHub Releases, verified against published `SHA2-256SUMS`, and checked with `--version` before switching configuration.
- External `yt-dlp` binaries installed via Scoop or manual paths are never overwritten, and `PATH` is untouched.
- After an update, you can revert to the previous binary offline at any time using **恢复上一版本 (Rollback)**.

## Permissions & Session Boundaries

| Permission | Purpose |
| --- | --- |
| `activeTab` | Reads the current tab's URL and title when you open the popup |
| `storage` | Stores per-site session toggles and download preferences; never stores cookie values |
| `nativeMessaging` | Connects to the local `com.videobridge.host` binary |
| Optional `cookies` + per-site HTTP/HTTPS origins | Requested on demand only for the current site (`http://` and `https://` patterns are both requested so Chrome also returns non-`Secure` session cookies such as Niconico's `user_session`); authenticated downloads are only allowed on HTTPS pages |

Temporary per-job cookie files are stored under `%LOCALAPPDATA%\VideoBridge\cookie-tasks\job-*` on Windows (`~/Library/Application Support/VideoBridge/cookie-tasks/job-*` on macOS) and removed as soon as the job finishes.

## Troubleshooting Error Codes

| Code | Meaning & Resolution |
| --- | --- |
| `HOST_DISCONNECTED` | Check Extension ID, Native Host registry entry, `config.json`, and Host binary path; reload the extension and recheck |
| `YTDLP_MISSING` / `FFMPEG_MISSING` | Click "Check & Install Dependencies" in the popup, or configure binary paths manually (`ffprobe` must be next to `ffmpeg`) |
| `SETUP_FAILED` / `SETUP_BUSY` | Check network connectivity or wait for an active tool installation/update to finish |
| `HOST_UPDATE_REQUIRED` | Reload the VideoBridge extension on `chrome://extensions` after updating the Host |
| `GIF_EMPTY` / `GIF_CONVERSION_FAILED` | Verify that the start offset is within the video duration and check disk space / FFmpeg status |
| `UNSUPPORTED_MEDIA` | `yt-dlp` did not find downloadable media on this URL; open the direct watch page or update `yt-dlp` |
| `DRM_PROTECTED` | DRM-encrypted media is not supported |
| `COOKIE_PERMISSION_REQUIRED` | Site session mode is enabled but cookie permission has not been granted; click Grant and retry |
| `COOKIE_UNAVAILABLE` | No valid login cookies were found in the current browser store for this site; sign in first or disable session mode |
| `PARTITIONED_COOKIES_UNSUPPORTED` | The session relies on partitioned cookies that cannot be losslessly exported to Netscape format |
| `COOKIE_REJECTED` | The site rejected the supplied cookies (expired or invalid); sign in again in your browser and retry |
| `MEMBERSHIP_REQUIRED` / `AUTH_REQUIRED` | The site requires authentication, membership, age verification, or anti-bot checks; ensure both site authorization and the session toggle are enabled and that the video plays in your browser |
| `EXTRACTOR_TOKEN_REQUIRED` | Update `yt-dlp` and consult upstream documentation for PO Token / JavaScript runtime requirements |
| `NETWORK_ERROR` | Check your network connection, proxy settings, and site availability |

## Testing

```powershell
npm run check
npm test
npm run build
npm run test:ui
cd host
go test ./...
go vet ./...
```

To run end-to-end browser + Native Messaging integration tests (uses an isolated browser profile and automatically restores your `HKCU` Native Messaging registration afterward):

```powershell
.\scripts\Test-BrowserNative.ps1 -Browser chrome
.\scripts\Test-BrowserNative.ps1 -Browser msedge
```

## Uninstalling

Stop any active downloads, remove the extension from your browser, and run:

```powershell
.\scripts\Uninstall-Host.ps1 -Browser Both
```

If installed via the Windows installer (`Setup.exe`), you can also uninstall directly from **Installed Apps** in Windows Settings or the **VideoBridge** Start Menu folder (downloaded media files are preserved).

## Acknowledgements

VideoBridge uses [yt-dlp](https://github.com/yt-dlp/yt-dlp) as its video extraction and download engine, and [FFmpeg](https://ffmpeg.org/) for media processing.

VideoBridge is an independent project and is not affiliated with or endorsed by the yt-dlp or FFmpeg developers.

Thanks to the yt-dlp and FFmpeg contributors for their excellent work.
