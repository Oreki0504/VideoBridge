[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go) { $go = Join-Path $env:ProgramFiles 'Go\bin\go.exe' }
$env:GOCACHE = Join-Path $root '.cache\go-build'
$env:GOMODCACHE = Join-Path $root '.cache\go-mod'
$prior = @{GOOS=$env:GOOS;GOARCH=$env:GOARCH;CGO_ENABLED=$env:CGO_ENABLED}
Push-Location $root
try {
    & npm.cmd run check
    if ($LASTEXITCODE -ne 0) { throw 'TypeScript check failed' }
    & npm.cmd test
    if ($LASTEXITCODE -ne 0) { throw 'Extension tests failed' }
    & npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw 'Extension build failed' }
    Push-Location (Join-Path $root 'host')
    try {
        & $go test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Host tests failed' }
        & $go vet ./...
        if ($LASTEXITCODE -ne 0) { throw 'Host vet failed' }
        $env:GOOS='darwin'; $env:CGO_ENABLED='0'
        foreach ($arch in @('arm64','amd64')) {
            $env:GOARCH=$arch
            $dest=Join-Path $root ('dist\macos\darwin-'+$arch)
            New-Item -ItemType Directory -Path $dest -Force | Out-Null
            & $go build -trimpath -o (Join-Path $dest 'videobridge') ./cmd/videobridge
            if ($LASTEXITCODE -ne 0) { throw "Mac $arch build failed" }
            & $go test -c -o (Join-Path $dest 'bridge.test') ./internal/bridge
            if ($LASTEXITCODE -ne 0) { throw "Mac $arch test compilation failed" }
            & $go test -c -o (Join-Path $dest 'macinstall.test') ./internal/macinstall
            if ($LASTEXITCODE -ne 0) { throw "Mac $arch installer test compilation failed" }
            & $go vet ./...
            if ($LASTEXITCODE -ne 0) { throw "Mac $arch vet failed" }
        }
    } finally { Pop-Location }
    & python (Join-Path $root 'scripts\package_macos.py') --root $root
    if ($LASTEXITCODE -ne 0) { throw 'Mac archive verification failed' }
} finally {
    $env:GOOS=$prior.GOOS; $env:GOARCH=$prior.GOARCH; $env:CGO_ENABLED=$prior.CGO_ENABLED
    Pop-Location
}
