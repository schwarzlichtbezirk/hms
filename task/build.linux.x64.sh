#!/bin/bash
cd $(dirname $0)/..

git describe --tags > buildvers.txt # puts version to file for docker image building
buildvers=`cat buildvers.txt`
builddate=$(date +'%F')
buildtime=$(date +'%T')

go env -w GOOS=linux GOARCH=amd64
go build -o $GOPATH/bin/hms.x64 -v -ldflags="-X 'github.com/schwarzlichtbezirk/hms.buildvers=%buildvers%' -X 'github.com/schwarzlichtbezirk/hms.builddate=%builddate%' -X 'github.com/schwarzlichtbezirk/hms.buildtime=%buildtime%'" ./cmd
