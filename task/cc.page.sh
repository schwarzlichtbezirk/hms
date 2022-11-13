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
 --js devmode/relmode.js\
 --js devmode/common.js\
 --js devmode/request.js\
 --js devmode/fileitem.js\
 --js devmode/cards.js\
 --js devmode/mp3player.js\
 --js devmode/slider.js\
 --js devmode/mainpage.js\
 --strict_mode_input\
 --js_output_file build/main.bundle.js\
 --create_source_map build/main.bundle.js.map

java -jar $cc\
 --js devmode/relmode.js\
 --js devmode/common.js\
 --js devmode/request.js\
 --js devmode/statpage.js\
 --strict_mode_input\
 --js_output_file build/stat.bundle.js\
 --create_source_map build/stat.bundle.js.map
