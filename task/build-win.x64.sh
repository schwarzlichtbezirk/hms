#!/bin/bash -u
# This script compiles project for Windows amd64.
# It produces static C-libraries linkage.

wd=$(realpath -s "$(dirname "$0")/..")

buildvers=$(git describe --tags)
buildtime=$(go run "$wd/task/timenow.go") # $(date -u +'%FT%TZ')

go env -w GOOS=windows GOARCH=amd64 CGO_ENABLED=1
go build -o $GOPATH/bin/hms.win.x64.exe -v -ldflags="-linkmode external -extldflags -static -X 'github.com/schwarzlichtbezirk/hms.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/hms.BuildTime=$buildtime'" $wd/cmd
