#!/bin/bash
cp -ruv $(dirname $0)/../config $GOPATH/bin/config
$GOPATH/bin/wpkbuild.exe $(dirname $0)/pack.lua
