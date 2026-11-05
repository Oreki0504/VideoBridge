[CmdletBinding()]
param(
    [ValidateSet('chrome','msedge')][string]$Browser = 'chrome',
    [switch]$RealDownload,
    [ValidateSet('youtube','x','bilibili','niconico')][string]$Site = 'youtube',
    [switch]$InteractiveLogin,
    [string]$YtDlpPath = '',
    [string]$FFmpegPath = ''
)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$registry=if($Browser -eq 'chrome'){'HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.videobridge.host'}else{'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.videobridge.host'}
$existed=Test-Path -LiteralPath $registry
$oldValue=if($existed){(Get-Item -LiteralPath $registry).GetValue('')}else{$null}
$env:VB_BROWSER=$Browser
$env:VB_REAL_DOWNLOAD=if($RealDownload){'1'}else{'0'}
$env:VB_TEST_SITE=$Site
$env:VB_INTERACTIVE_LOGIN=if($InteractiveLogin){'1'}else{'0'}
if($InteractiveLogin -and -not $RealDownload){throw 'InteractiveLogin requires RealDownload.'}
if($Site -ne 'youtube' -and (-not $RealDownload -or $InteractiveLogin)){throw 'Other-site integration uses RealDownload in anonymous mode.'}
$env:VB_YTDLP=$YtDlpPath
$env:VB_FFMPEG=$FFmpegPath
if(-not $RealDownload){
    $fixture=Join-Path $root '.cache\native-fixture'
    New-Item -ItemType Directory -Path $fixture -Force | Out-Null
    $env:GOCACHE=Join-Path $root '.cache\go-build'
    $env:GOMODCACHE=Join-Path $root '.cache\go-mod'
    $go=Get-Command go -ErrorAction SilentlyContinue
    $goExe=if($go){$go.Source}else{Join-Path $env:ProgramFiles 'Go\bin\go.exe'}
    Push-Location (Join-Path $root 'host')
    try{ & $goExe build -o (Join-Path $fixture 'yt-dlp.exe') ./internal/bridge/testdata/fake-tool; if($LASTEXITCODE -ne 0){throw 'Fixture build failed.'} }finally{Pop-Location}
    Copy-Item -LiteralPath (Join-Path $fixture 'yt-dlp.exe') -Destination (Join-Path $fixture 'ffmpeg.exe') -Force
    Copy-Item -LiteralPath (Join-Path $fixture 'yt-dlp.exe') -Destination (Join-Path $fixture 'ffprobe.exe') -Force
}
Push-Location (Join-Path $root 'extension')
try{
    & node tests/browser-native.mjs
    if($LASTEXITCODE -ne 0){throw 'Browser Native Messaging test failed; see docs/test-results.'}
}finally{
    Pop-Location
    if($existed){
        if($null -eq $oldValue){Remove-ItemProperty -LiteralPath $registry -Name '(default)' -ErrorAction SilentlyContinue}
        else{Set-Item -LiteralPath $registry -Value $oldValue}
    }elseif(Test-Path -LiteralPath $registry){Remove-Item -LiteralPath $registry}
    Write-Host 'Original Native Messaging registration restored.'
}
