@echo off
go get github.com/schwarzlichtbezirk/wpk/luawpk
go build -o %GOPATH%\bin\wpkbuild.exe -v github.com/schwarzlichtbezirk/wpk/util/build
