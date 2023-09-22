@echo off
rem This script compiles WPK-builder for any platform.

for /F "tokens=*" %%g in ('git describe --tags') do (set buildvers=%%g)
for /F "tokens=*" %%g in ('go run %~dp0/timenow.go') do (set buildtime=%%g)

go get github.com/schwarzlichtbezirk/wpk/luawpk
go build -o %GOPATH%/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=%buildvers%' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=%buildtime%'" github.com/schwarzlichtbezirk/wpk/util/build
