# VideoBridge

[English](README.md) | [简体中文](README.zh-CN.md)

通过 Chrome / Edge 调用 yt-dlp 和 FFmpeg，下载视频、音频、GIF 与字幕。支持 YouTube、Bilibili、Niconico、X/Twitter 及其他 [yt-dlp 支持的站点](https://github.com/yt-dlp/yt-dlp/blob/master/supportedsites.md)。

**[下载安装包与查看更新说明](https://github.com/Oreki0504/VideoBridge/releases)**

## 功能

- 选择视频画质、音频格式，转换 GIF，设置分片下载并发数。
- 下载 SRT / WebVTT 字幕，支持语言预设及站点自动字幕。
- 按站点授权使用浏览器登录会话，无需手动导出 Cookie 或关闭浏览器。
- 自动补齐依赖，支持 yt-dlp 稳定版 / 每日构建更新与回滚。
- 中英文界面与安装说明。

## 安装

### Windows

1. 从 [Releases](https://github.com/Oreki0504/VideoBridge/releases) 下载并运行 Windows 安装包。
2. 打开 `chrome://extensions` 或 `edge://extensions`，开启“开发者模式”。
3. 点击“加载已解压的扩展程序”，选择 `%LOCALAPPDATA%\VideoBridge\extension`。

安装无需管理员权限。目前尚未上架浏览器商店，首次需要手动加载扩展。

### macOS — 预览版

下载 macOS ZIP，按 [macOS 安装指南](installer/macos/README-Mac.md) 操作。支持 Apple Silicon 与 Intel；目前未签名、未公证，也尚未完成 Mac 实机测试。

## 使用

1. 打开媒体页面，点击工具栏中的 VideoBridge 图标。
2. 内容需要登录时，先点“授权”，再勾选“使用当前站点会话”。仅授权不会自动启用会话模式。
3. 选择“视频”“仅音频”“GIF”或“仅字幕”，调整选项后开始下载。

文件默认保存到用户目录下的 `Downloads/VideoBridge`。关闭弹窗不会中断下载。字幕是否可用取决于站点；不支持语音转写或 Bilibili 弹幕下载。

## 更新与排查

- **VideoBridge：**安装新版本后，在浏览器扩展管理页重新加载扩展。
- **yt-dlp：**在“本地桥接”中检查更新、选择稳定版 / 每日构建，或恢复上一版本。
- **缺少工具：**点击“本地桥接”中的“检测并补齐依赖”。
- **登录失败：**确认网页能播放该媒体，已授权站点且启用了会话模式。部分 YouTube 下载需要额外配置 [yt-dlp JavaScript 运行时或 PO Token](https://github.com/yt-dlp/yt-dlp#dependencies)。

请仅下载有权保存的内容；不支持 DRM 加密媒体。授权的站点 Cookie 交给本地下载器，用于向该站点发起请求，临时 Cookie 文件在任务结束后删除。

## 源码构建

需要 Node.js 22.12+、npm 和 Go 1.26+。在仓库根目录的 PowerShell 中运行：

```powershell
.\scripts\Build.ps1
```

构建脚本会执行检查与测试。打包脚本：[Windows 安装器](scripts/Build-Installer.ps1)（需要 Inno Setup 7）· [macOS 压缩包](scripts/Build-Mac.ps1)（需要 Python 3）。

## 致谢

VideoBridge 使用 [yt-dlp](https://github.com/yt-dlp/yt-dlp) 作为视频解析与下载引擎，并使用 [FFmpeg](https://ffmpeg.org/) 进行媒体处理。

VideoBridge 是独立项目，与 yt-dlp 或 FFmpeg 开发者没有隶属或官方背书关系。

感谢 yt-dlp 与 FFmpeg 贡献者们的出色工作。
