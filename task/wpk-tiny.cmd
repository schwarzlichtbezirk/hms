@echo off
rem This script produces "hms-tiny.wpk" package - minimal set with
rem two svg icons set. Can be used on lightweight systems.
%GOPATH%\bin\wpkbuild.exe %~dp0hms-tiny.lua
