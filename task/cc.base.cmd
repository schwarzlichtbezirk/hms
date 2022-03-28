@echo off
cd /d %~dp0../frontend

java -jar %~d0/tools/closure-compiler-v20210907.jar^
 --js plugin/leaflet.js^
 --js plugin/leaflet.markercluster.js^
 --js plugin/sha256.min.js^
 --strict_mode_input^
 --js_output_file build/app.bundle.js
