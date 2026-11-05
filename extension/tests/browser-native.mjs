// Real browser extension APIs and Native Messaging, isolated disposable profiles.
// Cookie cases use synthetic values; a separately selected case uses real yt-dlp anonymously.
import assert from 'node:assert/strict';
import { chromium } from 'playwright-core';
import { cp, mkdir, mkdtemp, readFile, writeFile, readdir } from 'node:fs/promises';
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root=fileURLToPath(new URL('../../',import.meta.url));
const channel=process.env.VB_BROWSER??'chrome';
const real=process.env.VB_REAL_DOWNLOAD==='1';
const site=process.env.VB_TEST_SITE??'youtube';
const isX=site==='x';
const isOther=site!=='youtube';
const mediaURL=({youtube:'https://www.youtube.com/watch?v=jNQXAC9IVRw',x:'https://x.com/i/web/status/1600649710662213632/video/2',bilibili:'https://www.bilibili.com/video/BV13x41117TL',niconico:'https://www.nicovideo.jp/watch/sm9'})[site];
const mediaOrigin=new URL(mediaURL).origin;
const interactive=process.env.VB_INTERACTIVE_LOGIN==='1';
const testRoot=await mkdtemp(path.join(root,'.cache','native-e2e-'));
const extensionDir=path.join(testRoot,'extension');
const localAppData=path.join(testRoot,'local-app-data');
const outputDir=path.join(testRoot,'downloads');
await mkdir(localAppData,{recursive:true});await mkdir(outputDir,{recursive:true});
await cp(path.join(root,'extension','.output','chrome-mv3'),extensionDir,{recursive:true});
// A manifest-only test variant grants exactly the optional permissions being exercised.
// Permission prompts are tested separately with the unmodified production manifest.
if(!real){
 const manifest=JSON.parse(await readFile(path.join(extensionDir,'manifest.json'),'utf8'));
 manifest.permissions.push('cookies');manifest.host_permissions=manifest.optional_host_permissions;
 delete manifest.optional_permissions;delete manifest.optional_host_permissions;
 await writeFile(path.join(extensionDir,'manifest.json'),JSON.stringify(manifest));
}
const executablePath=channel==='chrome'?'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe':'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe';
const context=await chromium.launchPersistentContext(path.join(testRoot,'profile'),{
 executablePath,headless:!interactive,ignoreDefaultArgs:['--disable-extensions'],args:['--enable-unsafe-extension-debugging'],
 env:{...process.env,LOCALAPPDATA:localAppData,VB_FIXTURE_MODE:'slow'},
});
const report={browser:channel,version:context.browser().version(),site,mode:interactive?'real-signed-in-ytdlp':real?'real-anonymous-ytdlp':'real-browser-synthetic-cookie',checks:[],outputDir};
const reportName=`native-${channel}${isOther?'-'+site+'-gif':''}${interactive?'-signed-in':real?'-real':''}.json`;
const check=(name)=>{report.checks.push(name);console.log('PASS',channel,name)};
let serial=0;
async function waitFor(fn,label,timeout=15000){
 const until=Date.now()+timeout;
 while(Date.now()<until){const value=await fn();if(value)return value;await new Promise(r=>setTimeout(r,100));}
 throw new Error(`Timed out: ${label}`);
}
try{
 const browserCDP=await context.browser().newBrowserCDPSession();
 const {id}=await browserCDP.send('Extensions.loadUnpacked',{path:extensionDir});
 const worker=context.serviceWorkers().find(w=>w.url().includes(id))??await context.waitForEvent('serviceworker');
 const fixture=path.join(root,'.cache','native-fixture','yt-dlp.exe');
 execFileSync('powershell.exe',['-NoProfile','-File',path.join(root,'scripts','Install-Host.ps1'),'-ExtensionId',id,'-Browser',channel==='chrome'?'Chrome':'Edge','-InstallDir',path.join(testRoot,'host'),'-OutputDir',outputDir,'-YtDlpPath',real?process.env.VB_YTDLP:fixture,'-FFmpegPath',real?process.env.VB_FFMPEG:path.join(root,'.cache','native-fixture','ffmpeg.exe')],{windowsHide:true,stdio:'pipe'});
 check('unpacked extension load and installer registration');
 const tab=context.pages()[0];
 if(!interactive) await context.route(mediaOrigin+'/**',route=>route.fulfill({contentType:'text/html',body:'<title>VideoBridge isolated test - YouTube</title><h1>Isolated browser test fixture</h1>'}));
 await tab.goto(mediaURL,{waitUntil:'domcontentloaded',timeout:60000});
 if(real && !interactive) await worker.evaluate(()=>chrome.storage.local.set({useCookies:false,useCookiesX:false}));
 else if(!real) await context.addCookies([{name:'SID',value:'synthetic-e2e-not-a-login',domain:'.youtube.com',path:'/',httpOnly:true,secure:true,sameSite:'Lax'}]);
 let popupTarget;
 async function openPopup(){
  const {targetInfos}=await browserCDP.send('Target.getTargets',{filter:[{type:'tab'}]});
  const active=targetInfos.find(t=>t.url.startsWith(mediaOrigin+'/'));
  await browserCDP.send('Extensions.triggerAction',{id,targetId:active.targetId});
  popupTarget=await waitFor(async()=> (await browserCDP.send('Target.getTargets')).targetInfos.find(t=>t.url===`chrome-extension://${id}/popup.html`),'popup target');
  const {sessionId}=await browserCDP.send('Target.attachToTarget',{targetId:popupTarget.targetId,flatten:false});
  const pending=new Map();
  const listener=event=>{if(event.sessionId!==sessionId)return;const msg=JSON.parse(event.message);const entry=pending.get(msg.id);if(entry){pending.delete(msg.id);clearTimeout(entry.timer);if(msg.error)entry.reject(new Error(msg.error.message));else entry.resolve(msg.result);}};
  browserCDP.on('Target.receivedMessageFromTarget',listener);
  const send=(method,params={})=>new Promise((resolve,reject)=>{
   const reqId=++serial;
   const timer=setTimeout(()=>{pending.delete(reqId);reject(new Error(`CDP timeout ${method}`))},25000);
   pending.set(reqId,{resolve,reject,timer});
   browserCDP.send('Target.sendMessageToTarget',{sessionId,message:JSON.stringify({id:reqId,method,params})}).catch(reject);
  });
  const evaluate=async expression=>{
   const result=await send('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true,userGesture:true});
   if(result.exceptionDetails)throw new Error(result.exceptionDetails.text+' '+(result.exceptionDetails.exception?.description??''));
   return result.result?.value;
  };
  await waitFor(async()=>evaluate('document.readyState === "complete" && document.getElementById("permission-state").textContent !== "正在检查授权…"'),'popup initialization');
  return {evaluate,send,close:async()=>{browserCDP.off('Target.receivedMessageFromTarget',listener);await browserCDP.send('Target.closeTarget',{targetId:popupTarget.targetId});}};
 }
 let popup=await openPopup();
 if(interactive){
  const already=await popup.evaluate('chrome.permissions.contains({permissions:["cookies"],origins:["http://*.youtube.com/*","https://*.youtube.com/*"]})');
  if(!already){
   await popup.evaluate('document.getElementById("grant").click(); true');
   console.log('WAITING FOR USER: accept the YouTube Cookie permission prompt, then sign in to YouTube in this isolated Chrome window.');
   await waitFor(()=>worker.evaluate(()=>chrome.permissions.contains({permissions:['cookies'],origins:['http://*.youtube.com/*','https://*.youtube.com/*']})),'user Cookie permission approval',15*60*1000);
  }
  await popup.close();
  console.log('WAITING FOR USER: sign in to YouTube. Credentials stay in the isolated browser; no Cookie values are printed.');
  // Only a Boolean login marker leaves the browser during this waiting stage.
  await waitFor(()=>worker.evaluate(async()=>{
   const cookies=await chrome.cookies.getAll({domain:'youtube.com'});
   return cookies.some(c=>/^(SID|HSID|SSID|SAPISID|APISID|__Secure-[13]P(SID|APISID))$/.test(c.name)&&c.value&&c.value!=='synthetic-e2e-not-a-login');
  }),'YouTube sign-in',20*60*1000);
  console.log('YouTube session detected; starting authorized download test.');
  await tab.goto('https://www.youtube.com/watch?v=jNQXAC9IVRw',{waitUntil:'domcontentloaded',timeout:60000});
  popup=await openPopup();
 }
 check('actual toolbar popup and activeTab access');
 const title=await popup.evaluate('document.getElementById("video-title").textContent');
 if(!interactive) assert.equal(title,'VideoBridge isolated test');
 const initial=await popup.evaluate('chrome.runtime.sendMessage({type:"state.get"})');
 assert.equal(initial.ok,true);
 check('popup to service worker messages');
 const status=await popup.evaluate('chrome.runtime.sendMessage({type:"host.check"})');
 assert.equal(status.ok,true,JSON.stringify(status));
 assert.equal(status.data.health.tools.every(t=>t.available),true);
 assert.equal(status.data.health.version,'0.8.0');
 assert.ok(status.data.health.capabilities.includes('gif'));
 report.hostVersion=status.data.health.version;
 check('browser launches real Go Host and tool diagnostics');
 if(!real || interactive){
  const summary=await worker.evaluate(async()=>{
   const [tab]=await chrome.tabs.query({active:true,currentWindow:true});
   const stores=await chrome.cookies.getAllCookieStores();
   const store=stores.find(s=>s.tabIds.includes(tab.id));
   const cookies=await chrome.cookies.getAll({domain:'youtube.com',storeId:store.id});
   return {storeMatched:!!store,count:cookies.length,hasHttpOnly:cookies.some(c=>c.httpOnly),partitioned:cookies.some(c=>!!c.partitionKey)};
  });
  assert.equal(summary.storeMatched,true);if(!interactive)assert.equal(summary.count,1);assert.ok(summary.count>0);assert.equal(summary.hasHttpOnly,true);
  check('real cookies API, tab Store and HttpOnly cookie');
 }
 await popup.evaluate(isOther?'document.getElementById("download-type").value="gif";document.getElementById("gif-duration").value="5";document.getElementById("gif-width").value="320";document.getElementById("download").click()':'document.getElementById("quality").value="2160";document.getElementById("download").click()');
 const getState=async()=>popup.evaluate('chrome.runtime.sendMessage({type:"state.get"})');
 let first=await waitFor(async()=>{const s=await getState();return s.data?.jobs[0]},'accepted job');
 assert.equal(first.status,'running',JSON.stringify(first));
 check(isOther?site+' GIF 5 second request accepted through actual popup':'4K cap accepted through actual popup');
 if(!real){
  const cookieRoot=path.join(localAppData,'VideoBridge','cookie-tasks');
  const dirs=await readdir(cookieRoot);assert.equal(dirs.length,1);
  await popup.close();
  await new Promise(resolve=>setTimeout(resolve,6000));
  popup=await openPopup();
  const completed=(await getState()).data.jobs[0];
  assert.equal(completed.status,'completed',JSON.stringify(completed));
  assert.equal((await readdir(cookieRoot)).length,0);
  check('closing popup preserves download, completion cleans cookie file');
  await popup.evaluate('document.getElementById("download-type").value="audio";document.getElementById("audio-format").value="mp3";document.getElementById("audio-quality").value="320";document.getElementById("download").click()');
  const second=await waitFor(async()=>{const s=await getState();return s.data?.jobs[1]},'second job');
  assert.equal(second.status,'running');
  await popup.evaluate(`chrome.runtime.sendMessage({type:"download.cancel",jobId:${JSON.stringify(second.jobId)}})`);
  await waitFor(async()=>{const s=await getState();return s.data.jobs[1].status==='cancelled'},'cancelled');
  assert.equal((await readdir(cookieRoot)).length,0);
  check('MP3 320 kbps request accepted, cancel stops task and clears cookie file');
  await context.clearCookies();
  const denied=await popup.evaluate('chrome.runtime.sendMessage({type:"download.start",options:{format:"best"}})');
  assert.equal(denied.ok,false);assert.equal(denied.code,'COOKIE_UNAVAILABLE');
  check('fresh cookie read rejects removed login cookie');
 }else{
  const terminal=await waitFor(async()=>{const s=await getState();const j=s.data.jobs[0];return j.status!=='running'?j:null},'real video download',180000);
  report.download=terminal;
  if(terminal.status==='completed'){
   const media=JSON.parse(execFileSync(path.join(path.dirname(process.env.VB_FFMPEG),'ffprobe.exe'),['-v','error','-show_entries','format=duration,size:stream=codec_type,codec_name,width,height,nb_frames','-of','json',terminal.files[0]],{encoding:'utf8',windowsHide:true}));
   assert.ok(Number(media.format.duration)>0);assert.ok(media.streams.some(s=>s.codec_type==='video'));
   if(isOther){
    assert.equal(media.streams.length,1);assert.equal(media.streams[0].codec_name,'gif');assert.equal(media.streams[0].width,320);assert.equal(Number(media.streams[0].nb_frames),60);assert.ok(Math.abs(Number(media.format.duration)-5)<0.1);
    assert.equal((await readdir(outputDir)).length,1);
    assert.equal((await readdir(path.join(localAppData,'VideoBridge','cookie-tasks'))).length,0);
    execFileSync(process.env.VB_FFMPEG,['-v','error','-i',terminal.files[0],'-f','null','-'],{windowsHide:true,stdio:'pipe'});
   }else{assert.ok(media.streams.some(s=>s.codec_type==='audio'));}
   report.media=media;check(isOther?'real '+site+' video to GIF, 60 frames / 5 seconds, full decode and temporary file cleanup':'real YouTube download with verified audio/video streams');
  }else{console.log('REAL DOWNLOAD RESULT',terminal.code,terminal.error);if(isOther)throw new Error(`X GIF download failed: ${terminal.code}`);}
 }
 await popup.close();
}catch(error){report.error=error.message;throw error;}
finally{
 await context.close();
 await mkdir(path.join(root,'docs','test-results'),{recursive:true});
 await writeFile(path.join(root,'docs','test-results',reportName),JSON.stringify(report,null,2));
 console.log('Report',path.join(root,'docs','test-results',reportName));
}
