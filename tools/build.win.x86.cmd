@echo off
cd /d %~dp0..
go env -w GOOS=windows GOARCH=386
go build -o %GOPATH%/bin/hms.x86.exe -v ./cmd
