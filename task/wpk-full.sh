#!/bin/bash -u
# This script produces "hms-full.wpk" package with
# full set of skins and icons with all available formats.
# Can be useful for old browsers without webp support.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-full.lua")
