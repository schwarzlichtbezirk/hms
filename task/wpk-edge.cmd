@echo off
rem This script produces "hms-edge.wpk" package - full set with
rem avif, webp and svg formats, useful for modern browsers.
%GOPATH%\bin\wpkbuild.exe %~dp0hms-edge.lua
