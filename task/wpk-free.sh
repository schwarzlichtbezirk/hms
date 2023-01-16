#!/bin/bash -u
# This script produces "hms-free.wpk" package with
# set of icons with public license and allowed commercial usage.
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-free.lua")
