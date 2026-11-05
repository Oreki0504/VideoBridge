import type { Site } from './types';
import { siteName } from './site-resolver';

export type Locale = 'zh' | 'en';

export function resolveLocale(saved: unknown): Locale {
  return saved === 'en' ? 'en' : 'zh';
}

export function localizedSiteName(site: Site, locale: Locale): string {
  if (locale === 'zh') return siteName(site);
  return ({
    youtube: 'YouTube',
    x: 'X / Twitter',
    'bilibili.com': 'Bilibili',
    'b23.tv': 'Bilibili Short Link',
    'nicovideo.jp': 'Niconico',
    'nico.ms': 'Niconico Short Link',
    'vimeo.com': 'Vimeo',
    'twitch.tv': 'Twitch',
    'dailymotion.com': 'Dailymotion',
  } as Record<string, string>)[site] ?? site;
}

export const UI_TEXT = {
  zh: {
    htmlLang: 'zh-CN',
    langToggleLabel: 'EN',
    langToggleTitle: 'Switch to English',
    brandSubtitle: '浏览器里的视频，本地的文件。',
    currentPageSuffix: '当前页面',
    readingVideo: '正在读取视频…',
    invalidPageTitle: '打开包含视频或音频的网页',
    invalidPageUrl: '支持 yt-dlp 可识别的媒体网站，包括 Bilibili、Niconico 等',
    xHint: '默认下载帖子中的第一个视频；链接含 /video/2 等编号时下载对应视频。X 默认使用匿名模式。',
    downloadTypeLabel: '下载内容',
    downloadTypes: {
      best: '视频 + 音频',
      audio: '仅音频',
      gif: 'GIF 动图 · 无声音',
      subtitles: '仅字幕',
    },
    subtitleDownload: '下载字幕',
    subtitleLanguageLabel: '字幕语言',
    subtitleLanguages: { 'zh-en': '中文 + 英文', zh: '中文', en: '英文', ja: '日文', all: '全部可用语言' },
    subtitleFormatLabel: '字幕格式',
    subtitleAutoLabel: '包含站点自动字幕',
    subtitleHint: '只下载字幕文件，不下载视频。优先人工字幕，可包含站点自动生成的字幕；不生成新字幕。Bilibili 弹幕不属于字幕。全部语言可能产生大量文件。',
    qualityLabel: '画质上限',
    qualities: {
      best: '最佳可用画质 · 不限分辨率',
      '4320': '最高 4320p · 8K',
      '2160': '最高 2160p · 4K',
      '1440': '最高 1440p · 2K',
      '1080': '最高 1080p · Full HD',
      '720': '最高 720p · HD',
      '480': '最高 480p',
      '360': '最高 360p',
      '240': '最高 240p',
      '144': '最高 144p',
    },
    qualityHint: '在所选上限内选最佳版本；实际画质取决于视频源，不会放大低分辨率视频。',
    audioFormatLabel: '音频格式',
    audioFormats: {
      original: '原始音频 · 不重新编码（推荐）',
      mp3: 'MP3 · 通用兼容',
      m4a: 'M4A / AAC',
      flac: 'FLAC',
      wav: 'WAV',
    },
    audioQualityLabel: '转换质量 / 目标码率',
    audioQualities: {
      best: '最佳转换质量 · 自动码率',
      '320': '320 kbps',
      '256': '256 kbps',
      '192': '192 kbps',
      '128': '128 kbps',
    },
    audioHints: {
      original: '保留源音轨编码，避免额外损失；文件扩展名由源格式决定。',
      lossless: '转换为 FLAC / WAV 会增大体积，不能恢复源音频已经丢失的细节。',
      m4a: '保存为 M4A，必要时转换为 AAC；转换不会提高源音质。',
      mp3: '这是转换后的目标码率，不是源音质；提高码率无法恢复源音频的细节。',
    },
    gifStartLabel: '起点（秒）',
    gifDurationLabel: '截取时长',
    gifDurations: {
      '5': '5 秒',
      '10': '10 秒',
      '15': '15 秒',
      '30': '30 秒',
      '60': '60 秒',
    },
    gifWidthLabel: '宽度上限',
    gifFpsLabel: '帧率',
    gifHint: '先下载源视频，再转换所选片段；不足指定时长则到片尾结束。GIF 无声音，较长、较大的动图会占用更多空间。',
    concurrencyLabel: '分片并发数',
    concurrencies: {
      '1': '1 · 单分片',
      '4': '4 · 推荐',
      '8': '8',
      '16': '16',
    },
    concurrencyHint: '仅对支持的 DASH / HLS 分片生效；普通 HTTP 文件不增加连接数。并发越高不一定越快。',
    cookieHint: '开启：使用登录身份下载；关闭：匿名下载',
    checkingPermission: '正在检查授权…',
    grant: '授权',
    revoke: '撤销',
    downloadButton: '下载当前视频',
    hostUnchecked: '本地桥接 · 尚未检测',
    hostConnected: '本地桥接 · 已连接',
    hostDisconnected: '本地桥接 · 未连接',
    checkDeps: '检测并补齐依赖',
    toolsDefault: '检测 yt-dlp、FFmpeg 和下载目录。缺失时自动下载并校验，首次安装需联网。需先安装本地 Host。',
    updateChannelLabel: 'yt-dlp 更新通道',
    updateChannels: {
      stable: '稳定版',
      nightly: '每日构建 · 更快获得站点修复',
    },
    updateCheck: '检查更新',
    updateInstall: '更新 yt-dlp',
    updateRollback: '恢复上一版本',
    updateStatusDefault: '先检查所选通道的版本，再选择更新。不会自动降级。',
    updateNote: '更新仅切换 VideoBridge 使用的 yt-dlp，保留原工具和上一版本。FFmpeg 仍通过“检测并补齐依赖”补齐。',
    backgroundNote: '关闭弹窗后任务继续；退出浏览器会停止任务。',
    scopeNote: '适用于 yt-dlp 支持的媒体页面，也会尝试通用提取。每次仅下载一个条目；会员、地区、DRM 等限制仍可能影响结果。',
    footerLeft: '本地处理 · 单条目下载',
    cancel: '取消',
    downloadFailed: '下载失败',
    downloadProgressAria: '下载进度',
    taskCancelledDetail: '任务已停止，临时 Cookie 已清理。',
    optionSaveFailed: '选项保存失败，本次下载仍可使用当前选择。',
    bridgeNoResponse: '本地桥接未响应。',
    grantedMessage: '已授权并开启当前站点会话。关闭会话开关可切回匿名下载。',
    grantIncompleteMessage: '授权未完成，无法读取登录会话。',
    grantErrorMessage: '无法申请站点权限，请检查浏览器扩展设置。',
    cookieDisabledMessage: '已关闭 Cookie：下一次下载将使用匿名模式，可能因站点认证要求失败。',
    httpAnonymousMessage: 'HTTP 页面使用匿名模式；登录会话仅用于 HTTPS 页面。',
    initFailedMessage: '初始化失败，请重新打开扩展。',
  },
  en: {
    htmlLang: 'en',
    langToggleLabel: '中文',
    langToggleTitle: '切换到中文',
    brandSubtitle: 'Browser videos, saved locally.',
    currentPageSuffix: 'Current Page',
    readingVideo: 'Reading page…',
    invalidPageTitle: 'Open a page with video or audio',
    invalidPageUrl: 'Supports media sites recognized by yt-dlp, including YouTube, Bilibili, Niconico, X, and more',
    xHint: 'Downloads the first video by default, or the matching clip when the URL ends with /video/2, etc. X uses anonymous mode by default.',
    downloadTypeLabel: 'Download Mode',
    downloadTypes: {
      best: 'Video + Audio',
      audio: 'Audio Only',
      gif: 'GIF Clip · No Audio',
      subtitles: 'Subtitles Only',
    },
    subtitleDownload: 'Download Subtitles',
    subtitleLanguageLabel: 'Subtitle Language',
    subtitleLanguages: { 'zh-en': 'Chinese + English', zh: 'Chinese', en: 'English', ja: 'Japanese', all: 'All available languages' },
    subtitleFormatLabel: 'Subtitle Format',
    subtitleAutoLabel: 'Include site-generated captions',
    subtitleHint: 'Saves subtitles without the video. Prefers human subtitles; can include site-generated captions. Does not transcribe audio. Bilibili danmaku is not subtitles. All languages may produce many files.',
    qualityLabel: 'Max Quality',
    qualities: {
      best: 'Best available · No resolution cap',
      '4320': 'Up to 4320p · 8K',
      '2160': 'Up to 2160p · 4K',
      '1440': 'Up to 1440p · 2K',
      '1080': 'Up to 1080p · Full HD',
      '720': 'Up to 720p · HD',
      '480': 'Up to 480p',
      '360': 'Up to 360p',
      '240': 'Up to 240p',
      '144': 'Up to 144p',
    },
    qualityHint: 'Selects the best stream within your cap; actual quality depends on the source and low-res videos are never upscaled.',
    audioFormatLabel: 'Audio Format',
    audioFormats: {
      original: 'Original Audio · No re-encode (Recommended)',
      mp3: 'MP3 · Universal',
      m4a: 'M4A / AAC',
      flac: 'FLAC',
      wav: 'WAV',
    },
    audioQualityLabel: 'Conversion Quality / Target Bitrate',
    audioQualities: {
      best: 'Best conversion quality · Auto bitrate',
      '320': '320 kbps',
      '256': '256 kbps',
      '192': '192 kbps',
      '128': '128 kbps',
    },
    audioHints: {
      original: 'Keeps the original audio stream without re-encoding; file extension depends on the source.',
      lossless: 'Converting to FLAC / WAV increases file size and cannot restore detail lost in a lossy source.',
      m4a: 'Saves as M4A (converts to AAC if needed); conversion does not improve source quality.',
      mp3: 'Target bitrate for MP3 conversion; higher bitrate cannot restore detail missing from the source.',
    },
    gifStartLabel: 'Start Offset (s)',
    gifDurationLabel: 'Clip Duration',
    gifDurations: {
      '5': '5 sec',
      '10': '10 sec',
      '15': '15 sec',
      '30': '30 sec',
      '60': '60 sec',
    },
    gifWidthLabel: 'Max Width',
    gifFpsLabel: 'Frame Rate',
    gifHint: 'Downloads the source video first, then converts the selected clip (stops at the end if shorter). GIFs have no audio; longer or wider clips produce larger files.',
    concurrencyLabel: 'Concurrent Fragments',
    concurrencies: {
      '1': '1 · Sequential',
      '4': '4 · Recommended',
      '8': '8',
      '16': '16',
    },
    concurrencyHint: 'Applies only to DASH / HLS fragmented streams; plain HTTP files use a single connection. Higher concurrency is not always faster.',
    cookieHint: 'On: download with signed-in session · Off: anonymous download',
    checkingPermission: 'Checking permissions…',
    grant: 'Grant',
    revoke: 'Revoke',
    downloadButton: 'Download Current Media',
    hostUnchecked: 'Local Bridge · Not checked',
    hostConnected: 'Local Bridge · Connected',
    hostDisconnected: 'Local Bridge · Disconnected',
    checkDeps: 'Check & Install Dependencies',
    toolsDefault: 'Checks yt-dlp, FFmpeg, and the download folder. Missing tools are automatically downloaded and verified (internet required on first run). Local Host must be installed first.',
    updateChannelLabel: 'yt-dlp Update Channel',
    updateChannels: {
      stable: 'Stable',
      nightly: 'Nightly · Faster site fixes',
    },
    updateCheck: 'Check Update',
    updateInstall: 'Update yt-dlp',
    updateRollback: 'Rollback',
    updateStatusDefault: 'Check the selected channel first, then update if a newer version is available. Never downgrades automatically.',
    updateNote: 'Updates only switch the yt-dlp binary used by VideoBridge, preserving external tools and the previous version. FFmpeg is still provisioned via "Check & Install Dependencies".',
    backgroundNote: 'Downloads continue after closing the popup; quitting the browser stops active jobs.',
    scopeNote: 'Works on media pages supported by yt-dlp and attempts generic extraction elsewhere. Downloads one item per task; membership, region, and DRM limits still apply.',
    footerLeft: 'LOCAL PROCESSING · SINGLE ITEM',
    cancel: 'Cancel',
    downloadFailed: 'Download failed',
    downloadProgressAria: 'Download progress',
    taskCancelledDetail: 'Task stopped and temporary cookies cleaned up.',
    optionSaveFailed: 'Failed to save options; current selection will still be used for this download.',
    bridgeNoResponse: 'Local bridge did not respond.',
    grantedMessage: 'Site session granted and enabled. Turn off the session toggle to switch back to anonymous mode.',
    grantIncompleteMessage: 'Permission was not granted; cannot read signed-in session.',
    grantErrorMessage: 'Could not request site permission. Please check browser extension settings.',
    cookieDisabledMessage: 'Cookies disabled: next download will run anonymously and may fail if the site requires authentication.',
    httpAnonymousMessage: 'HTTP pages use anonymous mode; signed-in sessions require HTTPS.',
    initFailedMessage: 'Initialization failed. Please reopen the extension.',
  },
} as const;

