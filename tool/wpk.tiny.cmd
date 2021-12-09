@echo off
xcopy %GOPATH%\src\github.com\schwarzlichtbezirk\hms\config %GOPATH%\bin\hms /f /d /i /e /k /y
%GOPATH%/bin/wpkbuild.exe %GOPATH%/src/github.com/schwarzlichtbezirk/hms/tool/hms-tiny.lua
