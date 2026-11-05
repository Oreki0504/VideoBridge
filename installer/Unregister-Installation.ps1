[CmdletBinding()]
param([Parameter(Mandatory)][string]$Root)
$ErrorActionPreference = 'Stop'
if (-not [IO.Path]::IsPathRooted($Root)) { throw 'Installation directory must be absolute.' }
$hostDir = Join-Path ([IO.Path]::GetFullPath($Root)) 'host'
foreach ($target in @('Chrome','Edge')) {
    $manifestPath = Join-Path $hostDir ('com.videobridge.host.' + $target.ToLowerInvariant() + '.json')
    $registry = if ($target -eq 'Chrome') { 'HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.videobridge.host' } else { 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.videobridge.host' }
    if (Test-Path -LiteralPath $registry) {
        $current = (Get-Item -LiteralPath $registry).GetValue('')
        # Never remove registration belonging to a different installation.
        if ($current -eq $manifestPath) { Remove-Item -LiteralPath $registry }
    }
}
# Inno removes only its tracked application files. Keep media, user configuration,
# and managed dependencies; no recursive deletion of a configurable directory.
