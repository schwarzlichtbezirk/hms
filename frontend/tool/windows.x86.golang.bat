@echo off
go env -w GO111MODULE=auto GOARCH=386
go build -o %GOPATH%\bin\hms.x86.exe -v github.com/schwarzlichtbezirk/hms/run