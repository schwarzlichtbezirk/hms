@echo off
rem This script produces "hms-full.wpk" package with
rem full set of skins and icons with all available formats.
rem Can be useful for old browsers without webp support.
%GOPATH%\bin\wpkbuild.exe %~dp0hms-full.lua
