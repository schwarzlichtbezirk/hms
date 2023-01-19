#!/bin/bash -u
# This script produces "hms-edge.wpk" package - full set with
# avif, webp and svg formats, useful for modern browsers.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-edge.lua")
