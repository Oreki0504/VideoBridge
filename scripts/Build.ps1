[CmdletBinding()]
param([switch]$SkipInstall)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$buildInfoPath = Join-Path $root 'dist\host\build-info.json'
# Invalidate a previous success record before starting a new build attempt.
if (Test-Path -LiteralPath $buildInfoPath) { Remove-Item -LiteralPath $buildInfoPath }
$go = Get-Command go -ErrorAction SilentlyContinue
if ($go) { $goExe = $go.Source } else { $goExe = Join-Path $env:ProgramFiles 'Go\bin\go.exe' }
if (-not (Test-Path -LiteralPath $goExe)) { throw 'Go 1.26+ is required. Install it from https://go.dev/dl/.' }
Push-Location $root
try {
    if (-not $SkipInstall) {
        & npm.cmd ci --prefix extension --cache .cache/npm --no-audit --no-fund
        if ($LASTEXITCODE -ne 0) { throw 'npm ci failed.' }
    }
    & npm.cmd run check
    if ($LASTEXITCODE -ne 0) { throw 'TypeScript check failed.' }
    & npm.cmd test
    if ($LASTEXITCODE -ne 0) { throw 'Extension tests failed.' }
    & npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw 'Extension build failed.' }
    $env:GOCACHE = Join-Path $root '.cache\go-build'
    $env:GOMODCACHE = Join-Path $root '.cache\go-mod'
    Push-Location (Join-Path $root 'host')
    try {
        & $goExe test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Go tests failed.' }
        & $goExe vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }
        New-Item -ItemType Directory -Path (Join-Path $root 'dist\host') -Force | Out-Null
        & $goExe build -trimpath -o (Join-Path $root 'dist\host\videobridge.exe') ./cmd/videobridge
        if ($LASTEXITCODE -ne 0) { throw 'Native Host build failed. Do not install a remaining old EXE. If Defender reported a detection, review Windows Security protection history; see docs/DEFENDER.md.' }
        $buildInfo = @{ version = (Get-Content (Join-Path $root 'extension\package.json') -Raw | ConvertFrom-Json).version; sha256 = (Get-FileHash (Join-Path $root 'dist\host\videobridge.exe') -Algorithm SHA256).Hash }
        [IO.File]::WriteAllText((Join-Path $root 'dist\host\build-info.json'),($buildInfo | ConvertTo-Json),(New-Object Text.UTF8Encoding($false)))
    } finally { Pop-Location }
    Copy-Item -LiteralPath (Join-Path $root 'host\config.example.json') -Destination (Join-Path $root 'dist\host\config.example.json') -Force
    Write-Host 'Build complete. Extension: extension\.output\chrome-mv3; Native Host: dist\host\videobridge.exe'
} finally { Pop-Location }
