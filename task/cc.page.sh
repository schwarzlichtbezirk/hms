#!/bin/bash
cd $(dirname $0)/../frontend

java -jar ~/tools/closure-compiler-v20210907.jar\
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

java -jar ~/tools/closure-compiler-v20210907.jar\
 --js devmode/relmode.js\
 --js devmode/common.js\
 --js devmode/request.js\
 --js devmode/statpage.js\
 --strict_mode_input\
 --js_output_file build/stat.bundle.js\
 --create_source_map build/stat.bundle.js.map