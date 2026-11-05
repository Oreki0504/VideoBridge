[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Root,
    [ValidateSet('Chrome','Edge','Both')][string]$Browser = 'Chrome',
    [switch]$SkipDependencies
)
$ErrorActionPreference = 'Stop'
# Inno reads captured output as UTF-8; Windows PowerShell otherwise emits the OEM code page.
[Console]::OutputEncoding = New-Object Text.UTF8Encoding($false)
$OutputEncoding = [Console]::OutputEncoding
if (-not [IO.Path]::IsPathRooted($Root)) { throw 'Installation directory must be absolute.' }
$Root = [IO.Path]::GetFullPath($Root).TrimEnd('\')
$hostDir = Join-Path $Root 'host'
$build = Get-Content -LiteralPath (Join-Path $hostDir 'build-info.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$package = Get-Content -LiteralPath (Join-Path $Root 'setup\package-info.json') -Raw -Encoding UTF8 | ConvertFrom-Json
if ($package.extensionId -notmatch '^[a-p]{32}$' -or $build.sha256 -notmatch '^[A-Fa-f0-9]{64}$' -or $build.version -ne $package.version) { throw 'Invalid package metadata.' }
$exe = Join-Path $hostDir ('videobridge-' + $build.sha256.Substring(0,12).ToLowerInvariant() + '.exe')
$hasher = [Security.Cryptography.SHA256]::Create()
$stream = [IO.File]::OpenRead($exe)
try { $actualHash = [BitConverter]::ToString($hasher.ComputeHash($stream)).Replace('-','') }
finally { $stream.Dispose(); $hasher.Dispose() }
if ($actualHash -ne $build.sha256) { throw 'Host checksum mismatch.' }
$configPath = Join-Path $hostDir 'config.json'
$utf8 = New-Object Text.UTF8Encoding($false)
if (-not (Test-Path -LiteralPath $configPath)) {
    $ytdlp = Get-Command yt-dlp.exe -ErrorAction SilentlyContinue
    $ffmpeg = Get-Command ffmpeg.exe -ErrorAction SilentlyContinue
    $config = @{outputDir=(Join-Path ([Environment]::GetFolderPath('UserProfile')) 'Downloads\VideoBridge');ytDlpPath='';ffmpegPath=''}
    if ($ytdlp) { $config.ytDlpPath = $ytdlp.Source }
    if ($ffmpeg) { $config.ffmpegPath = $ffmpeg.Source }
    [IO.File]::WriteAllText($configPath,($config | ConvertTo-Json),$utf8)
}
# Register before setup so a failed network download can be retried from the extension.
$targets = if ($Browser -eq 'Both') { @('Chrome','Edge') } else { @($Browser) }
foreach ($target in $targets) {
    $manifestPath = Join-Path $hostDir ('com.videobridge.host.' + $target.ToLowerInvariant() + '.json')
    $origins = @('chrome-extension://' + $package.extensionId + '/')
    if (Test-Path -LiteralPath $manifestPath) {
        $old = Get-Content -LiteralPath $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
        # Keep valid, already-authorized extension IDs during migration to a stable directory.
        $origins += @($old.allowed_origins | Where-Object { $_ -match '^chrome-extension://[a-p]{32}/$' })
    }
    $manifest = @{name='com.videobridge.host';description='VideoBridge local yt-dlp bridge';path=$exe;type='stdio';allowed_origins=@($origins | Select-Object -Unique)}
    [IO.File]::WriteAllText($manifestPath,($manifest | ConvertTo-Json),$utf8)
    $registry = if ($target -eq 'Chrome') { 'HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.videobridge.host' } else { 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.videobridge.host' }
    New-Item -Path $registry -Force | Out-Null
    Set-Item -LiteralPath $registry -Value $manifestPath
}
Write-Output '浏览器连接已配置，正在检查本地依赖…'
if ($SkipDependencies) { & $exe --check } else { & $exe --setup }
if ($LASTEXITCODE -ne 0) { throw '依赖安装未完成；可重新运行安装程序，或在插件中点击“检测并补齐依赖”。' }
Write-Output 'VideoBridge 安装配置完成。'
