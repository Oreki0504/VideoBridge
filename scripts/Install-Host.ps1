[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidatePattern('^[a-p]{32}$')][string]$ExtensionId,
    [ValidateSet('Chrome','Edge','Both')][string]$Browser = 'Chrome',
    [string]$OutputDir = (Join-Path ([Environment]::GetFolderPath('UserProfile')) 'Downloads\VideoBridge'),
    [string]$YtDlpPath = '',
    [string]$FFmpegPath = '',
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA 'VideoBridge\host'),
    [switch]$SkipDependencies
)
$ErrorActionPreference = 'Stop'
# Preserve existing choices unless explicitly overridden by the caller.
$configPath = Join-Path $InstallDir 'config.json'
$previousYtDlpPath = ''
if (Test-Path -LiteralPath $configPath) {
    $existing = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
    if (-not $PSBoundParameters.ContainsKey('OutputDir')) { $OutputDir = $existing.outputDir }
    if (-not $PSBoundParameters.ContainsKey('YtDlpPath')) { $YtDlpPath = $existing.ytDlpPath }
    if (-not $PSBoundParameters.ContainsKey('YtDlpPath')) { $previousYtDlpPath = $existing.previousYtDlpPath }
    if (-not $PSBoundParameters.ContainsKey('FFmpegPath')) { $FFmpegPath = $existing.ffmpegPath }
}
foreach ($path in @($OutputDir,$InstallDir)) {
    if (-not [IO.Path]::IsPathRooted($path)) { throw 'Output and installation directories must be absolute paths.' }
}
foreach ($path in @($YtDlpPath,$FFmpegPath)) {
    if ($path -and -not [IO.Path]::IsPathRooted($path)) { throw 'Tool paths must be absolute executable paths.' }
}
if (-not $YtDlpPath) { $tool = Get-Command yt-dlp -ErrorAction SilentlyContinue; if ($tool) { $YtDlpPath = $tool.Source } }
if (-not $FFmpegPath) { $tool = Get-Command ffmpeg -ErrorAction SilentlyContinue; if ($tool) { $FFmpegPath = $tool.Source } }
$root = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $root 'dist\host\videobridge.exe'
if (-not (Test-Path -LiteralPath $binary)) { throw 'Run scripts\Build.ps1 first.' }
$buildInfoPath = Join-Path $root 'dist\host\build-info.json'
if (-not (Test-Path -LiteralPath $buildInfoPath)) { throw 'No successful build record. Run scripts\Build.ps1; do not install a leftover EXE after a failed build.' }
$buildInfo = Get-Content -LiteralPath $buildInfoPath -Raw | ConvertFrom-Json
$version = (Get-Content (Join-Path $root 'extension\package.json') -Raw | ConvertFrom-Json).version
if ($buildInfo.version -ne $version -or $buildInfo.sha256 -ne (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash) { throw 'Build version or checksum mismatch. Run scripts\Build.ps1 again and review any security detection before installing.' }
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
# A running Native Host holds its executable open on Windows. Install immutable,
# content-addressed builds so an update never interrupts an active download.
$hasher = [System.Security.Cryptography.SHA256]::Create()
$binaryStream = [IO.File]::OpenRead($binary)
try { $buildId = ([BitConverter]::ToString($hasher.ComputeHash($binaryStream))).Replace('-','').Substring(0,12).ToLowerInvariant() }
finally { $binaryStream.Dispose(); $hasher.Dispose() }
$exe = Join-Path $InstallDir ('videobridge-' + $buildId + '.exe')
if (-not (Test-Path -LiteralPath $exe)) { Copy-Item -LiteralPath $binary -Destination $exe }
$utf8 = New-Object System.Text.UTF8Encoding($false)
$configFields = @{ outputDir = [IO.Path]::GetFullPath($OutputDir); ytDlpPath = $YtDlpPath; ffmpegPath = $FFmpegPath }
if ($previousYtDlpPath) { $configFields.previousYtDlpPath = $previousYtDlpPath }
$config = $configFields | ConvertTo-Json
[IO.File]::WriteAllText((Join-Path $InstallDir 'config.json'),$config,$utf8)
if (-not $SkipDependencies) {
    & $exe --setup
    if ($LASTEXITCODE -ne 0) { throw 'Dependency setup failed. Check network access and rerun, or configure tools manually with -SkipDependencies.' }
}
$targets = switch ($Browser) { 'Chrome' { 'Chrome' }; 'Edge' { 'Edge' }; 'Both' { 'Chrome'; 'Edge' } }
foreach ($target in $targets) {
    $manifestPath = Join-Path $InstallDir ('com.videobridge.host.' + $target.ToLowerInvariant() + '.json')
    $manifest = @{
        name = 'com.videobridge.host'; description = 'VideoBridge local yt-dlp bridge'
        path = $exe; type = 'stdio'; allowed_origins = @('chrome-extension://' + $ExtensionId + '/')
    } | ConvertTo-Json
    [IO.File]::WriteAllText($manifestPath,$manifest,$utf8)
    $registry = if ($target -eq 'Chrome') { 'HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.videobridge.host' } else { 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.videobridge.host' }
    New-Item -Path $registry -Force | Out-Null
    Set-Item -LiteralPath $registry -Value $manifestPath
}
Write-Host "Registered for $Browser (current user only). Extension ID: $ExtensionId"
& $exe --check
if ($LASTEXITCODE -ne 0) { throw 'Host diagnostics failed.' }
Write-Host 'Open a YouTube video or X post. Grant site cookies only when using its login session.'
