<#
.SYNOPSIS
  Build player-bridge.exe for the current Windows machine.
.DESCRIPTION
  Produces dist\player-bridge.exe (and prints the architecture).
  Run from anywhere: paths are resolved from the script location.
#>
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
New-Item -ItemType Directory -Force $dist | Out-Null

$go = 'go'
if (-not (Get-Command $go -ErrorAction SilentlyContinue)) {
  $candidate = 'D:\Project\Business\go1.27.0\bin\go.exe'
  if (Test-Path $candidate) { $go = $candidate } else { throw 'Go toolchain not found; install Go 1.26+ or set LEARNOS_GO.' }
}

Push-Location $root
try {
  # -H=windowsgui: a protocol launch must not flash a console window; the binary re-attaches
  # to a parent console when started from a terminal, so -verify / -play still print.
  & $go build -trimpath -ldflags '-s -w -H=windowsgui' -o (Join-Path $dist 'player-bridge.exe') ./cmd/player-bridge
  if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
  $exe = Join-Path $dist 'player-bridge.exe'
  'built: ' + $exe
  'arch : ' + [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
} finally { Pop-Location }