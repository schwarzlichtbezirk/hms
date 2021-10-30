@echo off
if not exist %GOPATH%\src\github.com\schwarzlichtbezirk\wpk (
    md %GOPATH%\src\github.com\schwarzlichtbezirk\wpk
    cd /d %GOPATH%\src\github.com\schwarzlichtbezirk\wpk
    git clone https://github.com/schwarzlichtbezirk/wpk.git
) else (
    cd /d %GOPATH%\src\github.com\schwarzlichtbezirk\wpk
    git pull https://github.com/schwarzlichtbezirk/wpk.git
)
go build -o %GOPATH%\bin\wpkbuild.exe -v github.com/schwarzlichtbezirk/wpk/util/build
