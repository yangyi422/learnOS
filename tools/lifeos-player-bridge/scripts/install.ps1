<#
.SYNOPSIS
  Install the LifeOS player bridge for the current user (no admin rights).
.DESCRIPTION
  1. verifies the Windows architecture and the built binary,
  2. detects PotPlayer,
  3. creates %LOCALAPPDATA%\LifeOS\PlayerBridge and copies config.example.yaml (never overwrites),
  4. registers the lifeos-player:// protocol under HKCU (URL Protocol + shell\open\command),
  5. reports whether the mapped NAS drive and Jellyfin are reachable.
.PARAMETER JellyfinToken
  Stored with cmdkey when provided; never written to the config file or the log.
.PARAMETER JellyfinUser
  Jellyfin user id used as the cmdkey user name (default: config jellyfin.user_id).
#>
param(
  [string]$JellyfinToken = '',
  [string]$JellyfinUser = '',
  # Optional: point at the built binary when it does not sit in dist\.
  [string]$ExePath = ''
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$candidates = @()
if ($ExePath) { $candidates += $ExePath }
$candidates += (Join-Path $root 'dist\player-bridge.exe')
$candidates += (Join-Path $root 'player-bridge.exe')
$exe = $candidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $exe) {
  throw ('player-bridge.exe not found. Looked in: ' + ($candidates -join '; ') + ' — copy the built binary there or pass -ExePath.')
}
$exe = (Resolve-Path $exe).Path
'bridge: ' + $exe

'architecture: ' + [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture

$potPlayer = @(
  (Join-Path $env:ProgramFiles 'DAUM\PotPlayer\PotPlayerMini64.exe'),
  (Join-Path $env:ProgramFiles 'DAUM\PotPlayer\PotPlayerMini.exe')
) | Where-Object { Test-Path $_ } | Select-Object -First 1
if ($potPlayer) { 'PotPlayer: ' + $potPlayer } else { 'WARNING: PotPlayer not found in Program Files — install it or set player.executable.' }

$configDir = Join-Path $env:LOCALAPPDATA 'LifeOS\PlayerBridge'
New-Item -ItemType Directory -Force $configDir | Out-Null
$configPath = Join-Path $configDir 'config.yaml'
if (-not (Test-Path $configPath)) {
  Copy-Item (Join-Path $root 'config.example.yaml') $configPath
  'config created: ' + $configPath + ' (edit path_mappings and jellyfin.user_id)'
} else {
  'config kept: ' + $configPath
}

if ($JellyfinToken) {
  if (-not $JellyfinUser) { $JellyfinUser = 'lifeos-bridge' }
  & cmdkey /generic:lifeos-player-bridge/jellyfin /user:$JellyfinUser /pass:$JellyfinToken | Out-Null
  'credential stored: lifeos-player-bridge/jellyfin (user ' + $JellyfinUser + ')'
}

$protocolRoot = 'HKCU:\Software\Classes\lifeos-player'
New-Item -Path $protocolRoot -Force | Out-Null
Set-ItemProperty -Path $protocolRoot -Name '(default)' -Value 'URL:LifeOS Player Protocol'
Set-ItemProperty -Path $protocolRoot -Name 'URL Protocol' -Value ''
$commandKey = Join-Path $protocolRoot 'shell\open\command'
New-Item -Path $commandKey -Force | Out-Null
Set-ItemProperty -Path $commandKey -Name '(default)' -Value ('"' + $exe + '" "%1"')

$registered = (Get-ItemProperty $commandKey).'(default)'
if ($registered -ne ('"' + $exe + '" "%1"')) { throw 'protocol registration verification failed' }
'protocol registered: lifeos-player:// → ' + $registered

'--- bridge self check ---'
& $exe -verify
'--- mapped drives ---'
Get-PSDrive -PSProvider FileSystem | Select-Object Name, Root, DisplayRoot | Format-Table -AutoSize | Out-String -Width 160
'next: edit ' + $configPath + ' so path_mappings match the NAS container path and the mapped drive.'