@echo off
xcopy %~dp0..\config %GOPATH%\bin\config /f /d /i /e /k /y
%GOPATH%/bin/wpkbuild.exe %~dp0hms-tiny.lua
