@echo off
rem This script produces "hms-free.wpk" package with
rem set of icons with public license and allowed commercial usage.
%GOPATH%\bin\wpkbuild.exe %~dp0hms-free.lua
