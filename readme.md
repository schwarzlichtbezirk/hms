
Home Media Server. Browse files on your computer as in explorer, listen music with folder as playlist, view photos and markers of them on map by theirs geotags. Share some file or folder to get access from internet.

Music: plays MP3, OGG and others formats supported by browser. Images: displays JPEG, PNG, GIF and others formats supported by browser. Also displays Adobe Photoshop PSD, TIFF, DDS, TGA images by converting to JPEG or PNG at server layer for browser representation. If any image have EXIF properties with geotags it will be placed at the map. Maps tiles provider can be changed, there is can be selected satellite view, streets view, topographic view, or hybrid. GPS-tracks in GPX file format also builds on map.

Files can be viewed by browsing file structure same as in Explorer. Disks ISO9660 images can be browsed same as file system folders. Also opens any popular playlist formats as the folder.

Screenshots:

[![hms #1](http://images.sevstar.net/images/86980114770981357724_thumb.png)](http://images.sevstar.net/images/86980114770981357724.jpg)
[![hms #1](http://images.sevstar.net/images/08282078015756047629_thumb.png)](http://images.sevstar.net/images/08282078015756047629.jpg)

# Download

Compiled binaries can be downloaded in [Releases](https://github.com/schwarzlichtbezirk/hms/releases) section.

# How to build

At first, install [Golang](https://golang.org/) minimum 1.16 version, and run those commands in command prompt:

```batch
go get github.com/schwarzlichtbezirk/hms
go build -o %GOPATH%\bin\wpkbuild.exe -v github.com/schwarzlichtbezirk/wpk/build
xcopy %GOPATH%\src\github.com\schwarzlichtbezirk\hms\conf %GOPATH%\bin\hms /f /d /i /s /e /k /y
%GOPATH%/bin/wpkbuild.exe %GOPATH%/src/github.com/schwarzlichtbezirk/hms/frontend/tool/pack.lua
go build -o %GOPATH%\bin\hms.exe -v github.com/schwarzlichtbezirk/hms/run
```
First command extracts program package with all dependencies. Second makes wpk-build utilite. Third copies/updates initial settings files from source folder to destination binary folder. 4th produces full resources package. 5th makes server executable file.

If you want some other shorter package, you can replace Lua-script name in 4th command.

# Packages variations

By default script `pack.lua` produces full package with all icons collections with both `png` and `webp` formats. `webp` icons are more than 5x times shorter than `png` and this format is supported by Android, Chrome, Opera, Edge, Firefox, but in some cases it's can be needed `png` yet.

To make full package with `webp` icons only, use `hms-all.lua` script:
```batch
%GOPATH%/bin/wpkbuild.exe %GOPATH%/src/github.com/schwarzlichtbezirk/hms/frontend/tool/hms-all.lua
```
To make package with minimal size, use `hms-tiny.lua` script. Script `hms-free.lua` produces package with icons, which have allowed commercial usage by their license.

# Configuration

Before server start you can configure some options. Any configuration files lays at "hms" folder, and have `yaml` file format.

If you're going to share resources, first of all you can open `settings.yaml` file and change `access-key` and `refresh-key` for tokens protection. This is main server settings file, and it does not modified by program. Then you can open `profiles.yaml` file and change default admin password to anything other. Changing authentication passwords and profile passwords - that's all modifications to provide basic protection access to server.

# Authorization

Server provides ability to make profiles each of which can have own set of drives, own set of shared resources, own templates for excluded files. User can be authorized for profile by its login+password. Unauthorized users can have access to shared by profile resources and have no ability for any modifications. List of profiles can be found in file `profiles.yaml`.

If page is opened on localhost, there is no need for authorization. On localhost you have silent authorized access to default profile. ID of default profile is `1` and can be modified at field `default-profile-id` of settings.

# Home page and sharing

Open home page in browser, `localhost` if it running on local computer, and there will be list of categories. Home page can be opened from any other location by "home" button at top left.

`Drives list` category contains list that can be modified by adding or removing some destinations each of which will be seems as a drive, "plus" and "minus" buttons on top right of folder slide serves for it. On first server start under the Windows, it will be scanned all available disks by their letters and added to list.

`Shared resources` contans list of all shared folders and list of all shared files on active profile. All those resources have can be accessed by anyone in internet without authorization. If folder is shared, all nested subfolders are also shared. Categories also can be shared. Shared categories will be seen for anyone at home page. So, if `Shared resources` category is shared, anyone will see list of all shared resources. If `Drives list` is shared, anyone will get access to any drive in list.

Other categories at home page contains list of folders with files of those category. Folder considered to some category if it has more then 50% files of this category. For example, if folder have 10 mp3 files and 2 jpg files, it will be in `Music and audio files`. Folders discovers, if they were opened at once.

If some category of files is shared, then anyone will see this category at home page, and have access to files of this category grouped into their folders. In this case there is no access to nested folders for anyone and no access to files of not shared categories in those folders. For example, if `Music and audio files` is shared, then anyone will see 10 mp3 files in some music folder, and does not see 2 jpg files in that folder.

(c) schwarzlichtbezirk, 2021.
