
Home Media Server. Browse files on your computer as in explorer, listen music with folder as playlist, view photos and markers of them on map by theirs geotags. Share some file or folder to get access from internet.

Screenshots:
[![hms #1](http://images.sevstar.net/images/86980114770981357724_thumb.png)](http://images.sevstar.net/images/86980114770981357724.jpg)
[![hms #1](http://images.sevstar.net/images/08282078015756047629_thumb.png)](http://images.sevstar.net/images/08282078015756047629.jpg)

# How to build.

At first, install [Golang](https://golang.org/) minimum 1.13 version, and run those commands in command prompt:

```batch
go get github.com/schwarzlichtbezirk/hms
go build -o %GOPATH%\bin\wpkbuild.exe -v github.com/schwarzlichtbezirk/wpk/build
xcopy %GOPATH%\src\github.com\schwarzlichtbezirk\hms\conf %GOPATH%\bin\hms /f /d /i /s /e /k /y
%GOPATH%/bin/wpkbuild.exe %GOPATH%/src/github.com/schwarzlichtbezirk/hms/frontend/tool/pack.lua
go build -o %GOPATH%\bin\hms.exe -v github.com/schwarzlichtbezirk/hms/run
```
First command extracts program package with all dependencies. Second makes wpk-build utilite. Third copies/updates initial settings files from source folder to destination binary folder. 4th produces full resources package. 5th makes server executable file.

If you want some other shorter package, you can replace Lua-script name in 4th command.

# Packages variations.

By default script `pack.lua` produces full package with all icons collections with both `png` and `webp` formats. `webp` icons are more than 5x times shorter than `png` and this format is supported by Android, Chrome, Opera, Edge, Firefox, but in some cases it's can be needed `png` yet.

To make full package with `webp` icons only, use `hms-all.lua` script:
```batch
%GOPATH%/bin/wpkbuild.exe %GOPATH%/src/github.com/schwarzlichtbezirk/hms/frontend/tool/hms-all.lua
```
To make package with minimal size, use `hms-tiny.lua` script. Script `hms-free.lua` produces package with icons, which have allowed commercial usage by their license.
