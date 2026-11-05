# VideoBridge

[English](README.md) | [简体中文](README.zh-CN.md)

Download videos, audio, GIFs, and subtitles from Chrome and Edge using yt-dlp and FFmpeg. Supports YouTube, Bilibili, Niconico, X/Twitter, and other [yt-dlp supported sites](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md).

**[Downloads & release notes](https://github.com/Oreki0504/VideoBridge/releases)**

## Features

- Video quality and audio format selection, GIF conversion, and concurrent fragment downloads.
- SRT/WebVTT subtitles with language presets and optional auto-generated captions.
- Browser session access with per-site authorization; no manual Cookie exports or browser restarts.
- Automatic dependency setup, Stable/Nightly yt-dlp updates, and rollback.
- English and Simplified Chinese interface and installation instructions.

## Installation

### Windows

1. Download and run the Windows installer from [Releases](https://github.com/Oreki0504/VideoBridge/releases).
2. Open `chrome://extensions` or `edge://extensions` and enable **Developer mode**.
3. Click **Load unpacked** and select `%LOCALAPPDATA%\VideoBridge\extension`.

No administrator privileges are required. The extension must currently be loaded manually; it is not available in browser stores.

### macOS — Preview

Download the macOS ZIP and follow the [macOS installation guide](installer/macos/README-Mac.md). Supports Apple Silicon and Intel; currently unsigned, not notarized, and not yet tested on physical Mac hardware.

## Usage

1. Open a media page and click the VideoBridge toolbar icon.
2. For content requiring login, click **Grant** for the site and enable **Use current site session**. Permission alone does not enable session mode.
3. Choose **Video**, **Audio Only**, **GIF**, or **Subtitles Only**, adjust the options, and start the download.

Files default to `Downloads/VideoBridge` in your user folder. Downloads continue when the popup is closed. Subtitle availability depends on the site; speech transcription and Bilibili danmaku are not supported.

## Updates & troubleshooting

- **VideoBridge:** Install the new release and reload the browser extension.
- **yt-dlp:** Open **Local Bridge** to check for updates, select Stable/Nightly, or roll back.
- **Missing tools:** Select **Check & Install Dependencies** in **Local Bridge**.
- **Login errors:** Confirm the media plays in your browser, grant site access, and enable session mode. Some YouTube downloads require additional [yt-dlp JavaScript runtime or PO Token setup](https://github.com/yt-dlp/yt-dlp#dependencies).

Only download content you are authorized to save. DRM-protected media is unsupported. Authorized site cookies are passed to the local downloader for requests to that site; temporary Cookie files are deleted after the task.

## Build from source

Requires Node.js 22.12+, npm, and Go 1.26+. From the repository root in PowerShell:

```powershell
.\scripts\Build.ps1
```

The build script runs checks and tests. Packaging: [Windows installer](scripts/Build-Installer.ps1) (Inno Setup 7) · [macOS archive](scripts/Build-Mac.ps1) (Python 3).

## Acknowledgements

VideoBridge uses [yt-dlp](https://github.com/yt-dlp/yt-dlp) as its video extraction and download engine, and [FFmpeg](https://ffmpeg.org/) for media processing.

VideoBridge is an independent project and is not affiliated with or endorsed by the yt-dlp or FFmpeg developers.

Thanks to the yt-dlp and FFmpeg contributors for their excellent work.
