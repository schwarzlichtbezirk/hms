@echo off
rem This script produces "hms-webp.wpk" package - full set with
rem webp and svg formats only, useful for modern browsers.
%GOPATH%\bin\wpkbuild.exe %~dp0hms-webp.lua
