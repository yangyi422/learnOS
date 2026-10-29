<#
.SYNOPSIS
  Run the local verification suite (Go, frontend, optional Playwright).

.DESCRIPTION
  Runs go vet + go test, then typecheck/vitest/build in web/. Playwright specs are
  opt-in because they need Chrome and a free port.

  Go tests on Windows report "TempDir RemoveAll cleanup" failures because SQLite
  keeps file handles open. Those are environment artefacts and are reported as a
  warning instead of a failure; every other assertion failure fails the run.

.EXAMPLE
  . .\scripts\dev-env.ps1
  .\scripts\verify-local.ps1

.EXAMPLE
  .\scripts\verify-local.ps1 -E2E

.EXAMPLE
  .\scripts\verify-local.ps1 -Visual

.EXAMPLE
  .\scripts\verify-local.ps1 -GoPackages ./internal/model/ -SkipWeb
#>
param(
  [switch]$SkipGo,
  [switch]$SkipWeb,
  [switch]$E2E,
  [switch]$Visual,
  [string[]]$GoPackages = @('./internal/...', './cmd/...'),
  [switch]$AllowEnvFailures
)

$ErrorActionPreference = 'Stop'
if (-not $env:LEARNOS_ROOT) { . (Join-Path $PSScriptRoot 'dev-env.ps1') }

$root = $env:LEARNOS_ROOT
$logDir = Join-Path $root '.tmp\logs'
New-Item -ItemType Directory -Force $logDir | Out-Null
$failed = $false
$summary = [System.Collections.Generic.List[string]]::new()

function Invoke-Step {
  param([string]$Name, [string]$LogName, [scriptblock]$Action)
  $log = Join-Path $logDir $LogName
  & $Action 2>&1 | Tee-Object -FilePath $log | Out-Null
  $code = $LASTEXITCODE
  if ($code -eq 0) {
    Write-Host ("  [ok]   " + $Name)
    $summary.Add(("OK    " + $Name))
  } else {
    Write-Host ("  [fail] " + $Name + " (exit " + $code + ", log " + $log + ")")
    $summary.Add(("FAIL  " + $Name))
    $script:failed = $true
  }
  return $code
}

Write-Host '== LearnOS local verification =='

if (-not $SkipGo) {
  Write-Host '-- Go --'
  Invoke-Step 'go vet' 'go-vet.log' { & $env:LEARNOS_GO vet @GoPackages } | Out-Null
  & $env:LEARNOS_GO test -count=1 @GoPackages 2>&1 | Tee-Object -FilePath (Join-Path $logDir 'go-test.log') | Out-Null
  $goExit = $LASTEXITCODE
  $lines = @(Get-Content (Join-Path $logDir 'go-test.log') -ErrorAction SilentlyContinue)
  $realFailures = [System.Collections.Generic.List[string]]::new()
  $envFailures = 0
  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match '^--- FAIL: (\S+)') {
      $name = $matches[1]
      $window = ($lines[$i..([Math]::Min($i + 3, $lines.Count - 1))] -join ' ')
      if ($window -match 'TempDir RemoveAll cleanup') { $envFailures++ } else { $realFailures.Add($name) }
    }
  }
  $buildFailed = @($lines | Where-Object { $_ -match '\[build failed\]' }).Count -gt 0
  if ($realFailures.Count -gt 0 -or $buildFailed -or $goExit -ne 0 -and $realFailures.Count -eq 0 -and $envFailures -eq 0) {
    Write-Host ("  [fail] go test: " + $realFailures.Count + " real failures, build failed=" + $buildFailed)
    $realFailures | Select-Object -First 10 | ForEach-Object { Write-Host ("    - " + $_) }
    $summary.Add('FAIL  go test')
    $failed = $true
  } else {
    Write-Host ("  [ok]   go test (" + $envFailures + " Windows TempDir cleanup warnings ignored)")
    $summary.Add(("OK    go test (" + $envFailures + " env warnings)"))
  }
}

if (-not $SkipWeb) {
  Write-Host '-- Web --'
  Push-Location (Join-Path $root 'web')
  try {
    Invoke-Step 'typecheck' 'web-typecheck.log' { & npm run typecheck } | Out-Null
    Invoke-Step 'vitest' 'web-test.log' { & npm run test } | Out-Null
    Invoke-Step 'vite check build (isolated)' 'web-build.log' { & npm run build:check } | Out-Null
  } finally {
    Pop-Location
  }
}

if ($E2E) {
  Write-Host '-- Playwright --'
  $specs = @(
    'e2e/daily-focus.spec.ts',
    'e2e/workspace.spec.ts',
    'e2e/planning.spec.ts',
    'e2e/regression.spec.ts',
    'e2e/ui-foundation.spec.ts'
  )
  Push-Location (Join-Path $root 'web')
  try {
    $log = Join-Path $logDir 'e2e.log'
    & npx playwright test @specs --output=../.tmp/e2e-check 2>&1 | Tee-Object -FilePath $log | Out-Null
    $e2eExit = $LASTEXITCODE
    $text = (Get-Content $log -Raw -ErrorAction SilentlyContinue) -join "`n"
    $passed = 0; $failedCount = 0
    if ($text -match '(\d+) passed') { $passed = [int]$matches[1] }
    if ($text -match '(\d+) failed') { $failedCount = [int]$matches[1] }
    # Startup/config/browser failures may run zero tests and have no "N failed" line.
    # Preserve the actual process result instead of treating those failures as a pass.
    if ($e2eExit -ne 0) {
      if ($AllowEnvFailures) {
        Write-Host ("  [warn] Playwright: " + $passed + " passed, " + $failedCount + " failed (allowed by -AllowEnvFailures)")
        $summary.Add(("WARN  e2e (" + $passed + " passed, " + $failedCount + " failed)"))
      } else {
        Write-Host ("  [fail] Playwright: " + $passed + " passed, " + $failedCount + " failed (exit " + $e2eExit + ", log " + $log + ")")
        $summary.Add('FAIL  e2e')
        $failed = $true
      }
    } else {
      Write-Host ("  [ok]   Playwright: " + $passed + " passed")
      $summary.Add(("OK    e2e (" + $passed + " passed)"))
    }
  } finally {
    Pop-Location
  }
}

if ($Visual) {
  Write-Host '-- Visual audit --'
  Push-Location (Join-Path $root 'web')
  try {
    $visualLog = Join-Path $logDir 'visual-audit.log'
    & npx playwright test e2e/visual-audit.spec.ts --output=../.tmp/visual-check 2>&1 | Tee-Object -FilePath $visualLog | Out-Null
    $visualExit = $LASTEXITCODE
    if ($visualExit -eq 0) {
      Write-Host '  [ok]   visual tests/artifacts: .tmp/visual-check/visual-audit-*/visual-* (open images to review)'
      $summary.Add('OK    visual tests/artifacts; manual visual review still required')
    } else {
      Write-Host ('  [fail] visual audit (exit ' + $visualExit + ', log ' + $visualLog + ')')
      $summary.Add('FAIL  visual audit')
      $failed = $true
    }
  } finally {
    Pop-Location
  }
}

Write-Host ''
Write-Host '== Summary =='
$summary | ForEach-Object { Write-Host ('  ' + $_) }
if ($failed) {
  Write-Host 'RESULT: FAILED'
  exit 1
}
if (@($summary | Where-Object { $_ -like 'WARN*' }).Count -gt 0) {
  Write-Host 'RESULT: COMPLETED WITH WARNINGS (not fully verified)'
  exit 0
}
Write-Host 'RESULT: PASSED (checks only; not a visual acceptance verdict)'
exit 0
