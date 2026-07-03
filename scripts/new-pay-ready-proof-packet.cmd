@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0New-PayReadyProofPacket.ps1" %*
exit /b %ERRORLEVEL%
