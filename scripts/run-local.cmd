@echo off
rem Launch LearnOS natively. Kept as a .cmd so it also works where the
rem PowerShell execution policy refuses unsigned .ps1 files: the policy is
rem bypassed for this child process only. Extra arguments are passed through,
rem e.g. run-local.cmd -Port 8081 -DataDir .tmp\data-probe
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0run-local.ps1" %*
exit /b %ERRORLEVEL%
