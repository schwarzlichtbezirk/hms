@echo off
go env -w GO111MODULE=auto GOARCH=amd64
go build -o %GOPATH%\bin\hms.x64.exe -v github.com/schwarzlichtbezirk/hms/run