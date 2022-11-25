@echo off

rem define the task directory
set taskdir=%~dp0


echo.
echo STAGE#1: make WPK-builder
call %taskdir%\make-builder.cmd


echo.
echo STAGE#2: download Google closures tools

rem check up java runtime existence
java -version >nul 2>&1 || (
	echo its expected that's Java runtime should be present, it can be installed from this page:
	echo     https://www.java.com/en/download/manual.jsp
	exit /b 1
)

rem check up closure-stylesheets existence
set cs=%~d0\devtools\closure-stylesheets.jar
if not exist %cs% (
	echo closure-stylesheets does not found, downloading it into '\devtools' folder.
	mkdir %~d0\devtools
	curl -L --output %cs%^
	 https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar
)

rem check up closure-compiler existence
set cv=v20220202
set cc=%~d0\devtools\closure-compiler-%cv%.jar
if not exist %cc% (
	echo closure-compiler does not found, downloading it into '\devtools' folder.
	mkdir %~d0\devtools
	curl -L --output %cc%^
	 https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/%cv%/closure-compiler-%cv%.jar
)


echo.
echo STAGE#3: download fonts and plugins
call %taskdir%\deploy-fonts.cmd
call %taskdir%\deploy-plugins.cmd


echo.
echo STAGE#4: compile CSS-scripts to bundles
call %taskdir%\cs-skin.cmd


echo.
echo STAGE#5: compile basic JS-scripts plugins to bundle
call %taskdir%\cc-base.cmd 2>nul


echo.
echo STAGE#6: compile pages JS-scripts to bundles
call %taskdir%\cc-page.cmd


echo.
echo STAGE#7: build application WPK-file
call %taskdir%\wpk-app.cmd


echo.
echo STAGE#8: build full resources WPK-file
call %taskdir%\wpk-full.cmd


echo.
echo all stages are done.
