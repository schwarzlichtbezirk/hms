@echo off
xcopy %~dp0..\config %GOPATH%\bin\hms /f /d /i /e /k /y
%GOPATH%/bin/wpkbuild.exe %~dp0pack.lua
