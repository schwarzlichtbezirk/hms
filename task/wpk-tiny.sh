#!/bin/bash -u
# This script produces "hms-tiny.wpk" package - minimal set with
# two svg icons set. Can be used on lightweight systems.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-tiny.lua")
