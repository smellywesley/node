@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0SecurityAudit.ps1" %*
