@echo off
cd /d %GOPATH%/src/github.com/schwarzlichtbezirk/hms/frontend

java -jar %~d0/tools/closure-compiler.jar^
 --js devmode/relmode.js^
 --js devmode/common.js^
 --js devmode/request.js^
 --js devmode/fileitem.js^
 --js devmode/cards.js^
 --js devmode/mp3player.js^
 --js devmode/slider.js^
 --js devmode/mainpage.js^
 --strict_mode_input^
 --js_output_file build/main.bundle.js^
 --create_source_map build/main.bundle.js.map

java -jar %~d0/tools/closure-compiler.jar^
 --js devmode/relmode.js^
 --js devmode/common.js^
 --js devmode/request.js^
 --js devmode/statpage.js^
 --strict_mode_input^
 --js_output_file build/stat.bundle.js^
 --create_source_map build/stat.bundle.js.map
