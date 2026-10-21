<#
.SYNOPSIS
  Configure this PowerShell session for LearnOS development on Windows.

.DESCRIPTION
  Sets up Go 1.26+, cgo C compiler (gcc or zig fallback), local caches,
  npm cache and the Playwright Chrome path. The environment is exported to the
  current process, so dot-source this file:

      . .\scripts\dev-env.ps1

  The harness sandbox blocks Go toolchain child pipes and npm writes outside the
  workspace. When running inside DeepSeek Harness, execute commands with
  full-access permission (pwsh sandbox_permissions=danger-full-access).
#>
$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$cacheRoot = Join-Path $repoRoot '.tmp'
$gocache = Join-Path $cacheRoot 'gocache'
$gopath = Join-Path $cacheRoot 'gopath'
$zigGlobal = Join-Path $cacheRoot 'zig-cache'
$zigLocal = Join-Path $cacheRoot 'zig-local-cache'
New-Item -ItemType Directory -Force $gocache, $gopath, $zigGlobal, $zigLocal | Out-Null

$env:LEARNOS_ROOT = $repoRoot
$env:GOCACHE = $gocache
$env:GOPATH = $gopath
$env:GOFLAGS = '-buildvcs=false'
$env:CGO_ENABLED = '1'
$env:npm_config_cache = Join-Path $cacheRoot 'npm-cache'

# Go 1.26+ is required by go.mod. Prefer this workstation's known SDK, then PATH,
# then a user-provided LEARNOS_GO.
$goCandidates = @()
if ($env:LEARNOS_GO) { $goCandidates += $env:LEARNOS_GO }
$goCandidates += 'D:\Project\Business\go1.27.0\bin\go.exe'
$goOnPath = Get-Command go -ErrorAction SilentlyContinue
if ($goOnPath) { $goCandidates += $goOnPath.Source }
$goExe = $null
foreach ($candidate in $goCandidates) {
  if ($candidate -and (Test-Path $candidate)) { $goExe = $candidate; break }
}
if (-not $goExe) {
  throw 'Go 1.26+ not found. Install Go or set LEARNOS_GO to the go.exe path.'
}
$env:LEARNOS_GO = $goExe
$goVersion = (& $goExe version 2>&1) -join ' '
if ($goVersion -notmatch 'go1\.(2[6-9]|[3-9][0-9])') {
  Write-Warning "Go version may be too old for go.mod (found: $goVersion)"
}

# cgo needs a C compiler for the SQLite driver. Use gcc when available; otherwise
# use the ziglang wheel's `zig cc` (verified working on this workstation).
$cc = $null
$gcc = Get-Command gcc -ErrorAction SilentlyContinue
if ($gcc) {
  $cc = $gcc.Source
} else {
  $zig = $null
  $zigOnPath = Get-Command zig -ErrorAction SilentlyContinue
  if ($zigOnPath) { $zig = $zigOnPath.Source }
  if (-not $zig) {
    $knownZig = 'C:\Users\HP\AppData\Local\Programs\Python\Python312\Lib\site-packages\ziglang\zig.exe'
    if (Test-Path $knownZig) { $zig = $knownZig }
  }
  if (-not $zig) {
    try {
      $probe = & python -c "import ziglang, os; print(os.path.join(os.path.dirname(ziglang.__file__), 'zig.exe'))" 2>$null
      if ($LASTEXITCODE -eq 0 -and $probe) {
        $candidate = $probe.Trim()
        if (Test-Path $candidate) { $zig = $candidate }
      }
    } catch { }
  }
  if (-not $zig -or -not (Test-Path $zig)) {
    throw 'No C compiler found for cgo. Install mingw-w64/gcc, or: python -m pip install ziglang'
  }
  $cc = "$zig cc"
  $env:ZIG_GLOBAL_CACHE_DIR = $zigGlobal
  $env:ZIG_LOCAL_CACHE_DIR = $zigLocal
}
$env:CC = $cc

# Playwright has no downloaded browser on this workstation; use system Chrome.
$chrome = 'C:\Program Files\Google\Chrome\Application\chrome.exe'
if (Test-Path $chrome) {
  $env:PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = $chrome
}

Write-Host 'LearnOS development environment:'
Write-Host ("  repo = " + $repoRoot)
Write-Host ("  go   = " + $goExe + " (" + $goVersion + ")")
Write-Host ("  cc   = " + $cc)
Write-Host ("  node = " + (& node -v 2>$null))
Write-Host ("  e2e  = " + $(if ($env:PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH) { $env:PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } else { 'not found (set PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH)' }))
Write-Host 'Next: .\scripts\verify-local.ps1  |  .\scripts\deploy-local.ps1'
