#!/bin/bash -u
# This script compiles project for Linux amd64.
# It produces static C-libraries linkage.

wd=$(realpath -s "$(dirname "$0")/..")

buildvers=$(git describe --tags)
buildtime=$(go run "$wd/task/timenow.go") # $(date -u +'%FT%TZ')

go env -w GOOS=linux GOARCH=amd64 CGO_ENABLED=1
go build -o $GOPATH/bin/hms.linux.x64.exe -v -ldflags="-linkmode external -extldflags -static -X 'github.com/schwarzlichtbezirk/hms/cfg.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/hms/cfg.BuildTime=$buildtime'" $wd/cmd
