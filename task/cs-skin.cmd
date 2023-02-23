@echo off
rem This script performs CSS-files minification for each skin.
rem Downloads closure-stylesheets tool if it necessary.

call :realpath %~dp0..\frontend\skin
set wd=%retval%

set cs=%~d0\devtools\closure-stylesheets.jar
if not exist %cs% (
	echo closure-stylesheets does not found, downloading it into '\devtools' folder.
	mkdir %~d0\devtools
	curl -L --output %cs%^
	 https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar
)

call :compileskin "daylight"
call :compileskin "light"
call :compileskin "blue"
call :compileskin "dark"
call :compileskin "neon"
call :compileskin "cup-of-coffee"
call :compileskin "coffee-beans"
call :compileskin "matrix"

exit /b 0

:realpath
set retval=%~f1
exit /b

:compileskin
java -jar %cs%^
 %wd%\%~1\page.css^
 %wd%\%~1\card.css^
 %wd%\%~1\icon.css^
 %wd%\%~1\iconmenu.css^
 %wd%\%~1\fileitem.css^
 %wd%\%~1\imgitem.css^
 %wd%\%~1\listitem.css^
 %wd%\%~1\map.css^
 %wd%\%~1\mp3player.css^
 %wd%\%~1\slider.css^
 -o %wd%\%~1.css
exit /b
