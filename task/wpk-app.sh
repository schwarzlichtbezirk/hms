#!/bin/bash -u
mkdir -p "$GOPATH/bin/cache"
cp -ruv "$(dirname $0)/../config" "$GOPATH/bin/config"
$GOPATH/bin/wpkbuild.exe $(dirname $0)/pack-app.lua
