@echo off
rem This script produces "hms-app.wpk" package with
rem js-code and html-templates used on frontend.
%GOPATH%\bin\wpkbuild.exe %~dp0pack-app.lua
