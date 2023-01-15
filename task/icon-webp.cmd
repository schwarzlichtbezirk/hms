@echo off
rem This script convert all PNG icons to WEBP format.
rem Downloads WebP tools if it necessary.
rem see https://developers.google.com/speed/webp/download

call :realpath %~dp0..\frontend\icon
set icondir=%retval%

set tooldir=%~d0\devtools

set webpver=1.3.0

set cwebp=%tooldir%\cwebp.exe
if exist %cwebp% (
	goto cwebpexist
)
set cwebp=%tooldir%\libwebp-%webpver%-windows-x64\bin\cwebp.exe
if exist %cwebp% (
	goto cwebpexist
)
echo WebP encoder tool does not found, downloading it into '%tooldir%' folder.
mkdir %tooldir%
curl -L --output %tooldir%\libwebp-%webpver%-windows-x64.zip^
 https://storage.googleapis.com/downloads.webmproject.org/releases/webp/libwebp-%webpver%-windows-x64.zip
tar -xf %tooldir%\libwebp-%webpver%-windows-x64.zip -C %tooldir%

:cwebpexist

for /r %icondir%\chakram %%f in (*.jpg) do (
	%cwebp%  -q 80 -alpha_filter best -m 6 -af -hint picture -short %%f -o %%~dpnf.webp
)

exit /b 0

:realpath
set retval=%~f1
exit /b
