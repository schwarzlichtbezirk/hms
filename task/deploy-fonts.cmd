@echo off
rem This script downloads fonts used on frontend.
set asstdir=%~dp0..\frontend\assets\iconfont
mkdir %asstdir%

rem material-icons
rem https://github.com/marella/material-icons
curl -L https://github.com/marella/material-icons/raw/main/iconfont/material-icons.css --output %asstdir%/material-icons.css
curl -L https://github.com/marella/material-icons/raw/main/iconfont/material-icons.woff2 --output %asstdir%/material-icons.woff2
curl -L https://github.com/marella/material-icons/raw/main/iconfont/material-icons.woff --output %asstdir%/material-icons.woff
