#!/bin/bash -u
# This script compiles WPK-builder for any platform.

buildvers=$(git describe --tags)
buildtime=$(go run "$(dirname "$0")/timenow.go") # $(date -u +'%FT%TZ')

go get github.com/schwarzlichtbezirk/wpk/luawpk
go build -o $GOPATH/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=$buildtime'" github.com/schwarzlichtbezirk/wpk/util/build
