@echo off
go env -w GOOS=windows GOARCH=amd64
cd /d %GOPATH%\src\github.com\schwarzlichtbezirk\hms
go build -o %GOPATH%/bin/hms.x64.exe -v ./cmd
