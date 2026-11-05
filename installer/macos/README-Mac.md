# VideoBridge __VERSION__ for Mac（预览版）

支持目标：macOS 12 或更新版本，Apple Silicon / Intel，Chrome / Edge。
当前已在 Windows 上交叉编译与检查包结构，尚未完成 Mac 实机安装、Gatekeeper 和下载验收。

## 安装

1. 将整个压缩包解压，保留目录结构，双击 `Install.command`。程序自动选择 Mac 架构；不需要 sudo、Go、Node.js、Python 或 Homebrew。
2. 等待依赖安装完成。文件集中保存到 `~/Library/Application Support/VideoBridge`。已有可运行的 Homebrew 工具会优先复用；GUI 浏览器也能识别 `/opt/homebrew/bin` 和 `/usr/local/bin`。
3. Chrome 打开 `chrome://extensions`（Edge 为 `edge://extensions`），开启开发者模式，点击“加载已解压的扩展程序”。按 Command + Shift + G，输入 `~/Library/Application Support/VideoBridge/extension`，选择此目录。
4. 打开媒体页，通过插件授权当前站点会话后下载。安装版扩展 ID 为 `knllhbbekgpmakfppghioeknafdonoao`。默认下载到 `~/Downloads/VideoBridge`。

本预览版没有 Apple Developer ID 签名或公证。如果 macOS 阻止打开，请核对下载来源与 SHA256，使用系统“隐私与安全性”提供的针对本文件的打开选项；如果系统没有允许入口，请保留提示并反馈。不要关闭 Gatekeeper，也不要批量移除隔离属性。安装程序本身不会修改这些保护设置。

完成安装后可以删除解压目录；浏览器必须加载固定安装目录。升级时运行新包的 Install.command，再重新加载扩展；下载目录和工具配置保留。不要在下载进行时重新加载扩展。

## 依赖与更新

yt-dlp 使用官方 `yt-dlp_macos` 通用二进制，SHA256 校验后安装；无需系统 Python。支持插件内稳定版 / 每日构建检查、更新和恢复上一版本。FFmpeg / FFprobe 缺失时使用 OSXExperts 第三方静态构建，按发布者公布的可执行文件 SHA256 固定校验：Apple Silicon 9.0.2，Intel 8.0。来源及限制见 `DEPENDENCIES.md`。这些工具按需下载，不随压缩包打包。

网络或校验失败时保留原配置，已注册的插件可以通过“本地桥接 → 检测并补齐依赖”重试。已有工具路径可在 `~/Library/Application Support/VideoBridge/host/config.json` 配置，使用绝对路径；FFprobe 应与 FFmpeg 在同一目录。工具安装不改 PATH、不写浏览器 Profile。

Cookie 临时文件位于 `~/Library/Caches/VideoBridge/cookie-tasks`，目录权限 0700、文件 0600，任务结束后删除。请只授予需要的网站会话权限。Safari 和 Firefox 不在此版本支持范围内。

## 卸载

先结束下载，再运行 `Uninstall.command` 解除属于本安装的 Chrome / Edge 本地连接，并从浏览器移除扩展。程序目录、依赖、配置和下载文件会保留。确认不再需要后，可手动移除 `~/Library/Application Support/VideoBridge`；你的下载目录不会被脚本删除。

## PKG 与验证

当前交付的是可运行安装脚本的 ZIP，**不是已签名或已实测的 PKG**。有 Mac 与 Apple Command Line Tools 时，可运行 `Build-Pkg.command` 生成当前用户范围的 PKG。该 PKG 安装后仍需运行固定目录中的 Install.command 完成注册与依赖设置。公开分发前需完成签名、公证、干净机器安装及 Chrome / Edge 实机下载验收。

在 Mac 开发环境可运行 `scripts/Test-Mac.sh`（源码仓库中）检查 Go 测试和构建。运行 `Install.command` 后应检查插件内 Host 版本、三项依赖、真实下载、取消、临时 Cookie 清理以及更新 / 恢复；Windows 上的通过结果不代表这些 Mac 行为已验证。
