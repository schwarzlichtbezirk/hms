#!/bin/bash
cd $(dirname $0)/..

buildvers=$(git describe --tags)
builddate=$(date +'%F')
buildtime=$(date +'%T')

go env -w GOOS=linux GOARCH=amd64
go build -o $GOPATH/bin/hms.linux.x64.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/hms.buildvers=$buildvers' -X 'github.com/schwarzlichtbezirk/hms.builddate=$builddate' -X 'github.com/schwarzlichtbezirk/hms.buildtime=$buildtime'" ./cmd