export function formatPermissionState(site: Site, permitted: boolean, locale: Locale): string {
  const name = localizedSiteName(site, locale);
  if (locale === 'zh') return `${permitted ? '已授权' : '尚未授权'} ${name} Cookie`;
  return `${name} Cookie ${permitted ? 'granted' : 'not granted'}`;
}

export function formatCookieLabel(site: Site, locale: Locale): string {
  const name = localizedSiteName(site, locale);
  return locale === 'zh' ? `使用当前 ${name} 会话` : `Use current ${name} session`;
}

export function formatSiteLabel(site: Site, locale: Locale): string {
  const name = localizedSiteName(site, locale);
  return `${name} · ${UI_TEXT[locale].currentPageSuffix}`;
}

export function formatEta(seconds: number, locale: Locale): string {
  const ceil = Math.ceil(seconds);
  return locale === 'zh' ? ` · 剩余 ${ceil} 秒` : ` · ${ceil}s left`;
}

export function formatToolsHealth(
  health: { version?: string; outputDir: string; tools: { name: string; available: boolean; version?: string }[] },
  locale: Locale,
): string {
  if (locale === 'zh') {
    return (
      `Host：${health.version ?? '旧版本'}\n` +
      health.tools.map(t => `${t.available ? '✓' : '×'} ${t.name}: ${t.available ? t.version : '未找到或无法运行'}`).join('\n') +
      `\n下载目录：${health.outputDir}`
    );
  }
  return (
    `Host: ${health.version ?? 'Legacy'}\n` +
    health.tools.map(t => `${t.available ? '✓' : '×'} ${t.name}: ${t.available ? t.version : 'Not found or not runnable'}`).join('\n') +
    `\nDownload folder: ${health.outputDir}`
  );
}

