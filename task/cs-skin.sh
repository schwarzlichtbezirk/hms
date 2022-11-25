#!/bin/bash -u

# define the working directory
wd=$(realpath -s "$(dirname "$0")/../frontend/skin")

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

# check up closure-stylesheets existence
cs="$tooldir/closure-stylesheets.jar"
if [ ! -f "$cs" ]; then
	echo "closure-stylesheets does not found, downloading it into '$tooldir' folder."
	if [ hash wget 2>/dev/null ]; then
		wget --no-clobber --no-check-certificate -O $cs\
		 https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar
	else
		curl -L --output $cs\
		 https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar
	fi
fi

compileskin() {
	java -jar $cs\
	 $wd/$1/page.css\
	 $wd/$1/card.css\
	 $wd/$1/icon.css\
	 $wd/$1/iconmenu.css\
	 $wd/$1/fileitem.css\
	 $wd/$1/imgitem.css\
	 $wd/$1/listitem.css\
	 $wd/$1/map.css\
	 $wd/$1/mp3player.css\
	 $wd/$1/slider.css\
	 -o $wd/$1.css
}

compileskin "daylight"
compileskin "light"
compileskin "blue"
compileskin "dark"
compileskin "neon"
compileskin "cup-of-coffee"
compileskin "coffee-beans"
compileskin "old-monitor"
