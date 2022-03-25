#!/bin/bash
cp -ruv $(dirname $0)/../config $GOPATH/bin/config
$GOPATH/bin/wpkbuild $(dirname $0)/hms-tiny.lua