export function formatUpdateStatus(
  update: { currentVersion: string; latestVersion: string; message: string },
  locale: Locale,
): string {
  const msg = translateDynamicText(update.message, locale);
  if (locale === 'zh') {
    return `当前：${update.currentVersion || '未找到'}${update.latestVersion ? ` · 最新：${update.latestVersion}` : ''}\n${msg}`;
  }
  return `Current: ${update.currentVersion || 'Not found'}${update.latestVersion ? ` · Latest: ${update.latestVersion}` : ''}\n${msg}`;
}

const EXACT_EN_MAP: Record<string, string> = {
  '下载字幕': 'Downloading subtitles',
  '准备下载': 'Preparing download',
  '字幕下载需要 v0.8 或更新的 Host，请安装新版后重新加载扩展。': 'Subtitle downloads require Host v0.8 or newer. Install the latest version and reload the extension.',
  '此视频没有所选语言的可用字幕；可尝试其他语言或开启自动字幕。Bilibili 弹幕不属于字幕。': 'No subtitles are available in the selected languages. Try another language or enable site-generated captions. Bilibili danmaku is not subtitles.',
  '无法验证字幕文件，请更新 yt-dlp 后重试。': 'Could not verify subtitle files. Update yt-dlp and retry.',
  '部分字幕未成功下载或转换，请重试或更换字幕格式。': 'Some subtitles could not be downloaded or converted. Retry or choose another format.',
  '字幕未转换为所选格式，请检查 FFmpeg 后重试。': 'Subtitles were not converted to the selected format. Check FFmpeg and retry.',
  '无法保存字幕，请检查下载目录和磁盘空间。': 'Could not save subtitles. Check the download folder and free disk space.',
  '无法创建下载临时目录。': 'Could not create a temporary download folder.',
  // Phases
  '正在启动下载…': 'Starting download…',
  '下载中': 'Downloading',
  '合并 / 后处理': 'Merging / Post-processing',
  '转换 GIF（无声音）': 'Converting GIF (no audio)',
  '已完成': 'Completed',
  '已取消': 'Cancelled',
  '下载失败': 'Download failed',
  // Setup & update progress
  '正在检测本地桥接…': 'Checking local bridge…',
  '正在自动补齐 yt-dlp / FFmpeg…': 'Automatically installing yt-dlp / FFmpeg…',
  '正在查询 yt-dlp 官方版本…': 'Checking official yt-dlp releases…',
  '查询 yt-dlp 官方版本…': 'Checking official yt-dlp releases…',
  '正在恢复上一版本…': 'Rolling back to previous version…',
  '正在更新 yt-dlp…': 'Updating yt-dlp…',
  '验证新版本能否运行…': 'Verifying new binary…',
  '获取 FFmpeg 校验值（FFmpeg.org 推荐的 Gyan 构建）…': 'Fetching FFmpeg checksums (Gyan build)…',
  '下载并校验 FFmpeg / FFprobe（约 115 MB）…': 'Downloading & verifying FFmpeg / FFprobe (~115 MB)…',
  '下载并校验 FFmpeg…': 'Downloading & verifying FFmpeg…',
  '解压 FFmpeg / FFprobe…': 'Extracting FFmpeg / FFprobe…',
  '检查已下载的工具…': 'Checking downloaded tools…',
  '本地依赖已就绪。': 'Local dependencies are ready.',
  '依赖安装失败，请重试。': 'Dependency installation failed. Please retry.',
  '工具操作失败，当前版本已保留。': 'Tool operation failed; current version kept.',
  // Update status messages
  '自动更新目前支持 Windows x64 和 macOS（Apple Silicon / Intel）。': 'Automatic updates currently support Windows x64 and macOS (Apple Silicon / Intel).',
  '无法比较当前自定义版本，请通过原安装方式更新。': 'Cannot compare custom build version; please update via your original installation method.',
  '当前版本比此通道的最新版更新，保持当前版本。': 'Current version is newer than this channel’s latest release; keeping current version.',
  '已是此通道最新版本。': 'Already up to date on this channel.',
  '发现新版本，可更新。': 'New version available.',
  '发现新版本': 'New version available',
  '已恢复上一版本。': 'Rolled back to previous version.',
  '已恢复上一版本': 'Rolled back to previous version',
  'yt-dlp 已更新；如遇兼容问题，可恢复上一版本。': 'yt-dlp updated; you can roll back if you encounter compatibility issues.',
  'yt-dlp 已更新': 'yt-dlp updated',
  // Common errors
  '本地桥接未响应。': 'Local bridge did not respond.',
  '本地桥接未连接。请检查 Native Host 安装、扩展 ID 和配置文件。': 'Local bridge disconnected. Check Native Host installation, Extension ID, and config file.',
  '本地桥接响应超时，请检查本地程序。': 'Local bridge timed out. Please check the local Host binary.',
  '本地桥接需要 v0.4 或更新版本才能自动安装 yt-dlp / FFmpeg。请运行最新安装脚本。': 'Auto-installing yt-dlp / FFmpeg requires Host v0.4 or newer. Please run the latest installer.',
  'yt-dlp 更新管理需要 v0.6 或更新的 Host，请安装新版并重新加载扩展。': 'yt-dlp update management requires Host v0.6 or newer. Please install the latest version and reload the extension.',
  'X / GIF 需要 v0.4 或更新的 Host，请完成本地更新后重新加载扩展。': 'X / GIF support requires Host v0.4 or newer. Please update the Host and reload the extension.',
  '其他媒体网站需要 v0.5 或更新的 Host，请安装新版后重新加载扩展。': 'Generic media site support requires Host v0.5 or newer. Please update the Host and reload the extension.',
  '请选择有效的更新通道。': 'Please select a valid update channel.',
  '请等待当前下载或工具操作完成。': 'Please wait for the current download or tool operation to finish.',
  '请等待当前任务完成或先取消。': 'Please wait for the current task to finish or cancel it first.',
  '已有任务正在执行。': 'Another task is already running.',
  '任务 ID 无效。': 'Invalid job ID.',
  '下载选项无效。': 'Invalid download options.',
  '不支持的下载选项。': 'Unsupported download options.',
  '无法读取当前标签页。': 'Cannot read the active tab.',
  'HTTP 页面仅支持匿名下载；请打开 HTTPS 页面后使用会话。': 'HTTP pages only support anonymous downloads; open an HTTPS page to use your session.',
  '不支持的请求。': 'Unsupported request.',
  '操作失败，请检查权限与本地安装。': 'Operation failed. Check permissions and local installation.',
  '检测到分区 Cookie；Netscape 格式无法保留分区信息，当前版本不能安全转换。': 'Partitioned cookies detected; Netscape format cannot preserve partition keys.',
  '未读取到 Niconico 登录凭据。请重新授权该站点，并确认当前浏览器已登录后重试。': 'Niconico login cookie (user_session) not found. Re-grant permission and ensure you are signed in.',
  '当前版本不支持隐身窗口。': 'Incognito windows are not supported in this version.',
  '请先授权当前站点 Cookie 访问。': 'Please grant cookie access for the current site first.',
  '无法唯一确定当前标签页的 Cookie Store，请重新打开视频页面。': 'Could not uniquely determine the Cookie Store for this tab. Please reopen the page.',
  '读取站点 Cookie 失败，请检查权限后重试。': 'Failed to read site cookies. Check permissions and retry.',
  '媒体受 DRM 保护，当前下载器不能下载。': 'This media is DRM-protected and cannot be downloaded.',
  'yt-dlp 未识别到可下载的媒体。请打开具体播放页面，或更新 yt-dlp 后重试。': 'yt-dlp did not find downloadable media. Open the specific watch page or update yt-dlp.',
  'YouTube 提取或 PO Token / JavaScript 运行时要求未满足；请更新 yt-dlp 并检查其官方运行时与 Token 支持。': 'YouTube extraction or PO Token / JS runtime requirements were not met; update yt-dlp and check upstream docs.',
  'yt-dlp 报告 Cookie 不可用或已过期，请重新登录当前站点后重试。': 'yt-dlp reported invalid or expired cookies. Sign in again in your browser and retry.',
  '站点未向下载器开放此会员或付费视频。若当前浏览器能完整播放，请重新授权该站点会话后重试；否则需具有该视频的访问权限。': 'Membership or paid access required. If the video plays in your browser, enable and re-authorize the site session; otherwise an eligible account is required.',
  '本次为匿名下载，站点要求会员或付费访问权限。请开启“使用当前站点会话”后重试；仅授予 Cookie 权限不会改变已关闭的会话开关。': 'Anonymous download rejected: membership or paid access required. Enable "Use current site session" and retry.',
  '站点要求登录、年龄验证或访问权限；Cookie 导出成功不等于具有该视频的权限。': 'Site requires sign-in, age verification, or access rights.',
  'FFmpeg / FFprobe 媒体处理失败，请检查本地版本与视频格式。': 'FFmpeg / FFprobe media processing failed. Check your local binaries and media format.',
  '网络请求失败，请检查网络、代理及站点限制。': 'Network request failed. Check your connection, proxy, and site availability.',
  '视频不可用，可能已移除或受到地区限制。': 'Video is unavailable (removed or region-restricted).',
  '找不到 yt-dlp，请安装并配置可执行文件路径。': 'yt-dlp not found. Please install it via Local Bridge or configure its path.',
  '找不到 FFmpeg 或同目录的 FFprobe，请安装并配置。': 'FFmpeg or co-located FFprobe not found. Please install via Local Bridge or configure its path.',
  'yt-dlp 执行失败。请检查版本、视频支持情况及本地工具配置。': 'yt-dlp execution failed. Check your yt-dlp version, site support, and local tool configuration.',
  'yt-dlp 未返回可验证的输出文件。': 'yt-dlp did not produce a verifiable output file.',
  'GIF 转换失败，请检查时间范围与 FFmpeg。': 'GIF conversion failed. Check clip time range and FFmpeg.',
  '所选时间范围没有可转换的视频帧。': 'No video frames found in the selected time range.',
  '无法查询 yt-dlp 官方版本。请检查网络或稍后重试；当前工具未改动。': 'Failed to check official yt-dlp releases. Check your network and retry; current tool was not modified.',
  '另一个工具安装或更新正在进行，请稍后重试。': 'Another tool installation or update is in progress. Please retry shortly.',
  '上一版本不存在或无法运行，当前配置已保留。': 'Previous version does not exist or is not runnable; current configuration kept.',
  '恢复上一版本失败，当前配置已保留。': 'Rollback failed; current configuration kept.',
  'yt-dlp 下载或 SHA256 校验失败，当前版本已保留。': 'yt-dlp download or SHA256 verification failed; current version kept.',
  '下载校验失败，当前版本已保留。': 'Download verification failed; current version kept.',
  '依赖下载或 SHA256 校验失败，原配置已保留。请检查网络后点“检测并补齐依赖”重试，或手动配置工具。': 'Dependency download or SHA256 verification failed; original config kept. Check your network and retry.',
};

