#!/bin/bash -u
# This script produces "hms-app.wpk" package with
# js-code and html-templates used on frontend.
mkdir -p "$GOPATH/bin/cache"
cp -ruv "$(dirname $0)/../confdata" "$GOPATH/bin/config"
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/pack-app.lua")
