#!/bin/bash -u
# This script produces "hms-webp.wpk" package - full set with
# webp and svg formats only, useful for modern browsers.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-webp.lua")
