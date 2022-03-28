#!/bin/bash
go get -d github.com/schwarzlichtbezirk/wpk/util/build
go build -o $GOPATH/bin/wpkbuild -v github.com/schwarzlichtbezirk/wpk/util/build
