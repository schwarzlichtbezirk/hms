#!/bin/bash
mkdir -p $GOPATH/bin/cache
cp -ruv $(dirname $0)/../config $GOPATH/bin/config
$GOPATH/bin/wpkbuild.exe $(dirname $0)/hms-tiny.lua
