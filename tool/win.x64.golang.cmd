@echo off
cd /d %GOPATH%\src\github.com\schwarzlichtbezirk\hms
go env -w GOOS=windows GOARCH=amd64
go build -o %GOPATH%\bin\hms.x64.exe -v github.com/schwarzlichtbezirk/hms/cmd
