#!/bin/bash -u
# This script produces "hms-app.wpk" package with
# js-code and html-templates used on frontend.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/pack-app.lua")
