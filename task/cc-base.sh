#!/bin/bash -u

# define the working directory
wd=$(realpath -s "$(dirname "$0")/../frontend")

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

# check up closure-compiler existence
cv=v20220202 # see https://mvnrepository.com/artifact/com.google.javascript/closure-compiler
cc="$tooldir/closure-compiler-$cv.jar"
if [ ! -f "$cc" ]; then
	echo "closure-compiler does not found, downloading it into '$tooldir' folder."
	curl -L --output $cc\
	 https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/$cv/closure-compiler-$cv.jar
fi

java -jar $cc\
 --js $wd/plugin/leaflet.js\
 --js $wd/plugin/leaflet.markercluster.js\
 --js $wd/plugin/sha256.min.js\
 --strict_mode_input\
 --js_output_file $wd/build/app.bundle.js
