@echo off
go env -w GOOS=windows GOARCH=386
go build -o %GOPATH%\bin\hms.x86.exe -v github.com/schwarzlichtbezirk/hms/cmd
