#!/bin/bash -u
# This script downloads fonts used on frontend.

asstdir=$(dirname $0)/../frontend/assets/iconfont
mkdir -pv "$asstdir"

# material-icons
# https://github.com/marella/material-icons
curl -L https://github.com/marella/material-icons/raw/main/iconfont/material-icons.css --output %asstdir%/material-icons.css
curl -L https://github.com/marella/material-icons/raw/main/iconfont/material-icons.woff2 --output $asstdir/material-icons.woff2
curl -L https://github.com/marella/material-icons/raw/main/iconfont/material-icons.woff --output $asstdir/material-icons.woff
