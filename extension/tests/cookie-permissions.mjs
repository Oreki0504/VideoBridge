// Isolated Chrome/Edge with synthetic cookies and no external website requests.
import assert from 'node:assert/strict';
import { chromium } from 'playwright-core';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root=fileURLToPath(new URL('../../',import.meta.url));
const productionManifest=JSON.parse(await readFile(path.join(root,'extension','.output','chrome-mv3','manifest.json'),'utf8'));
assert.deepEqual(productionManifest.optional_host_permissions,['http://*/*','https://*/*']);
assert.equal(productionManifest.host_permissions,undefined);
for (const bothSchemes of [false,true]) {
const scratch=await mkdtemp(path.join(root,'.cache','cookie-permissions-'));
const extensionDir=path.join(scratch,'extension');
await mkdir(extensionDir);
await writeFile(path.join(extensionDir,'manifest.json'),JSON.stringify({
 manifest_version:3,name:'Synthetic cookie permission test',version:'1.0',
 permissions:['cookies'],host_permissions:bothSchemes?['http://*.example.org/*','https://*.example.org/*']:['https://*.example.org/*'],
 background:{service_worker:'background.js'},
}));
await writeFile(path.join(extensionDir,'background.js'),'chrome.runtime.onInstalled.addListener(() => {});');
const browser=process.env.VB_BROWSER??'chrome';
const context=await chromium.launchPersistentContext(path.join(scratch,'profile'),{
 executablePath:browser==='chrome'?'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe':'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
 headless:true,ignoreDefaultArgs:['--disable-extensions'],args:['--enable-unsafe-extension-debugging'],
});
try {
 await context.route('**/*',route=>route.abort());
 await context.addCookies([
  {name:'user_session',value:'synthetic-only',domain:'.example.org',path:'/',secure:false,httpOnly:true,sameSite:'Lax'},
  {name:'secure_marker',value:'synthetic-only',domain:'.example.org',path:'/',secure:true,httpOnly:true,sameSite:'Lax'},
  {name:'foreign',value:'synthetic-only',domain:'.unrelated.example',path:'/',secure:false,httpOnly:true,sameSite:'Lax'},
 ]);
 const cdp=await context.browser().newBrowserCDPSession();
 const {id}=await cdp.send('Extensions.loadUnpacked',{path:extensionDir});
 const worker=context.serviceWorkers().find(w=>w.url().includes(id))??await context.waitForEvent('serviceworker');
 const result=await worker.evaluate(async()=>({
  completePermission:await chrome.permissions.contains({permissions:['cookies'],origins:['http://*.example.org/*','https://*.example.org/*']}),
  domain:(await chrome.cookies.getAll({domain:'example.org'})).map(c=>c.name).sort(),
  url:(await chrome.cookies.getAll({url:'https://www.example.org/'})).map(c=>c.name).sort(),
  visible:(await chrome.cookies.getAll({})).map(c=>({name:c.name,secure:c.secure,httpOnly:c.httpOnly})),
 }));
 const expected=bothSchemes?['secure_marker','user_session']:['secure_marker'];
 assert.equal(result.completePermission,bothSchemes);
 assert.deepEqual(result.domain,expected);
 assert.deepEqual(result.url,expected);
 assert.deepEqual(result.visible.map(c=>c.name).sort(),expected);
 if (bothSchemes) assert.deepEqual(result.visible.find(c=>c.name==='user_session'),{name:'user_session',secure:false,httpOnly:true});
 console.log(`PASS ${browser} ${context.browser().version()}: ${bothSchemes?'HTTP + HTTPS includes the non-Secure session':'HTTPS-only reproduces the missing session'}; unrelated site cookies remain inaccessible`);
} finally { await context.close(); }
}
