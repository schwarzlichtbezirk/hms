@echo off
rem This script convert all PNG icons to AVIF format.
rem Requires Node.js 14.15.0+.
rem see https://github.com/lovell/avif-cli

call :realpath %~dp0..\frontend\icon
set wd=%retval:\=/%

npx avif --input="%wd%/chakram/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/delta/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/junior/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/oxygen/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/senary/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/senary2/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/tulliana/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/ubuntu/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/whistlepuff/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="%wd%/xrabbit/*.png" --quality=50 --effort=9 --overwrite --verbose

exit /b 0

:realpath
set retval=%~f1
exit /b
