#!/bin/bash -u

wd=$(realpath -s "$(dirname "$0")/..")

buildvers=$(git describe --tags)
builddate=$(date +'%F')
buildtime=$(date +'%T')

go env -w GOOS=windows GOARCH=amd64 CGO_ENABLED=1
go build -o $GOPATH/bin/hms.win.x64.exe -v -ldflags="-linkmode external -extldflags -static -X 'github.com/schwarzlichtbezirk/hms.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/hms.BuildDate=$builddate' -X 'github.com/schwarzlichtbezirk/hms.BuildTime=$buildtime'" $wd/cmd
