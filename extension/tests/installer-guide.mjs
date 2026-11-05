import assert from 'node:assert/strict';
import { cp, mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright-core';

const root = fileURLToPath(new URL('../../', import.meta.url));
const info = JSON.parse(await readFile(path.join(root, 'dist/installer-info.json'), 'utf8'));
const scratch = await mkdtemp(path.join(root, '.cache/installer-languages-'));
const customMessages = (await readFile(path.join(root, 'installer/VideoBridge.iss'), 'utf8')).split('[CustomMessages]')[1].split('[Tasks]')[0];
const englishKeys = [...customMessages.matchAll(/^english\.([^=]+)=/gm)].map(match => match[1]).sort();
const chineseKeys = [...customMessages.matchAll(/^chinesesimp\.([^=]+)=/gm)].map(match => match[1]).sort();
assert.ok(englishKeys.length >= 15);
assert.deepEqual(chineseKeys, englishKeys);

for (const channel of ['chrome', 'msedge']) {
  const browser = await chromium.launch({ channel, headless: true });
  try {
    for (const locale of ['en', 'zh']) {
      const installRoot = path.join(scratch, `VideoBridge 测试 # ${channel} ${locale}`);
      const setupRoot = path.join(installRoot, 'setup');
      await cp(path.join(info.payload, 'setup'), setupRoot, { recursive: true });
      await writeFile(path.join(setupRoot, 'install-language.js'), `window.VIDEOBRIDGE_INSTALL_LANGUAGE = '${locale}';`);
      const context = await browser.newContext({ locale: locale === 'en' ? 'zh-CN' : 'en-US', viewport: { width: 900, height: 800 } });
      const page = await context.newPage();
      const errors = [];
      page.on('pageerror', error => errors.push(error.message));
      await page.addInitScript(() => {
        globalThis.clipboardShouldFail = false;
        Object.defineProperty(navigator, 'clipboard', { value: { writeText: async text => {
          if (globalThis.clipboardShouldFail) throw new Error('Clipboard unavailable');
          globalThis.copiedPath = text;
        } } });
      });
      await page.goto(pathToFileURL(path.join(setupRoot, 'welcome.html')).href);
      assert.equal(await page.locator('html').getAttribute('lang'), locale === 'zh' ? 'zh-CN' : 'en');
      assert.equal(await page.locator('h1').textContent(), locale === 'zh' ? 'VideoBridge 安装说明' : 'VideoBridge Installation Guide');
      assert.equal(await page.locator('#path').textContent(), path.join(installRoot, 'extension'));
      await page.locator('#copy').click();
      assert.equal(await page.evaluate(() => globalThis.copiedPath), path.join(installRoot, 'extension'));
      assert.equal(await page.locator('#status').textContent(), locale === 'zh' ? '已复制。' : 'Copied.');
      await page.locator('#language').click();
      assert.equal(await page.locator('#status').textContent(), locale === 'en' ? '已复制。' : 'Copied.');
      await page.evaluate(() => { globalThis.clipboardShouldFail = true; });
      await page.locator('#copy').click();
      assert.equal(await page.locator('#status').textContent(), locale === 'en' ? '请选中上面的路径并复制。' : 'Select the folder path above and copy it manually.');
      await page.locator('#language').click();
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await mkdir(path.join(root, 'docs/screenshots'), { recursive: true });
      await page.screenshot({ path: path.join(root, `docs/screenshots/installer-guide-${locale}-${channel}.png`), fullPage: true });
      await page.setViewportSize({ width: 400, height: 760 });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      assert.deepEqual(errors, []);
      await context.close();
    }
    const context = await browser.newContext({ locale: 'en-US' });
    const page = await context.newPage();
    await page.goto(pathToFileURL(path.join(root, 'installer/welcome.html')).href);
    assert.equal(await page.locator('html').getAttribute('lang'), 'en');
    await context.close();
    console.log(`PASS ${channel}: guide follows installer language, language switching, Unicode folder copy, clipboard fallback and responsive layout`);
  } finally {
    await browser.close();
  }
}
