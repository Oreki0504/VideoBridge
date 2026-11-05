[CmdletBinding()]
param([ValidateSet('Chrome','Edge','Both')][string]$Browser = 'Both')
$ErrorActionPreference = 'Stop'
$keys = @()
if ($Browser -in @('Chrome','Both')) { $keys += 'HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.videobridge.host' }
if ($Browser -in @('Edge','Both')) { $keys += 'HKCU:\Software\Microsoft\Edge\NativeMessagingHosts\com.videobridge.host' }
foreach ($key in $keys) { if (Test-Path -LiteralPath $key) { Remove-Item -LiteralPath $key } }
Write-Host 'Native Host registration removed. Downloaded media and local application files were preserved.'
