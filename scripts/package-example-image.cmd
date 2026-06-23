@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0Package-ExampleImage.ps1" %*
