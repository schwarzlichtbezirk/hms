#!/bin/bash -u
$GOPATH/bin/wpkbuild.exe $(realpath -s "$(dirname $0)/hms-free.lua")
