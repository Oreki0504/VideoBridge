[CmdletBinding()]
param([string]$Compiler = '')
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
if (-not $Compiler) { $Compiler = Join-Path $root '.cache\inno-setup-7.1.1\compiler\ISCC.exe' }
if (-not (Test-Path -LiteralPath $Compiler)) { throw 'Inno Setup 7 compiler is required. Pass its ISCC.exe path with -Compiler.' }
$build = Get-Content -LiteralPath (Join-Path $root 'dist\host\build-info.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$manifest = Get-Content -LiteralPath (Join-Path $root 'extension\.output\chrome-mv3\manifest.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$exe = Join-Path $root 'dist\host\videobridge.exe'
if ($build.version -ne $manifest.version -or (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash -ne $build.sha256) { throw 'Host and extension must have matching successful build records.' }
$key = (Get-Content -LiteralPath (Join-Path $root 'installer\extension-key.txt') -Raw).Trim()
$hasher = [Security.Cryptography.SHA256]::Create()
try { $digest = $hasher.ComputeHash([Convert]::FromBase64String($key)) } finally { $hasher.Dispose() }
$extensionId = -join (([BitConverter]::ToString($digest[0..15])).Replace('-','').ToLowerInvariant().ToCharArray() | ForEach-Object { [char](97 + [Convert]::ToInt32([string]$_,16)) })
$manifest | Add-Member -NotePropertyName key -NotePropertyValue $key -Force
$payload = Join-Path $root ('.cache\installer-payload-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path (Join-Path $payload 'host'),(Join-Path $payload 'setup') -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $root 'extension\.output\chrome-mv3') -Destination (Join-Path $payload 'extension') -Recurse
$utf8 = New-Object Text.UTF8Encoding($false)
[IO.File]::WriteAllText((Join-Path $payload 'extension\manifest.json'),($manifest | ConvertTo-Json -Depth 8),$utf8)
$hostFile = 'videobridge-' + $build.sha256.Substring(0,12).ToLowerInvariant() + '.exe'
Copy-Item -LiteralPath $exe -Destination (Join-Path $payload ('host\' + $hostFile))
Copy-Item -LiteralPath (Join-Path $root 'dist\host\build-info.json') -Destination (Join-Path $payload 'host')
foreach ($name in @('Configure-Installation.ps1','Unregister-Installation.ps1','welcome.html')) { Copy-Item -LiteralPath (Join-Path $root ('installer\' + $name)) -Destination (Join-Path $payload 'setup') }
[IO.File]::WriteAllText((Join-Path $payload 'setup\package-info.json'),(@{version=$build.version;extensionId=$extensionId} | ConvertTo-Json),$utf8)
# Windows PowerShell 5.1 requires a BOM for Chinese source text.
foreach ($name in @('Configure-Installation.ps1','Unregister-Installation.ps1')) {
    $file = Join-Path $payload ('setup\' + $name)
    [IO.File]::WriteAllText($file,[IO.File]::ReadAllText($file),(New-Object Text.UTF8Encoding($true)))
}
& $Compiler ('/DPayloadDir=' + $payload) ('/DAppVersion=' + $build.version) ('/DHostFile=' + $hostFile) (Join-Path $root 'installer\VideoBridge.iss')
if ($LASTEXITCODE -ne 0) { throw 'Installer compilation failed.' }
$output = Join-Path $root ('dist\VideoBridge-' + $build.version + '-Setup.exe')
$hash = (Get-FileHash -LiteralPath $output -Algorithm SHA256).Hash.ToLowerInvariant()
[IO.File]::WriteAllText(($output + '.sha256'),($hash + '  ' + [IO.Path]::GetFileName($output) + [Environment]::NewLine),$utf8)
[IO.File]::WriteAllText((Join-Path $root 'dist\installer-info.json'),(@{version=$build.version;extensionId=$extensionId;sha256=$hash;hostSha256=$build.sha256;payload=$payload} | ConvertTo-Json),$utf8)
Write-Output "Installer ready: $output"
Write-Output "Stable extension ID: $extensionId"
