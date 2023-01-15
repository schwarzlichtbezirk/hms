@echo off
rem This script compiles WPK-builder for any platform.
go get github.com/schwarzlichtbezirk/wpk/luawpk
go build -o %GOPATH%\bin\wpkbuild.exe -v github.com/schwarzlichtbezirk/wpk/util/build
