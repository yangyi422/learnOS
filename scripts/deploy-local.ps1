<#
.SYNOPSIS
  Deploy the current LearnOS checkout through the WSL2 Docker engine.

.DESCRIPTION
  Runs deploy/deploy.sh inside the WSL deployment checkout. Use -SyncChangedFiles
  to copy the uncommitted files of this Windows workspace to the WSL checkout
  first (the deployment checkout is a separate git clone). Prefer committing and
  pushing, then use -Pull, to avoid file syncing entirely.

.PARAMETER Distro
  WSL distribution name. Default: Ubuntu-24.04.

.PARAMETER DeployPath
  Linux path of the WSL deployment checkout. Default: /home/hp/projects/learnos-starter.

.PARAMETER SyncChangedFiles
  Copy the files listed by `git status` from this workspace to DeployPath.
  .env, data, backups and node_modules are never copied.

.PARAMETER Pull
  Run `git pull --ff-only` in the deployment checkout before building.

.PARAMETER NoVerify
  Skip post-deploy checks.

.EXAMPLE
  . .\scripts\dev-env.ps1
  .\scripts\deploy-local.ps1 -SyncChangedFiles

.EXAMPLE
  .\scripts\deploy-local.ps1 -Pull
#>
param(
  [string]$Distro = 'Ubuntu-24.04',
  [string]$DeployPath = '/home/hp/projects/learnos-starter',
  [switch]$SyncChangedFiles,
  [switch]$Pull,
  [switch]$NoVerify
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function ConvertTo-WslPath {
  param([string]$WindowsPath)
  $full = [System.IO.Path]::GetFullPath($WindowsPath)
  if ($full.Length -lt 2 -or $full[1] -ne ':') { throw "Not a drive path: $WindowsPath" }
  $drive = $full.Substring(0, 1).ToLowerInvariant()
  $rest = $full.Substring(2).Replace('\', '/')
  return "/mnt/$drive$rest"
}

function Invoke-Wsl {
  param([string]$Command)
  & wsl.exe -d $Distro -- bash -lc $Command
  if ($LASTEXITCODE -ne 0) { throw ("WSL command failed (" + $LASTEXITCODE + "): " + $Command) }
}

Write-Host ("== LearnOS WSL deployment: " + $Distro + " " + $DeployPath + " ==")
& wsl.exe -d $Distro -- true
if ($LASTEXITCODE -ne 0) { throw "WSL distro '$Distro' is not available." }

if ($SyncChangedFiles) {
  Write-Host '-- syncing changed files from this workspace --'
  $tmp = Join-Path $root '.tmp'
  New-Item -ItemType Directory -Force $tmp | Out-Null
  $listPath = Join-Path $tmp 'deploy-files.txt'
  $files = git status --porcelain | ForEach-Object {
    $path = $_.Substring(3).Trim()
    if ($path -match ' -> ') { $path = ($path -split ' -> ')[1] }
    $path
  } | Where-Object {
    $_ -and $_ -notmatch '^\.tmp/' -and $_ -notmatch '\.dsh-ops-update-' -and $_ -notmatch '\.env$'
  } | Sort-Object -Unique
  if (-not $files -or $files.Count -eq 0) {
    Write-Host '  no changed files to sync.'
  } else {
    $files | Set-Content -Path $listPath -Encoding ascii
    $wslRoot = ConvertTo-WslPath $root
    $wslList = ConvertTo-WslPath $listPath
    $helper = Join-Path $tmp 'deploy-sync.sh'
    $helperText = @'
#!/usr/bin/env bash
set -euo pipefail
DST="$1"; SRC="$2"; LIST="$3"
cd "$DST"
git config core.fileMode false
rsync -a --exclude '.env' --files-from="$LIST" "$SRC"/ "$DST"/
while IFS= read -r f; do
  if [ -n "$f" ] && [ -f "$f" ]; then sed -i 's/\r$//' "$f"; fi
done < "$LIST"
echo "  synced $(grep -c . "$LIST") files"
git status --short | head -40
'@
    [System.IO.File]::WriteAllText($helper, $helperText.Replace("`r`n", "`n"))
    $wslHelper = ConvertTo-WslPath $helper
    & wsl.exe -d $Distro -- bash $wslHelper $DeployPath $wslRoot $wslList
    if ($LASTEXITCODE -ne 0) { throw 'file sync failed.' }
  }
}

$flags = @()
if ($Pull) { $flags += '--pull' }
if ($NoVerify) { $flags += '--no-verify' }
$flagText = if ($flags.Count) { ' ' + ($flags -join ' ') } else { '' }
Write-Host '-- building and deploying --'
Invoke-Wsl ("cd " + $DeployPath + " && bash deploy/deploy.sh" + $flagText)

Write-Host ''
Write-Host 'Deployment finished. Open http://127.0.0.1:8080 (URL follows .env APP_BIND_HOST/APP_PORT).'
