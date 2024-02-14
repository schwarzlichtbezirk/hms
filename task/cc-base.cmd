@echo off
rem This script performs js-files minification for some plugins.
rem Downloads closure-compiler tool if it necessary.

call :realpath %~dp0..\frontend
set wd=%retval%

set cv=v20220202
set cc=%~d0\devtools\closure-compiler-%cv%.jar
if not exist %cc% (
	echo closure-compiler does not found, downloading it into '\devtools' folder.
	mkdir %~d0\devtools
	curl -L --output %cc%^
	 https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/%cv%/closure-compiler-%cv%.jar
)

"%JAVA_HOME%\bin\java.exe" -jar %cc%^
 --js %wd%\plugin\leaflet.js^
 --js %wd%\plugin\leaflet.markercluster.js^
 --js %wd%\plugin\sha256.min.js^
 --strict_mode_input^
 --js_output_file %wd%\build\app.bundle.js

exit /b 0

:realpath
set retval=%~f1
exit /b
