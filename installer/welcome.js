const messages = {
  en: {
    title: 'VideoBridge · Installation Guide',
    heading: 'VideoBridge Installation Guide',
    tagline: 'Videos, audio, GIFs and subtitles, saved locally.',
    loadHeading: 'Next: load the browser extension',
    openChrome: 'In Chrome, open',
    openEdge: '; in Edge, open',
    developerMode: 'Enable Developer mode and select Load unpacked.',
    selectFolder: 'Select the installed folder below:',
    copy: 'Copy extension folder',
    copied: 'Copied.',
    copyFallback: 'Select the folder path above and copy it manually.',
    folderNote: 'Setup has prepared this folder; no manual extraction or copying is needed. VideoBridge is not yet available in the Chrome Web Store, so the extension must be loaded manually once.',
    downloadHeading: 'Start downloading',
    download: 'Open a YouTube, Bilibili, Niconico or other media page and click VideoBridge. Choose video, audio, GIF or subtitles. For content requiring login, grant site access and enable Use current site session. Availability depends on yt-dlp and the site\'s access rules.',
    tools: 'If tools are missing, open Local Bridge and select Check & Install Dependencies. If Setup reported a failure, run Setup again after resolving it.',
    popupLanguage: 'The popup follows your browser language on first use. Use its EN / 中文 button to switch languages; your choice is saved.',
    upgradeHeading: 'Upgrade or move an existing installation',
    migration: 'If you previously loaded VideoBridge from a project or extracted folder, remove that old extension before loading the installed folder above. Downloads and local tool settings are preserved. Grant site permissions again for the new extension.',
    upgrade: 'For future upgrades, run the new installer and reload the extension. Uninstall from Windows Installed Apps or the VideoBridge Start menu folder. Downloaded files, settings and managed tools are kept; remove the browser extension separately.',
    footer: 'Program files are stored together in your VideoBridge installation folder. Existing download folders are preserved; new installations default to Downloads\\VideoBridge in your user profile.',
  },
  zh: {
    title: 'VideoBridge · 安装说明',
    heading: 'VideoBridge 安装说明',
    tagline: '视频、音频、GIF 与字幕，保存到本地。',
    loadHeading: '下一步：在浏览器中加载扩展',
    openChrome: '打开 Chrome 的',
    openEdge: '；Edge 使用',
    developerMode: '开启“开发者模式”，点击“加载已解压的扩展程序”。',
    selectFolder: '选择下面的固定目录：',
    copy: '复制扩展目录',
    copied: '已复制。',
    copyFallback: '请选中上面的路径并复制。',
    folderNote: '目录已经由安装程序准备好，不用自己解压或复制文件。Chrome 商店版尚未发布，因此首次仍需手动加载一次。',
    downloadHeading: '开始下载',
    download: '打开 YouTube、Bilibili、Niconico 或其他媒体页面，点击 VideoBridge。选择视频、音频、GIF 或仅字幕。需要登录时，请授权站点并开启“使用当前站点会话”。支持范围取决于 yt-dlp 和站点的访问规则。',
    tools: '如果提示缺少工具，点击“本地桥接”→“检测并补齐依赖”。若安装器报告失败，请解决问题后重新运行安装器。',
    popupLanguage: '扩展首次使用时跟随浏览器语言，之后可用右上角 EN / 中文 按钮切换并记住选择。',
    upgradeHeading: '升级或迁移旧版',
    migration: '如果之前从项目或解压目录加载过 VideoBridge，先在扩展管理页移除旧扩展，再加载上面的固定目录。下载文件和本地工具配置会保留，新扩展需重新授予站点权限。',
    upgrade: '以后升级只需运行新安装程序，再重新加载扩展。卸载入口位于 Windows“已安装的应用”和开始菜单 VideoBridge；卸载保留下载文件、设置及已下载依赖。浏览器扩展需单独移除。',
    footer: '程序组件集中在 VideoBridge 安装目录。下载文件保存在原有目录；新安装默认使用用户目录下的 Downloads\\VideoBridge。',
  },
};

const browserLocale = navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en';
const installedLocale = window.VIDEOBRIDGE_INSTALL_LANGUAGE;
let locale = installedLocale === 'en' || installedLocale === 'zh' ? installedLocale : browserLocale;
let theme = 'dark';
let statusKey = '';
const extensionURL = new URL('../extension/', location.href);
const directory = decodeURIComponent(extensionURL.pathname).replace(/^\//, '').replaceAll('/', '\\').replace(/\\$/, '');
document.querySelector('#path').textContent = directory;

function render() {
  document.documentElement.lang = locale === 'zh' ? 'zh-CN' : 'en';
  document.title = messages[locale].title;
  for (const element of document.querySelectorAll('[data-text]')) {
    element.textContent = messages[locale][element.dataset.text];
  }
  document.documentElement.dataset.theme = theme;
  const themeButton = document.querySelector('#theme');
  if (themeButton) {
    const isLight = theme === 'light';
    themeButton.textContent = locale === 'zh' ? (isLight ? '☾ 深色' : '☀ 浅色') : (isLight ? '☾ Dark' : '☀ Light');
    themeButton.setAttribute('aria-label', locale === 'zh' ? (isLight ? '切换到深色模式' : '切换到浅色模式') : (isLight ? 'Switch to dark mode' : 'Switch to light mode'));
  }
  const languageButton = document.querySelector('#language');
  languageButton.textContent = locale === 'en' ? '简体中文' : 'English';
  languageButton.setAttribute('aria-label', locale === 'en' ? '切换到中文' : 'Switch to English');
  document.querySelector('#status').textContent = statusKey ? messages[locale][statusKey] : '';
}

const themeToggle = document.querySelector('#theme');
if (themeToggle) {
  themeToggle.onclick = () => {
    theme = theme === 'dark' ? 'light' : 'dark';
    render();
  };
}
document.querySelector('#language').onclick = () => {
  locale = locale === 'en' ? 'zh' : 'en';
  render();
};
document.querySelector('#copy').onclick = async () => {
  try {
    await navigator.clipboard.writeText(directory);
    statusKey = 'copied';
  } catch {
    statusKey = 'copyFallback';
  }
  render();
};
render();
