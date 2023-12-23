@echo off
rem This script compiles project for Linux amd64.
rem It produces static C-libraries linkage.
set wd=%~dp0..

if not exist "%GOPATH%\bin\cache" (mkdir "%GOPATH%\bin\cache")
xcopy %wd%\confdata %GOPATH%\bin\config /f /d /i /e /k /y

for /F "tokens=*" %%g in ('git describe --tags') do (set buildvers=%%g)
for /F "tokens=*" %%g in ('go run %~dp0/timenow.go') do (set buildtime=%%g)

go env -w GOOS=linux GOARCH=amd64 CGO_ENABLED=1
go build -o %GOPATH%/bin/hms_linux_x64 -v -ldflags="-linkmode external -extldflags -static -X 'github.com/schwarzlichtbezirk/hms/config.BuildVers=%buildvers%' -X 'github.com/schwarzlichtbezirk/hms/config.BuildTime=%buildtime%'" %wd%
