<#
.SYNOPSIS
  Build and run LearnOS natively on Windows: one learnos.exe plus a SQLite file.

.DESCRIPTION
  This is the local runtime path since WSL2 and Docker are not used on this
  workstation. It compiles the Vue bundle when the embedded assets are missing,
  compiles cmd/server with cgo (gcc, or zig cc from dev-env.ps1) and starts the
  binary. Configuration is read from .env (KEY=VALUE; the same file docker
  compose used), while the listen address and the data directory are resolved
  to Windows paths here.

.PARAMETER Port
  Listen port on this machine. Default: APP_PORT from .env, else 8080.

.PARAMETER Environment
  development or production. Default: APP_ENV from .env, else development.

.PARAMETER DataDir
  Directory holding learnos.db and backups. Default: data\<environment>.

.PARAMETER BindHost
  Interface to bind. Default: 127.0.0.1.

.PARAMETER RebuildWeb
  Rebuild the Vue bundle into internal/webassets/dist before compiling.

.PARAMETER NoBuild
  Skip compilation and run the existing bin\learnos.exe.

.PARAMETER Detached
  Start in the background, wait for /health, then print the URL and the PID.

.EXAMPLE
  .\scripts\run-local.ps1

.EXAMPLE
  .\scripts\run-local.ps1 -Environment production -Port 8888 -RebuildWeb

.EXAMPLE
  .\scripts\run-local.ps1 -Detached -DataDir .tmp\data-probe
#>
param(
  [int]$Port = 0,
  [ValidateSet('development', 'production')]
  [string]$Environment = '',
  [string]$DataDir = '',
  [string]$BindHost = '127.0.0.1',
  [switch]$RebuildWeb,
  [switch]$NoBuild,
  [switch]$Detached
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

# docker compose read .env through env_file; natively the same file is exported.
$dotenv = @{}
$envFile = Join-Path $root '.env'
if (Test-Path $envFile) {
  foreach ($line in Get-Content $envFile) {
    if ($line -match '^\s*#') { continue }
    if ($line -match '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*?)\s*$') {
      $value = $Matches[2]
      if ($value.Length -ge 2 -and (($value[0] -eq "'" -and $value[-1] -eq "'") -or ($value[0] -eq '"' -and $value[-1] -eq '"'))) {
        $value = $value.Substring(1, $value.Length - 2)
      }
      $dotenv[$Matches[1]] = $value
    }
  }
}

if (-not $Environment) { $Environment = if ($dotenv['APP_ENV']) { $dotenv['APP_ENV'] } else { 'development' } }
if ($Port -le 0) { $Port = if ($dotenv['APP_PORT']) { [int]$dotenv['APP_PORT'] } else { 8080 } }
if (-not $DataDir) {
  $DataDir = Join-Path $root (Join-Path 'data' $(if ($Environment -eq 'production') { 'prod' } else { 'dev' }))
} elseif (-not [System.IO.Path]::IsPathRooted($DataDir)) {
  $DataDir = Join-Path $root $DataDir
}

foreach ($key in $dotenv.Keys) {
  # Container-only keys: the container paths and host port mapping mean nothing here.
  if ($key -in @('APP_ADDR', 'DATA_HOST_DIR', 'BACKUP_HOST_DIR', 'BACKUP_DIR')) { continue }
  if (-not (Test-Path "Env:$key")) { Set-Item -Path "Env:$key" -Value $dotenv[$key] }
}

$env:APP_ENV = $Environment
$env:APP_ADDR = "${BindHost}:${Port}"
$env:APP_DATA_DIR = $DataDir
if (-not (Test-Path 'Env:BACKUP_DIR')) { $env:BACKUP_DIR = Join-Path $root 'backups' }

if (-not $NoBuild) {
  . (Join-Path $PSScriptRoot 'dev-env.ps1')
  if ($RebuildWeb -or -not (Test-Path (Join-Path $root 'internal/webassets/dist/index.html'))) {
    Write-Host '-- building the Vue bundle --'
    Push-Location (Join-Path $root 'web')
    try { & npm run build; if ($LASTEXITCODE -ne 0) { throw 'npm run build failed' } } finally { Pop-Location }
  }
  $exe = Join-Path $root 'bin\learnos.exe'
  New-Item -ItemType Directory -Force (Split-Path -Parent $exe) | Out-Null
  Write-Host '-- building bin\learnos.exe --'
  & $env:LEARNOS_GO build -trimpath -ldflags '-s -w' -o $exe ./cmd/server
  if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
} else {
  $exe = Join-Path $root 'bin\learnos.exe'
  if (-not (Test-Path $exe)) { throw "bin\learnos.exe is missing; run without -NoBuild first." }
}

if ($Environment -eq 'production' -and (-not $env:APP_USERNAME -or -not $env:APP_PASSWORD_HASH)) {
  Write-Warning 'production without APP_USERNAME/APP_PASSWORD_HASH: every API call will fail authentication. Set both in .env.'
}

Write-Host ("== LearnOS " + $Environment + " -> http://" + $BindHost + ":" + $Port + "  data=" + $DataDir + " ==")

if (-not $Detached) {
  & $exe
  exit $LASTEXITCODE
}

$logDir = Join-Path $root '.tmp'
New-Item -ItemType Directory -Force $logDir | Out-Null
$outLog = Join-Path $logDir 'run-local.log'
$errLog = Join-Path $logDir 'run-local.err.log'
$proc = Start-Process -FilePath $exe -WorkingDirectory $root -PassThru -RedirectStandardOutput $outLog -RedirectStandardError $errLog
$deadline = (Get-Date).AddSeconds(60)
$healthy = $false
while ((Get-Date) -lt $deadline) {
  Start-Sleep -Milliseconds 500
  try {
    $probe = Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 ("http://127.0.0.1:" + $Port + "/health")
    if ($probe.StatusCode -eq 200) { $healthy = $true; break }
  } catch { }
  if ($proc.HasExited) { break }
}
if (-not $healthy) {
  if (Test-Path $errLog) { Get-Content $errLog -Tail 20 }
  if (-not $proc.HasExited) { Stop-Process -Id $proc.Id -Force }
  throw 'the server did not answer /health within 60s'
}
Set-Content -Path (Join-Path $logDir 'run-local.pid') -Value $proc.Id
Write-Host ("health ok: " + $probe.Content)

# Same post-deploy checks as deploy/deploy.sh: served asset version and the
# unauthenticated API guard, so a stale embedded bundle is visible here.
$index = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 ("http://127.0.0.1:" + $Port + "/")
$asset = [regex]::Match($index.Content, 'assets/index-[A-Za-z0-9_-]+\.js').Value
Write-Host ("index " + $index.StatusCode + " len=" + $index.Content.Length + " " + $asset)
# development injects a development principal, so the unauthenticated guard only exists in production.
if ($Environment -eq 'production') {
  try {
    Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 ("http://127.0.0.1:" + $Port + "/api/v1/courses") | Out-Null
    Write-Host 'unauth api: unexpected 2xx'
  } catch { Write-Host ("unauth api: " + [int]$_.Exception.Response.StatusCode) }
}
Write-Host ("pid " + $proc.Id + " (stop: Stop-Process -Id " + $proc.Id + ")  log: " + $outLog)
