<#
.SYNOPSIS
  One-shot setup for the LifeOS player bridge.
.DESCRIPTION
  Verifies the Jellyfin token, resolves the dedicated user id, stores the token in
  Windows Credential Manager and writes config.yaml. Nothing is written before the
  token has been accepted by Jellyfin.
.NOTES
  An API key has no associated user on the Jellyfin side, so /Users/Me may answer
  400. This script falls back to /Users (admin key), then to -JellyfinUser, then to
  an interactive prompt, so all three credential kinds work.
#>
param(
  [string]$JellyfinUrl = 'http://192.168.31.122:8096',
  [string]$JellyfinToken = '',
  [string]$JellyfinUser = '',
  [string]$JellyfinPassword = '',
  [string]$MoviesSource = '/media/movies/',
  [string]$MoviesTarget = '',
  [string]$Library = 'movies',
  [string]$ExePath = '',
  [string]$ConfigPath = ''
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$base = $JellyfinUrl.TrimEnd('/')
if (-not $ConfigPath) {
  $configDir = Join-Path $env:LOCALAPPDATA 'LifeOS\PlayerBridge'
  New-Item -ItemType Directory -Force $configDir | Out-Null
  $ConfigPath = Join-Path $configDir 'config.yaml'
}

function Invoke-JellyfinJson([string]$Path) {
  try {
    return @{ ok = $true; data = (Invoke-RestMethod -Uri ($base + $Path) -Headers $script:headers -TimeoutSec 15) }
  } catch {
    $status = 0
    if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
      $status = [int]$_.Exception.Response.StatusCode
    }
    return @{ ok = $false; status = $status; message = $_.Exception.Message }
  }
}

# Preferred path: sign in as the dedicated non-admin user and store that user's token,
# so Jellyfin-side library permissions actually apply to the bridge.
$loginUser = ''
if (-not $JellyfinToken -and $JellyfinUser -and $JellyfinPassword) {
  $loginUser = $JellyfinUser
} elseif (-not $JellyfinToken -and -not $JellyfinPassword) {
  Write-Host 'two ways to authenticate:'
  Write-Host '  [1] dedicated user name + password (recommended: permissions stay with that user)'
  Write-Host '  [2] API key (server-wide access; only the bridge allowlist then limits libraries)'
  $choice = Read-Host 'choose 1 or 2 (default 1)'
  if ($choice -eq '2') {
    $secure = Read-Host -AsSecureString ('Jellyfin API key for ' + $base)
    $JellyfinToken = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
  } else {
    $loginUser = Read-Host 'dedicated Jellyfin user name'
    $secure = Read-Host -AsSecureString ('password for ' + $loginUser)
    $JellyfinPassword = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure))
  }
}
$loginTokenUserId = ''
if (-not $JellyfinToken -and $loginUser -and $JellyfinPassword) {
  $authHeaders = @{ 'Content-Type' = 'application/json'; 'Authorization' = 'MediaBrowser Client="LifeOS Bridge", Device="Windows", DeviceId="lifeos-player-bridge", Version="0.1.0"' }
  $body = @{ Username = $loginUser; Pw = $JellyfinPassword } | ConvertTo-Json
  try {
    $session = Invoke-RestMethod -Method Post -Uri ($base + '/Users/AuthenticateByName') -Headers $authHeaders -Body $body -TimeoutSec 15
  } catch {
    throw ('Jellyfin login failed for ' + $loginUser + ': ' + $_.Exception.Message) }
  if (-not $session.AccessToken) { throw 'Jellyfin login returned no AccessToken' }
  $JellyfinToken = $session.AccessToken
  $loginTokenUserId = $session.User.Id
  Write-Host ('signed in as ' + $session.User.Name + ' (' + $loginTokenUserId + ')')
}
$JellyfinToken = $JellyfinToken.Trim()
if (-not $JellyfinToken) { throw 'a Jellyfin token is required' }
$script:headers = @{ 'X-Emby-Token' = $JellyfinToken; 'Accept' = 'application/json' }

'--- checking that Jellyfin accepts the token ---'
$info = Invoke-JellyfinJson '/System/Info'
if (-not $info.ok) {
  $hint = switch ($info.status) {
    401 { 'token rejected (401) — create a new API key for the dedicated user' }
    403 { 'token has no access (403)' }
    0   { 'Jellyfin unreachable — check the URL and the home network' }
    default { 'HTTP ' + $info.status + ' — ' + $info.message }
  }
  throw ('cannot verify the token: ' + $hint + ' [' + $base + ']')
}
'token accepted: ' + $info.data.ProductName + ' ' + $info.data.Version

