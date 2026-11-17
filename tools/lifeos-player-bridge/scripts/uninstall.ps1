<#
.SYNOPSIS
  Remove the lifeos-player:// protocol registration for the current user.
.DESCRIPTION
  Deletes only HKCU\Software\Classes\lifeos-player. User configuration, the stored
  credential and PotPlayer itself are left untouched.
#>
$ErrorActionPreference = 'Stop'
$protocolRoot = 'HKCU:\Software\Classes\lifeos-player'
if (Test-Path $protocolRoot) {
  Remove-Item -Path $protocolRoot -Recurse -Force
  'protocol removed: lifeos-player://'
} else {
  'protocol was not registered'
}
'kept: config at ' + (Join-Path $env:LOCALAPPDATA 'LifeOS\PlayerBridge\config.yaml')
'kept: credential lifeos-player-bridge/jellyfin (remove with: cmdkey /delete:lifeos-player-bridge/jellyfin)'
'kept: PotPlayer installation'