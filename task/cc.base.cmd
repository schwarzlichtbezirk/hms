@echo off
call :realpath %~dp0..\frontend
set wd=%retval%

set cv=v20220202
set cc=%~d0\tools\closure-compiler-%cv%.jar
if not exist %cc% (
	echo closure-compiler does not found, downloading it into '\tools' folder.
	mkdir %~d0\tools
	curl https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/%cv%/closure-compiler-%cv%.jar --output %cc%
)

java -jar %cc%^
 --js %wd%\plugin\leaflet.js^
 --js %wd%\plugin\leaflet.markercluster.js^
 --js %wd%\plugin\sha256.min.js^
 --strict_mode_input^
 --js_output_file %wd%\build\app.bundle.js

goto :eof

:realpath
set retval=%~f1
goto :eof