const REVERSE_ZH_MAP: Record<string, string> = Object.fromEntries(
  Object.entries(EXACT_EN_MAP).map(([zh, en]) => [en, zh]),
);

export function translateDynamicText(text: string, locale: Locale): string {
  if (!text) return '';
  if (locale === 'zh') {
    if (REVERSE_ZH_MAP[text]) return REVERSE_ZH_MAP[text]!;
    const dlMatchEn = /^Downloading & verifying yt-dlp (.+)…$/.exec(text);
    if (dlMatchEn) return `下载并校验 yt-dlp ${dlMatchEn[1]}…`;
    const macMatchEn = /^Downloading & verifying macOS (.+) \(OSXExperts build\)…$/.exec(text);
    if (macMatchEn) return `下载并校验 macOS ${macMatchEn[1]}（OSXExperts 构建）…`;
    const noCookieEn = /^No usable (.+) session cookies found\. Sign in to this profile first, or disable session mode for public videos\.( \(COOKIE_UNAVAILABLE\))?$/.exec(text);
    if (noCookieEn) return `未发现 ${noCookieEn[1]} 可用会话 Cookie。请先在当前 Profile 登录，或关闭会话模式尝试公开视频。${noCookieEn[2] ?? ''}`;
    const codeSuffixEn = /^(.*) \(([A-Z0-9_]+)\)$/.exec(text);
    if (codeSuffixEn && REVERSE_ZH_MAP[codeSuffixEn[1]!]) {
      return `${REVERSE_ZH_MAP[codeSuffixEn[1]!]} (${codeSuffixEn[2]})`;
    }
    return text;
  }

  if (EXACT_EN_MAP[text]) return EXACT_EN_MAP[text]!;

  const dlMatch = /^下载并校验 yt-dlp (.+)…$/.exec(text);
  if (dlMatch) return `Downloading & verifying yt-dlp ${dlMatch[1]}…`;

  const macMatch = /^下载并校验 macOS (.+)（OSXExperts 构建）…$/.exec(text);
  if (macMatch) return `Downloading & verifying macOS ${macMatch[1]} (OSXExperts build)…`;

  const noCookieMatch = /^未发现 (.+) 可用会话 Cookie。请先在当前 Profile 登录，或关闭会话模式尝试公开视频。( \(COOKIE_UNAVAILABLE\))?$/.exec(text);
  if (noCookieMatch) {
    return `No usable ${noCookieMatch[1]} session cookies found. Sign in to this profile first, or disable session mode for public videos.${noCookieMatch[2] ?? ''}`;
  }

  const codeSuffix = /^(.*) \(([A-Z0-9_]+)\)$/.exec(text);
  if (codeSuffix && EXACT_EN_MAP[codeSuffix[1]!]) {
    return `${EXACT_EN_MAP[codeSuffix[1]!]} (${codeSuffix[2]})`;
  }

  return text;
}
