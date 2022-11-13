#!/bin/bash
cd $(dirname $0)/../frontend

cv=v20220202 # see https://mvnrepository.com/artifact/com.google.javascript/closure-compiler
cc=~/tools/closure-compiler-$cv.jar
if [ ! -f "$cc" ]; then
	echo "closure-compiler does not found, downloading it into '~/tools' folder."
	mkdir -pv ~/tools
	curl https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/$cv/closure-compiler-$cv.jar --output $cc
fi

java -jar $cc\
 --js plugin/leaflet.js\
 --js plugin/leaflet.markercluster.js\
 --js plugin/sha256.min.js\
 --strict_mode_input\
 --js_output_file build/app.bundle.js
