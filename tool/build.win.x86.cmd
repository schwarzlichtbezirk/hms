@echo off
go env -w GOOS=windows GOARCH=386
cd /d %GOPATH%\src\github.com\schwarzlichtbezirk\hms
go build -o %GOPATH%/bin/hms.x86.exe -v ./cmd
