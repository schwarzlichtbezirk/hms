#!/bin/bash
go get -d github.com/schwarzlichtbezirk/wpk/util/build
go build -o $GOPATH/bin/wpkbuild.exe -v github.com/schwarzlichtbezirk/wpk/util/build
