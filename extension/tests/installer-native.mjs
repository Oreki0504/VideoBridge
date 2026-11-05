import assert from 'node:assert/strict';
import { chromium } from 'playwright-core';
import { readFile, mkdir } from 'node:fs/promises';
import path from 'node:path';
const root=process.env.VB_INSTALLER_TEST_ROOT;
assert.ok(root && path.isAbsolute(root));
const scratch=process.env.VB_INSTALLER_TEST_SCRATCH??root;
const info=JSON.parse(await readFile(path.join(root,'setup','package-info.json'),'utf8'));
async function waitFor(fn,label,timeout=15000){
 const until=Date.now()+timeout;
 while(Date.now()<until){if(await fn())return;await new Promise(r=>setTimeout(r,100));}
 throw new Error(`Timed out: ${label}`);
}
await mkdir(path.join(scratch,'test-local-app-data'),{recursive:true});
const context=await chromium.launchPersistentContext(path.join(scratch,'test-browser-profile'),{
  executablePath:'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',headless:true,
  ignoreDefaultArgs:['--disable-extensions'],args:['--enable-unsafe-extension-debugging'],
  env:{...process.env,LOCALAPPDATA:path.join(scratch,'test-local-app-data')},
});
try{
 const cdp=await context.browser().newBrowserCDPSession();
 const {id}=await cdp.send('Extensions.loadUnpacked',{path:path.join(root,'extension')});
 assert.equal(id,info.extensionId);
 const worker=context.serviceWorkers().find(w=>w.url().includes(id))??await context.waitForEvent('serviceworker');
 const health=await worker.evaluate(()=>new Promise((resolve,reject)=>{
  const port=chrome.runtime.connectNative('com.videobridge.host');
  const timeout=setTimeout(()=>{port.disconnect();reject(new Error('Native status timed out'));},15000);
  port.onMessage.addListener(e=>{if(e.type==='host.status'){clearTimeout(timeout);resolve(e.payload);port.disconnect();}});
  port.onDisconnect.addListener(()=>{clearTimeout(timeout);if(chrome.runtime.lastError)reject(new Error(chrome.runtime.lastError.message));});
  port.postMessage({version:1,id:'installer-check',type:'host.status',payload:{}});
 }));
 assert.equal(health.version,info.version);assert.ok(health.tools.every(t=>t.available));
 if(process.env.VB_INSTALLER_TEST_UPDATE_CHECK==='1'){
  const configFile=path.join(root,'host','config.json');
  const configBefore=await readFile(configFile);
  assert.ok(health.capabilities.includes('ytdlp.update'));
  // A normal extension tab is intentionally rejected by the background sender
  // check. Exercise the real toolbar popup, as users do.
  const tab=context.pages()[0];
  await context.route('https://example.com/**',route=>route.fulfill({contentType:'text/html',body:'<title>VideoBridge isolated updater test</title>'}));
  await tab.goto('https://example.com/videobridge-test');
  const targets=await cdp.send('Target.getTargets',{filter:[{type:'tab'}]});
  const active=targets.targetInfos.find(t=>t.url==='https://example.com/videobridge-test');
  await cdp.send('Extensions.triggerAction',{id,targetId:active.targetId});
  let popupTarget;
  await waitFor(async()=>popupTarget=(await cdp.send('Target.getTargets')).targetInfos.find(t=>t.url===`chrome-extension://${id}/popup.html`),'toolbar popup');
  const {sessionId}=await cdp.send('Target.attachToTarget',{targetId:popupTarget.targetId,flatten:false});
  let serial=0;
  const pending=new Map();
  const listener=e=>{
   if(e.sessionId!==sessionId)return;
   const m=JSON.parse(e.message),entry=pending.get(m.id);
   if(entry){pending.delete(m.id);clearTimeout(entry.timer);m.error?entry.reject(new Error(m.error.message)):entry.resolve(m.result);}
  };
  cdp.on('Target.receivedMessageFromTarget',listener);
  const evaluate=expression=>new Promise((resolve,reject)=>{
   const requestId=++serial;
   const timer=setTimeout(()=>{pending.delete(requestId);reject(new Error('popup evaluation timed out'));},15000);
   pending.set(requestId,{timer,reject,resolve:result=>{
    if(result.exceptionDetails)reject(new Error(JSON.stringify(result.exceptionDetails)));
    else resolve(result.result?.value);
   }});
   cdp.send('Target.sendMessageToTarget',{sessionId,message:JSON.stringify({id:requestId,method:'Runtime.evaluate',params:{expression,returnByValue:true,awaitPromise:true,userGesture:true}})}).catch(reject);
  });
  await waitFor(()=>evaluate('document.readyState === "complete" && document.getElementById("permission-state").textContent !== "正在检查授权…"'),'popup initialization');
  await evaluate('document.querySelector("details").open=true');
  for(const channel of ['stable','nightly']){
   await evaluate(`document.getElementById('update-channel').value=${JSON.stringify(channel)};document.getElementById('update-channel').dispatchEvent(new Event('change'));document.getElementById('update-check').click()`);
   await waitFor(()=>evaluate('document.getElementById("update-status").textContent.includes("最新：") || !document.getElementById("message").hidden'),'official update check',65000);
   assert.equal(await evaluate('document.getElementById("message").hidden'),true,await evaluate('document.getElementById("message").textContent'));
   assert.equal(await evaluate('document.getElementById("update-install").disabled'),false);
   assert.equal(await evaluate('document.querySelector(".version").textContent'),info.version);
   console.log(`PASS installed updater ${channel}: ${await evaluate('document.getElementById("update-status").textContent')}`);
  }
  assert.deepEqual(await readFile(configFile),configBefore,'read-only checks changed config');
  assert.equal(await worker.evaluate(()=>chrome.permissions.contains({permissions:['cookies']})),false);
  cdp.off('Target.receivedMessageFromTarget',listener);
  await cdp.send('Target.closeTarget',{targetId:popupTarget.targetId});
 }
 const page=await context.newPage();
 await page.goto('file:///'+path.join(root,'setup','welcome.html').replaceAll('\\','/'));
 assert.equal(await page.locator('#path').textContent(),path.join(root,'extension'));
 assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
 await page.screenshot({path:path.join(scratch,'welcome-test.png'),fullPage:true});
 console.log(`PASS installer: stable extension ID ${id}, real Chrome Native Messaging ${health.version}, Chinese-path welcome guide`);
}finally{await context.close();}
