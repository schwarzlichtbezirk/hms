@echo off
xcopy %GOPATH%\src\github.com\schwarzlichtbezirk\hms\conf %GOPATH%\bin\hms /f /d /i /s /e /k /y
%GOPATH%/bin/wpkbuild.x64.exe %GOPATH%/src/github.com/schwarzlichtbezirk/hms/tool/hms-tiny.lua
