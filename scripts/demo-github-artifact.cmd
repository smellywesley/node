@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0Demo-GitHubArtifact.ps1" %*
exit /b %ERRORLEVEL%
