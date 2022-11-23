@echo off
call :realpath %~dp0..\frontend\skin
set wd=%retval%

set cs=%~d0\tools\closure-stylesheets.jar
if not exist %cs% (
	echo closure-stylesheets does not found, downloading it into '\tools' folder.
	mkdir %~d0\tools
	curl https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar --output %cs%
)

call :compileskin "daylight"
call :compileskin "light"
call :compileskin "blue"
call :compileskin "dark"
call :compileskin "neon"
call :compileskin "cup-of-coffee"
call :compileskin "coffee-beans"
call :compileskin "old-monitor"

goto :eof

:realpath
set retval=%~f1
goto :eof

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
goto :eof
