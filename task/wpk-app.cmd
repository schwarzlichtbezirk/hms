@echo off
rem This script produces "hms-app.wpk" package with
rem js-code and html-templates used on frontend.
if not exist "%GOPATH%\bin\cache" (mkdir "%GOPATH%\bin\cache")
xcopy %~dp0..\config %GOPATH%\bin\config /f /d /i /e /k /y
%GOPATH%\bin\wpkbuild.exe %~dp0pack-app.lua
