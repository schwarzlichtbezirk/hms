#!/bin/bash
cd $(dirname $0)/../frontend

cs=~/tools/closure-stylesheets.jar
if [ ! -f "$cs" ]; then
	echo "closure-stylesheets does not found, downloading it into '~/tools' folder."
	mkdir -pv ~/tools
	curl https://github.com/google/closure-stylesheets/releases/download/v1.5.0/closure-stylesheets.jar --output $cs
fi

java -jar $cs\
 skin/daylight/page.css\
 skin/daylight/card.css\
 skin/daylight/iconmenu.css\
 skin/daylight/fileitem.css\
 skin/daylight/imgitem.css\
 skin/daylight/listitem.css\
 skin/daylight/map.css\
 skin/daylight/mp3player.css\
 skin/daylight/slider.css\
 -o skin/daylight.css

java -jar $cs\
 skin/blue/page.css\
 skin/blue/card.css\
 skin/blue/iconmenu.css\
 skin/blue/fileitem.css\
 skin/blue/imgitem.css\
 skin/blue/listitem.css\
 skin/blue/map.css\
 skin/blue/mp3player.css\
 skin/blue/slider.css\
 -o skin/blue.css

java -jar $cs\
 skin/dark/page.css\
 skin/dark/card.css\
 skin/dark/iconmenu.css\
 skin/dark/fileitem.css\
 skin/dark/imgitem.css\
 skin/dark/listitem.css\
 skin/dark/map.css\
 skin/dark/mp3player.css\
 skin/dark/slider.css\
 -o skin/dark.css

java -jar $cs\
 skin/neon/page.css\
 skin/neon/card.css\
 skin/neon/iconmenu.css\
 skin/neon/fileitem.css\
 skin/neon/imgitem.css\
 skin/neon/listitem.css\
 skin/neon/map.css\
 skin/neon/mp3player.css\
 skin/neon/slider.css\
 -o skin/neon.css

java -jar $cs\
 skin/cup-of-coffee/page.css\
 skin/cup-of-coffee/card.css\
 skin/cup-of-coffee/iconmenu.css\
 skin/cup-of-coffee/fileitem.css\
 skin/cup-of-coffee/imgitem.css\
 skin/cup-of-coffee/listitem.css\
 skin/cup-of-coffee/map.css\
 skin/cup-of-coffee/mp3player.css\
 skin/cup-of-coffee/slider.css\
 -o skin/cup-of-coffee.css

java -jar $cs\
 skin/coffee-beans/page.css\
 skin/coffee-beans/card.css\
 skin/coffee-beans/iconmenu.css\
 skin/coffee-beans/fileitem.css\
 skin/coffee-beans/imgitem.css\
 skin/coffee-beans/listitem.css\
 skin/coffee-beans/map.css\
 skin/coffee-beans/mp3player.css\
 skin/coffee-beans/slider.css\
 -o skin/coffee-beans.css

java -jar $cs\
 skin/old-monitor/page.css\
 skin/old-monitor/card.css\
 skin/old-monitor/iconmenu.css\
 skin/old-monitor/fileitem.css\
 skin/old-monitor/imgitem.css\
 skin/old-monitor/listitem.css\
 skin/old-monitor/map.css\
 skin/old-monitor/mp3player.css\
 skin/old-monitor/slider.css\
 -o skin/old-monitor.css
