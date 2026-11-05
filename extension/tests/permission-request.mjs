import { chromium } from 'playwright-core';
import assert from 'node:assert/strict';
import { build } from 'esbuild';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root=fileURLToPath(new URL('../../',import.meta.url));
const scratch=await mkdtemp(path.join(root,'.cache','permission-request-'));
const extensionDir=path.join(scratch,'extension');
await mkdir(extensionDir);
const manifest=JSON.parse(await readFile(path.join(root,'extension','.output','chrome-mv3','manifest.json'),'utf8'));
delete manifest.background;
await writeFile(path.join(extensionDir,'manifest.json'),JSON.stringify(manifest));
await writeFile(path.join(extensionDir,'popup.html'),'<button id="request">Request</button><output id="result">idle</output><script src="request.js"></script>');
await build({stdin:{resolveDir:path.join(root,'extension'),contents:`import { sitePermission } from './lib/site-resolver';
document.querySelector('#request').onclick=()=>{
 const origins=location.hash==='#production'?sitePermission('example.org').origins:['*://*.example.org/*'];
 document.querySelector('#result').textContent='pending';
 chrome.permissions.request({permissions:['cookies'],origins}).then(granted=>document.querySelector('#result').textContent='granted='+granted).catch(e=>document.querySelector('#result').textContent=e.message);
};`},bundle:true,format:'iife',outfile:path.join(extensionDir,'request.js')});
const browser=process.env.VB_BROWSER??'chrome';
const context=await chromium.launchPersistentContext(path.join(scratch,'profile'),{
 executablePath:browser==='chrome'?'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe':'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',headless:true,
 ignoreDefaultArgs:['--disable-extensions'],args:['--enable-unsafe-extension-debugging'],
});
try {
 const cdp=await context.browser().newBrowserCDPSession();
 const {id}=await cdp.send('Extensions.loadUnpacked',{path:extensionDir});
 const page=await context.newPage();
 for(const hash of ['wildcard','production']) {
  await page.goto(`chrome-extension://${id}/popup.html#${hash}`);
  await page.locator('#request').click();
  await page.waitForFunction(()=>document.querySelector('#result').textContent!=='pending',{},{timeout:2000}).catch(()=>{});
  const result=await page.locator('#result').textContent();
  if(hash==='wildcard') assert.match(result,/Only permissions specified in the manifest/);
  else assert.ok(['pending','granted=true','granted=false'].includes(result),result);
  console.log(`PASS ${browser} ${hash}: ${result}`);
 }
} finally {await context.close();}