function Resolve-UserId {
  # Only the id goes to the output stream; everything else is host output, otherwise
  # the caller would receive a concatenation of every message.
  $me = Invoke-JellyfinJson '/Users/Me'
  if ($me.ok -and $me.data.Id) {
    Write-Host ('user (from /Users/Me): ' + $me.data.Name + ' (' + $me.data.Id + ')')
    if ($me.data.Policy -and $me.data.Policy.IsAdministrator) { Write-Host 'WARNING: this user is a Jellyfin administrator; prefer a dedicated non-admin user.' }
    return $me.data.Id
  }
  Write-Host ('note: /Users/Me answered ' + $me.status + ' - an API key is not tied to a user, falling back to /Users')

  $users = Invoke-JellyfinJson '/Users'
  $list = @()
  if ($users.ok) { $list = @($users.data) }
  if ($list.Count -gt 0) {
    if ($JellyfinUser) {
      $match = $list | Where-Object { $_.Id -eq $JellyfinUser -or $_.Name -eq $JellyfinUser } | Select-Object -First 1
      if (-not $match) { throw ('no Jellyfin user matches ' + $JellyfinUser) }
      Write-Host ('user (matched by name/id): ' + $match.Name + ' (' + $match.Id + ')')
      return $match.Id
    }
    $index = 1
    foreach ($entry in $list) {
      $suffix = ''
      if ($entry.Policy -and $entry.Policy.IsAdministrator) { $suffix = ' (admin)' }
      Write-Host ('  [{0}] {1}{2}  {3}' -f $index, $entry.Name, $suffix, $entry.Id)
      $index++
    }
    $pick = Read-Host 'choose the dedicated user by number (or paste the 32-char user id)'
    if ($pick -match '^\d+$') {
      $chosen = $list[[int]$pick - 1]
      if (-not $chosen) { throw 'no user with that number' }
      Write-Host ('user: ' + $chosen.Name + ' (' + $chosen.Id + ')')
      return $chosen.Id
    }
    if ($pick -match '^[0-9a-fA-F]{32}$') { return $pick.ToLower() }
    throw 'no valid user chosen'
  }

  if ($JellyfinUser) {
    if ($JellyfinUser -notmatch '^[0-9a-fA-F]{32}$') { throw 'JellyfinUser must be the 32-char user id when /Users is not readable' }
    return $JellyfinUser.ToLower()
  }
  Write-Host ('note: /Users is not readable with this token either (HTTP ' + $users.status + ')')
  Write-Host 'Open Jellyfin -> Dashboard -> Users -> your dedicated user; the address bar shows userId=<32 hex>.'
  $manual = Read-Host 'paste the 32-char Jellyfin user id'
  if ($manual -notmatch '^[0-9a-fA-F]{32}$') { throw 'that is not a 32-char Jellyfin user id' }
  return $manual.ToLower()
}
$userId = if ($loginTokenUserId) { $loginTokenUserId } else { Resolve-UserId }
$probe = Invoke-JellyfinJson ('/Users/' + $userId)
if (-not $probe.ok) { throw ('the token cannot read user ' + $userId + ' (HTTP ' + $probe.status + ') — give the API key the same user''s library permissions, or use that user''s own token') }
'user readable: ' + $probe.data.Name

$stored = Join-Path $root 'dist\player-bridge.exe'
if ($ExePath) { $stored = $ExePath }
if (-not (Test-Path $stored)) { $stored = Join-Path $root 'player-bridge.exe' }
if (-not (Test-Path $stored)) { throw 'player-bridge.exe not found; build it or pass -ExePath' }
& cmdkey '/generic:lifeos-player-bridge/jellyfin' ('/user:' + $userId) ('/pass:' + $JellyfinToken) | Out-Null
'credential stored: lifeos-player-bridge/jellyfin'

if (-not $MoviesTarget) {
  '--- drive mappings (net use) ---'
  & net use
  '--- top-level folders per mapped drive ---'
  Get-PSDrive -PSProvider FileSystem | Where-Object { $_.DisplayRoot } | ForEach-Object {
    $drive = $_.Name
    try {
      $dirs = Get-ChildItem ($drive + ':\') -Directory -ErrorAction Stop | Select-Object -First 8 -ExpandProperty Name
      if ($dirs) { $drive + ': ' + ($dirs -join ', ') } else { $drive + ': (no folders)' }
    } catch { $drive + ': <unreachable>' }
  }
  $MoviesTarget = Read-Host 'Windows path of the movies library (e.g. Q:\Movies\)'
}
if (-not $MoviesTarget) { throw 'a Windows movies path is required (-MoviesTarget)' }
if ($MoviesTarget -notmatch '^[A-Za-z]:\\') { throw 'MoviesTarget must look like Q:\Movies\' }
if (-not $MoviesTarget.EndsWith('\')) { $MoviesTarget += '\' }
if (-not (Test-Path $MoviesTarget)) { 'WARNING: ' + $MoviesTarget + ' is not accessible right now (SMB not connected?)' }
if (-not $MoviesSource.StartsWith('/')) { throw 'MoviesSource must be an absolute container path like /media/movies/' }
if (-not $MoviesSource.EndsWith('/')) { $MoviesSource += '/' }

if (Test-Path $ConfigPath) { Copy-Item $ConfigPath ($ConfigPath + '.bak') -Force; 'backup: ' + $ConfigPath + '.bak' }
$yaml = @'
jellyfin:
  base_url: __URL__
  user_id: "__USER__"
  credential_target: lifeos-player-bridge/jellyfin
  allowed_libraries:
    - __LIBRARY__

player:
  executable: auto
  type: potplayer

path_mappings:
  - source: __SRC__
    target: __TGT__

playback:
  check_file_exists: true
  allowed_extensions:
    - mkv
    - mp4
    - avi
    - mov
    - m4v
'@
$yaml = $yaml.Replace('__URL__', $base).Replace('__USER__', $userId).Replace('__LIBRARY__', $Library).Replace('__SRC__', $MoviesSource).Replace('__TGT__', $MoviesTarget)
Set-Content -Path $ConfigPath -Value $yaml -Encoding UTF8
'config written: ' + $ConfigPath

'--- bridge self check ---'
& $stored -config $ConfigPath -verify