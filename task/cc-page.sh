#!/bin/bash -u

# define the working directory
wd=$(realpath -s "$(dirname "$0")/../frontend")

# find tools directory
tmp=$wd
while [ "$tmp" != "/" ]; do
	if [ -d "$tmp/tools" ]; then
		tooldir="$tmp/tools"
		break
	fi
	tmp=$(realpath -s "$tmp/..")
done
unset tmp
if [ -z "$tooldir" ]; then
	tooldir="~/tools"
	mkdir -pv "$tooldir"
fi

# check up closure-compiler existence
cv=v20220202 # see https://mvnrepository.com/artifact/com.google.javascript/closure-compiler
cc="$tooldir/closure-compiler-$cv.jar"
if [ ! -f "$cc" ]; then
	echo "closure-compiler does not found, downloading it into '$tooldir' folder."
	curl -L --output $cc\
	 https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/$cv/closure-compiler-$cv.jar
fi

java -jar $cc\
 --js $wd/devmode/relmode.js\
 --js $wd/devmode/common.js\
 --js $wd/devmode/request.js\
 --js $wd/devmode/fileitem.js\
 --js $wd/devmode/cards.js\
 --js $wd/devmode/mp3player.js\
 --js $wd/devmode/slider.js\
 --js $wd/devmode/mainpage.js\
 --strict_mode_input\
 --js_output_file $wd/build/main.bundle.js\
 --create_source_map $wd/build/main.bundle.js.map

java -jar $cc\
 --js $wd/devmode/relmode.js\
 --js $wd/devmode/common.js\
 --js $wd/devmode/request.js\
 --js $wd/devmode/statpage.js\
 --strict_mode_input\
 --js_output_file $wd/build/stat.bundle.js\
 --create_source_map $wd/build/stat.bundle.js.map
