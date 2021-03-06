@echo off
cd /d %GOPATH%/src/github.com/schwarzlichtbezirk/hms/frontend

java -jar %~d0/tools/closure-compiler.jar^
 --js plugin/popper.min.js^
 --js plugin/bootstrap.min.js^
 --strict_mode_input^
 --js_output_file build/base.bundle.js^
 
java -jar %~d0/tools/closure-compiler.jar^
 --js plugin/leaflet.js^
 --js plugin/leaflet.markercluster.js^
 --js plugin/sha256.min.js^
 --strict_mode_input^
 --js_output_file build/app.bundle.js^
