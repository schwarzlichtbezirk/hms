#!/bin/bash -u
# This script produces "hms-avif.wpk" package - full set with
# avif and svg formats only, useful for modern browsers.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-avif.lua")
