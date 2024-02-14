@echo off
rem This script performs js-files minification for all pages.
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
 --js %wd%\devmode\relmode.js^
 --js %wd%\devmode\common.js^
 --js %wd%\devmode\request.js^
 --js %wd%\devmode\fileitem.js^
 --js %wd%\devmode\cards.js^
 --js %wd%\devmode\mp3player.js^
 --js %wd%\devmode\slider.js^
 --js %wd%\devmode\mainpage.js^
 --strict_mode_input^
 --js_output_file %wd%\build\main.bundle.js^
 --create_source_map %wd%\build\main.bundle.js.map

java -jar %cc%^
 --js %wd%\devmode\relmode.js^
 --js %wd%\devmode\common.js^
 --js %wd%\devmode\request.js^
 --js %wd%\devmode\statpage.js^
 --strict_mode_input^
 --js_output_file %wd%\build\stat.bundle.js^
 --create_source_map %wd%\build\stat.bundle.js.map

exit /b 0

:realpath
set retval=%~f1
exit /b
