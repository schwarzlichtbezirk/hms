@echo off
rem This script convert all PNG icons to AVIF format.
rem Requires Node.js 14.15.0+.
rem see https://github.com/lovell/avif-cli

call :realpath %~dp0..\frontend\icon
set icondir=%retval:\=/%

npx avif --input="%icondir%/chakram/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/delta/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/junior/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/oxygen/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/senary/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/senary2/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/tulliana/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/ubuntu/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/whistlepuff/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%icondir%/xrabbit/*.png" --quality=50 --effort=9 --overwrite --verbose

exit /b 0

:realpath
set retval=%~f1
exit /b
