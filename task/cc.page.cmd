@echo off
cd /d %~dp0../frontend

set cv=v20220202
set cc=%~d0/tools/closure-compiler-%cv%.jar
if not exist %cc% (
	echo closure-compiler does not found, downloading it into '\tools' folder.
	mkdir %~d0\tools
	curl https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/%cv%/closure-compiler-%cv%.jar --output %cc%
)

java -jar %cc%^
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

java -jar %cc%^
 --js devmode/relmode.js^
 --js devmode/common.js^
 --js devmode/request.js^
 --js devmode/statpage.js^
 --strict_mode_input^
 --js_output_file build/stat.bundle.js^
 --create_source_map build/stat.bundle.js.map
