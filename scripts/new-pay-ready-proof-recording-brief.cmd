@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0New-PayReadyProofRecordingBrief.ps1" %*
exit /b %ERRORLEVEL%
