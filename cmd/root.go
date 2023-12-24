package cmd

import (
	"fmt"

	"github.com/schwarzlichtbezirk/hms/config"
	"github.com/spf13/cobra"
)

const rootShort = "Home Media Server. Play media files and view images, photos with theirs markers on the map."
const rootLong = `Browse files on your computer as in explorer, listen music with folder as playlist, view photos and markers of them on map by theirs geotags. Share some file or folder to get access from internet.
Music: plays MP3, OGG and others formats supported by browser. Video: display browser native supported formats, MP4 in all cases. Images: displays JPEG, PNG, GIF and others formats supported by browser. Also displays Adobe Photoshop PSD, TIFF, DDS, TGA images by converting to JPEG or PNG at server layer for browser representation. If any image have EXIF properties with geotags it will be placed at the map. Maps tiles provider can be changed, there is can be selected satellite view, streets view, topographic view, or hybrid. GPS-tracks in GPX file format also builds on map.
Files can be viewed by browsing file structure same as in Explorer. Disks ISO9660 images can be browsed same as file system folders. Also opens any popular playlist formats as the folder.`

var (
	rootCmd = &cobra.Command{
		Use:     config.AppName,
		Version: config.BuildVers,
		Short:   rootShort,
		Long:    rootLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("***")
			return nil
		},
	}
)

func init() {
	cobra.OnInitialize(config.InitConfig)

	var flags = rootCmd.PersistentFlags()
	flags.StringVarP(&config.CfgFile, "config", "c", "", "config file (default is config/slot.yaml at executable location), binded to CFGFILE variable")
	flags.StringVarP(&config.PkgPath, "package", "w", "", "resource packages path (default is executable location), binded to PKGPATH variable")
	flags.StringVarP(&config.TmbPath, "thumb", "t", "", "images cache path (default is 'cache' subdirectory at executable location), binded to TMBPATH variable")
	flags.BoolVarP(&config.DevMode, "devmode", "d", false, "start application in developer mode")
	rootCmd.SetVersionTemplate(fmt.Sprintf("version: %s, builton: %s", config.BuildVers, config.BuildTime))
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
