#!/bin/bash -u
# This script perform all jobs for project deployment after git clone.

# define the task directory
taskdir="$(dirname "$0")"


echo
echo "STAGE#1: make WPK-builder"
source $taskdir/make-builder.sh


echo
echo "STAGE#2: download Google closures tools"

# find devtools directory
tmp=$taskdir
while [ "$tmp" != "/" ]; do
	if [ -d "$tmp/devtools" ]; then
		tooldir="$tmp/devtools"
		break
	fi
	tmp=$(realpath -s "$tmp/..")
done
unset tmp
if [ -z "$tooldir" ]; then
	tooldir="~/devtools"
	mkdir -pv "$tooldir"
fi
echo "development tools directory is '$tooldir'"

# check up java runtime existence
if [ ! hash java 2>/dev/null ]; then
	echo "its expected that's Java runtime should be present, it can be installed with:"
	echo "    sudo apt install default-jre"
	echo "    java -version"
	exit 1
fi

# check up closure-stylesheets existence
cs="$tooldir/closure-stylesheets.jar"
if [ ! -f "$cs" ]; then
	echo "closure-stylesheets does not found, downloading it."
	curl -L --output $cs\
	 https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar
fi

# check up closure-compiler existence
cv=v20220202 # see https://mvnrepository.com/artifact/com.google.javascript/closure-compiler
cc="$tooldir/closure-compiler-$cv.jar"
if [ ! -f "$cc" ]; then
	echo "closure-compiler does not found, downloading it."
	curl -L --output $cc\
	 https://repo1.maven.org/maven2/com/google/javascript/closure-compiler/$cv/closure-compiler-$cv.jar
fi


echo
echo "STAGE#3: download fonts and plugins"
source $taskdir/deploy-fonts.sh
source $taskdir/deploy-plugins.sh


echo
echo "STAGE#4: compile CSS-scripts to bundles"
source $taskdir/cs-skin.sh


echo
echo "STAGE#5: compile basic JS-scripts plugins to bundle"
source $taskdir/cc-base.sh &>/dev/null


echo
echo "STAGE#6: compile pages JS-scripts to bundles"
source $taskdir/cc-page.sh


echo
echo "STAGE#7: build application WPK-file"
source $taskdir/wpk-app.sh


echo
echo "STAGE#8: build full resources WPK-file"
source $taskdir/wpk-full.sh


echo
echo "all stages are done."
