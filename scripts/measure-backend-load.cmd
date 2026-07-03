@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0Measure-BackendLoad.ps1" %*
exit /b %ERRORLEVEL%
