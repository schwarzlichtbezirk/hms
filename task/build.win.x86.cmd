@echo off
cd /d %~dp0..

for /F "tokens=*" %%g in ('git describe --tags') do (set buildvers=%%g)
set builddate=%date%
set buildtime=%time:~0,8%
if "%buildtime:~0,1%" == " " set buildtime=0%buildtime:~1,7%

go env -w GOOS=windows GOARCH=386
go build -o %GOPATH%/bin/hms.win.x86.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/hms.BuildVers=%buildvers%' -X 'github.com/schwarzlichtbezirk/hms.BuildDate=%builddate%' -X 'github.com/schwarzlichtbezirk/hms.BuildTime=%buildtime%'" ./cmd
