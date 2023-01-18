#!/bin/bash -u
# This script convert all PNG icons to WEBP format.
# Downloads WebP tools if it necessary.
# see https://developers.google.com/speed/webp/download

wd=$(realpath -s "$(dirname "$0")/../frontend/icon")

webpver=1.3.0

# detect host platform configuration
arc=$(uname -m)
if [[ "$OSTYPE" == "linux-gnu"* ]] && [[ "$arc" == "x86_64" ]]; then
	# some Linux-based platform detected
	cfg="linux-x86-64"
	exe=""
elif [[ "$OSTYPE" == "darwin"* ]] && [[ "$arc" == "x86_64" ]]; then
	# Mac OSX, x86-64
	cfg="mac-x86-64"
	exe=""
elif [[ "$OSTYPE" == "darwin"* ]] && [[ "$arc" == "aarch64" ]]; then
	# Mac OSX, arm64
	cfg="mac-arm64"
	exe=""
elif [[ "$OSTYPE" == "msys" ]] && [[ "$arc" == "x86_64" ]]; then
	# Lightweight shell and GNU utilities compiled for Windows (part of MinGW)
	cfg="windows-x64"
	exe=".exe"
elif [[ "$OSTYPE" == "cygwin" ]] && [[ "$arc" == "x86_64" ]]; then
	# POSIX compatibility layer and Linux environment emulation for Windows
	cfg="windows-x64"
	exe=".exe"
else
	echo "current platform does not supported"
	exit 1
fi

# find devtools directory
tmp=$wd
while [ "$tmp" != "/" ]; do
	if [ -d "$tmp/devtools" ]; then
		tooldir="$tmp/devtools"
		break
	fi
	tmp=$(realpath -s "$tmp/..")
done
unset tmp
if [ -z "$tooldir" ]; then
	tooldir="~/devtools"
	mkdir -pv "$tooldir"
fi

# check up that tools are downloaded
cwebp="$tooldir/cwebp$exe"
if [ ! -f "$cwebp" ]; then
	cwebp="$tooldir/libwebp-$webpver-$cfg/bin/cwebp$exe"
	if [ ! -f "$cwebp" ]; then
		echo "WebP encoder tool does not found, downloading it into '$tooldir' folder."
		curl -L --output "$tooldir/libwebp-$webpver-$cfg.zip"^
		 "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/libwebp-$webpver-$cfg.zip"
		tar -xf "$tooldir/libwebp-$webpver-$cfg.zip" -C "$tooldir"
	fi
fi

function convertpath() {
	icondir="$wd/$1/*.png"
	for f in $(ls $icondir)
	do
		echo "$1/$(basename "$f")"
		$cwebp -mt -q 80 -alpha_filter best -m 6 -af -hint picture -short "$f" -o "${f:0:-4}.webp"
	done
}

convertpath "chakram"
convertpath "delta"
convertpath "junior"
convertpath "oxygen"
convertpath "senary"
convertpath "senary2"
convertpath "tulliana"
convertpath "ubuntu"
convertpath "whistlepuff"
convertpath "xrabbit"
