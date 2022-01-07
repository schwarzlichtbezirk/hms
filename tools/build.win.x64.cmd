@echo off
cd /d %~dp0..
go env -w GOOS=windows GOARCH=amd64
go build -o %GOPATH%/bin/hms.x64.exe -v ./cmd
