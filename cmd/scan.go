package cmd

import (
	"github.com/schwarzlichtbezirk/hms/config"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:     "scan",
	Aliases: []string{"cache"},
	Short:   "Scan shared folders and cache thumbnails and tiles for founded images",
	Long:    `Prepare list of unique shared folders in all profiles. Then scan each shared folder and puts to cache thumbnails and tiles for founded images. Cache to database files embedded tags to make access faster.`,
	Example: config.AppName + " scan",
	RunE: func(cmd *cobra.Command, args []string) error {
		Init()
		RunCacher()
		Done()
		return nil
	},
}

var (
	IncludePath []string
	ExcludePath []string
)

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringSliceVarP(&IncludePath, "include", "i", nil, "Cache thumbnails and tiles at given paths in addition to shared paths")
	scanCmd.Flags().StringSliceVarP(&ExcludePath, "exclude", "e", nil, "Paths to exclude from scanning")
}
