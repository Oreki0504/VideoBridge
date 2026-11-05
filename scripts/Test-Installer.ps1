[CmdletBinding()]
param([Parameter(Mandatory)][string]$YtDlpPath,[Parameter(Mandatory)][string]$FFmpegPath)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$version = (Get-Content (Join-Path $root 'dist\installer-info.json') -Raw | ConvertFrom-Json).version
$installer = Join-Path $root ('dist\VideoBridge-' + $version + '-Setup.exe')
$appRegistry = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{8A8DD097-86D8-4ED0-9C18-A83A6F23DA15}_is1'
if (Test-Path -LiteralPath $appRegistry) { throw 'An installer-managed VideoBridge exists. Run lifecycle tests in a disposable account.' }
$testRoot = Join-Path $root ('.cache\installer-test-' + [guid]::NewGuid().ToString('N'))
# Validate the absolute workspace target before invoking an uninstaller against it.
$testRoot = [IO.Path]::GetFullPath($testRoot)
if (-not $testRoot.StartsWith(([IO.Path]::GetFullPath((Join-Path $root '.cache')) + '\'),[StringComparison]::OrdinalIgnoreCase)) { throw 'Unsafe test root.' }
$app = Join-Path $testRoot 'VideoBridge 测试'
$media = Join-Path $testRoot '下载文件'
New-Item -ItemType Directory -Path (Join-Path $app 'host'),$media -Force | Out-Null
$utf8 = New-Object Text.UTF8Encoding($false)
$configPath = Join-Path $app 'host\config.json'
[IO.File]::WriteAllText($configPath,(@{outputDir=$media;ytDlpPath=$YtDlpPath;ffmpegPath=$FFmpegPath} | ConvertTo-Json),$utf8)
$configBefore = [IO.File]::ReadAllText($configPath)
$keep = Join-Path $media 'keep.txt';[IO.File]::WriteAllText($keep,'preserve downloaded media',$utf8)
$legacyOrigin = 'chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/'
[IO.File]::WriteAllText((Join-Path $app 'host\com.videobridge.host.chrome.json'),(@{allowed_origins=@($legacyOrigin)} | ConvertTo-Json),$utf8)
$keys = @('HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.videobridge.host','HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.videobridge.host')
$saved = @($keys | ForEach-Object { @{Key=$_;Exists=(Test-Path -LiteralPath $_);Value=$(if(Test-Path -LiteralPath $_){(Get-Item -LiteralPath $_).GetValue('')}else{$null})} })
function Run-Setup([string]$log,[int]$expected = 0) {
    $p = Start-Process -FilePath $installer -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART','/NOICONS',('/DIR="' + $app + '"'),'/TASKS="chrome,edge"',('/LOG="' + $log + '"')) -WindowStyle Hidden -PassThru -Wait
    if ($p.ExitCode -ne $expected) { throw "Installer exit $($p.ExitCode), expected $expected; see $log" }
}
try {
    Run-Setup (Join-Path $testRoot 'install.log')
    if (-not (Test-Path -LiteralPath $appRegistry)) { throw 'Windows uninstall entry missing.' }
    if ([IO.File]::ReadAllText($configPath) -ne $configBefore) { throw 'Existing config changed.' }
    $package = Get-Content -LiteralPath (Join-Path $app 'setup\package-info.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach ($target in @('chrome','edge')) {
        $m = Get-Content -LiteralPath (Join-Path $app ('host\com.videobridge.host.' + $target + '.json')) -Raw -Encoding UTF8 | ConvertFrom-Json
        if ($m.allowed_origins -notcontains ('chrome-extension://' + $package.extensionId + '/')) { throw 'Stable extension ID missing.' }
        if ($target -eq 'chrome' -and $m.allowed_origins -notcontains $legacyOrigin) { throw 'Legacy extension permission was lost.' }
    }
    Write-Output 'PASS: installation, uninstall entry, stable ID, legacy origin and Unicode configuration preservation.'
    if (-not ([IO.File]::ReadAllText((Join-Path $testRoot 'install.log'))).Contains('浏览器连接已配置')) { throw 'Chinese setup progress encoding is incorrect.' }
    [IO.File]::WriteAllText($configPath,'{"outputDir":"relative","ytDlpPath":"","ffmpegPath":""}',$utf8)
    Run-Setup (Join-Path $testRoot 'expected-failure.log') 1
    [IO.File]::WriteAllText($configPath,$configBefore,$utf8)
    Run-Setup (Join-Path $testRoot 'upgrade.log')
    if ([IO.File]::ReadAllText($configPath) -ne $configBefore) { throw 'Upgrade changed configuration.' }
    $env:VB_INSTALLER_TEST_ROOT=$app
    & node (Join-Path $root 'extension\tests\installer-native.mjs')
    if ($LASTEXITCODE -ne 0) { throw 'Installed browser integration failed.' }
    Write-Output 'PASS: failure exit code, recovery, repeated installation and actual browser connection.'
    $externalManifest = Join-Path $testRoot 'another-installation.json'
    Set-Item -LiteralPath $keys[0] -Value $externalManifest
    $uninstaller = Join-Path $app 'unins000.exe'
    $p = Start-Process -FilePath $uninstaller -ArgumentList @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART',('/LOG="' + (Join-Path $testRoot 'uninstall.log') + '"')) -WindowStyle Hidden -PassThru -Wait
    if ($p.ExitCode -ne 0) { throw 'Uninstall failed.' }
    if ((Get-Item -LiteralPath $keys[0]).GetValue('') -ne $externalManifest) { throw 'Uninstaller removed a different installation registration.' }
    if (Test-Path -LiteralPath $keys[1]) { throw 'Owned Edge registration retained.' }
    if (Test-Path -LiteralPath $appRegistry) { throw 'Uninstall entry retained.' }
    if (Test-Path -LiteralPath (Join-Path $app 'extension\manifest.json')) { throw 'Application files retained.' }
    if (-not (Test-Path -LiteralPath $keep) -or [IO.File]::ReadAllText($configPath) -ne $configBefore) { throw 'Uninstall removed media or settings.' }
    Write-Output 'PASS: uninstall removes components and owned registration, preserves media/settings and other registration.'
    [IO.File]::WriteAllText((Join-Path $root 'docs\test-results\installer-lifecycle.json'),(@{passed=$true;testRoot=$testRoot;extensionId=$package.extensionId;version=$package.version} | ConvertTo-Json),$utf8)
} finally {
    foreach ($s in $saved) {
        if ($s.Exists) { New-Item -Path $s.Key -Force | Out-Null;Set-Item -LiteralPath $s.Key -Value $s.Value }
        elseif (Test-Path -LiteralPath $s.Key) { Remove-Item -LiteralPath $s.Key }
    }
    Write-Output 'Original Chrome/Edge registrations restored.'
}
