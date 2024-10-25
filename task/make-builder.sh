#!/bin/bash -u
# This script compiles WPK-builder for any platform.

buildvers=$(git describe --tags)
# See https://tc39.es/ecma262/#sec-date-time-string-format
# time format acceptable for Date constructors.
buildtime=$(date +'%FT%T.%3NZ')

go get github.com/schwarzlichtbezirk/wpk/luawpk
go build -o $GOPATH/bin/wpkbuild.exe -v -ldflags="-X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildVers=$buildvers' -X 'github.com/schwarzlichtbezirk/wpk/luawpk.BuildTime=$buildtime'" github.com/schwarzlichtbezirk/wpk/util/build
