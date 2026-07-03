@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0Test-PilotReadiness.ps1" %*
exit /b %ERRORLEVEL%
