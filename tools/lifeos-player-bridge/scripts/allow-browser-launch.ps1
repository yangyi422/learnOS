<#
.SYNOPSIS
  Let one LifeOS origin launch lifeos-player:// without the browser confirmation.
.DESCRIPTION
  Chrome/Edge treat an external protocol launch as a security boundary, so the dialog
  cannot be removed by the web page or by the bridge. Two supported ways out:
    1. ExternalProtocolDialogShowAlwaysOpenCheckbox = 1  -> the dialog shows an
       "always allow" checkbox you tick once per origin.
    2. AutoLaunchProtocolsFromOrigins                     -> no dialog at all for the
       listed origin/protocol pairs.
  Both are browser policies. Per-user policies are written under HKCU by default;
  pass -Machine to write under HKLM (needs an elevated shell).
.PARAMETER Origin
  Origin that may auto-launch, e.g. https://43.138.202.56:8888
.PARAMETER Protocol
  Protocol scheme, default lifeos-player
.PARAMETER Browser
  chrome, edge or both (default both)
.PARAMETER Machine
  Write HKLM policies instead of HKCU (administrator required)
.PARAMETER Remove
  Delete the policies again
.EXAMPLE
  .\scripts\allow-browser-launch.ps1 -Origin https://43.138.202.56:8888
.EXAMPLE
  .\scripts\allow-browser-launch.ps1 -Origin https://43.138.202.56:8888 -Remove
#>
param(
  [Parameter(Mandatory = $true)][string]$Origin,
  [string]$Protocol = 'lifeos-player',
  [ValidateSet('chrome', 'edge', 'both')][string]$Browser = 'both',
  [switch]$Machine,
  [switch]$Remove
)
$ErrorActionPreference = 'Stop'
$Origin = $Origin.TrimEnd('/')
if ($Origin -notmatch '^https?://[^/]+$') { throw 'Origin must look like https://host:port (no path)' }

$hive = if ($Machine) { 'HKLM:' } else { 'HKCU:' }
$targets = [ordered]@{}
if ($Browser -eq 'chrome' -or $Browser -eq 'both') { $targets['chrome'] = $hive + '\Software\Policies\Google\Chrome' }
if ($Browser -eq 'edge' -or $Browser -eq 'both') { $targets['edge'] = $hive + '\Software\Policies\Microsoft\Edge' }

$autoLaunch = ConvertTo-Json -Compress -Depth 5 -InputObject @(@{ allowed_origins = @($Origin); protocol = $Protocol })

foreach ($name in $targets.Keys) {
  $key = $targets[$name]
  if ($Remove) {
    if (Test-Path $key) {
      foreach ($value in 'AutoLaunchProtocolsFromOrigins', 'ExternalProtocolDialogShowAlwaysOpenCheckbox') {
        if ((Get-ItemProperty -Path $key -Name $value -ErrorAction SilentlyContinue)) {
          Remove-ItemProperty -Path $key -Name $value -Force
          $name + ': removed ' + $value
        }
      }
    } else {
      $name + ': no policy key at ' + $key
    }
    continue
  }
  try {
    New-Item -Path $key -Force | Out-Null
    New-ItemProperty -Path $key -Name 'AutoLaunchProtocolsFromOrigins' -Value $autoLaunch -PropertyType String -Force | Out-Null
    New-ItemProperty -Path $key -Name 'ExternalProtocolDialogShowAlwaysOpenCheckbox' -Value 1 -PropertyType DWord -Force | Out-Null
  } catch [System.UnauthorizedAccessException] {
    $name + ': cannot write ' + $key
    '  Windows protects the Policies key: run this script from an elevated PowerShell, e.g.'
    '  Start-Process powershell -Verb RunAs -ArgumentList ''-NoProfile -ExecutionPolicy Bypass -File "' + $PSCommandPath + '" -Origin ' + $Origin + ''''
    continue
  }
  $readBack = (Get-ItemProperty -Path $key -Name 'AutoLaunchProtocolsFromOrigins').AutoLaunchProtocolsFromOrigins
  $name + ': ' + $key
  $name + ': AutoLaunchProtocolsFromOrigins = ' + $readBack
}

if (-not $Remove) {
  ''
  'next steps'
  '  1. fully quit the browser (all windows) and start it again — policies are read at startup'
  '  2. open chrome://policy (or edge://policy) and press "Reload policies"'
  '  3. look for AutoLaunchProtocolsFromOrigins; it should list ' + $Origin + ' -> ' + $Protocol
  '  4. clicking 使用 PotPlayer 播放 on ' + $Origin + ' now launches without the dialog'
  ''
  'note: an auto-launched protocol lets any page on that origin start a local app without asking;',
  '      keep the origin narrow (one host + port) and remove the policy with -Remove when you no longer need it.'
}