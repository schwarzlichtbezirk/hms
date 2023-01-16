@echo off
rem This script convert all PNG icons to AVIF format.
rem Requires Node.js 14.15.0+.
rem see https://github.com/lovell/avif-cli

call :realpath %~dp0..\frontend\icon
set icondir=%retval%

for /r %icondir%\chakram %%f in (*.png) do (
	npx avif --input=%%f -o %%~dpf --overwrite --quality=40 --effort=9
)

exit /b 0

:realpath
set retval=%~f1
exit /b
