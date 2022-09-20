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
 --js plugin/leaflet.js^
 --js plugin/leaflet.markercluster.js^
 --js plugin/sha256.min.js^
 --strict_mode_input^
 --js_output_file build/app.bundle.js
