package cmd

import (
	"context"
	"fmt"

	"github.com/schwarzlichtbezirk/hms/config"
	"github.com/spf13/cobra"
)

const scanShort = "Scan shared folders and cache thumbnails and tiles for founded images"
const scanLong = `Prepare list of unique shared folders in all profiles. Then scan each shared folder and puts to cache thumbnails and tiles for founded images. Cache to database files embedded tags to make access faster.`
const scanExmp = `Start scanning with all shares at profiles:
  %s scan`

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:     "scan",
	Aliases: []string{"cache"},
	Short:   scanShort,
	Long:    scanLong,
	Example: fmt.Sprintf(scanExmp, config.AppName),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var exitctx context.Context
		if exitctx, err = Init(); err != nil {
			return
		}
		RunCacher(exitctx)
		err = Done()
		return
	},
}

var (
	IncludePath []string
	ExcludePath []string
)

func init() {
	rootCmd.AddCommand(scanCmd)

	var flags = scanCmd.Flags()
	flags.StringSliceVarP(&IncludePath, "include", "i", nil, "cache thumbnails and tiles at given paths in addition to shared paths")
	flags.StringSliceVarP(&ExcludePath, "exclude", "e", nil, "paths to exclude from scanning")
}
