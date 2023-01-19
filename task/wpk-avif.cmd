@echo off
rem This script produces "hms-avif.wpk" package - full set with
rem avif and svg formats only, useful for modern browsers.
%GOPATH%\bin\wpkbuild.exe %~dp0hms-avif.lua
