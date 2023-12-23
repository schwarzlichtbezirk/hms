package cmd

import (
	"github.com/gorilla/mux"
	"github.com/schwarzlichtbezirk/hms/config"
	srv "github.com/schwarzlichtbezirk/hms/server"
	"github.com/spf13/cobra"
)

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:     "web",
	Aliases: []string{"srv"},
	Short:   "Starts web-server.",
	Long:    ``,
	Example: config.AppName + " web",
	RunE: func(cmd *cobra.Command, args []string) error {
		Init()
		var gmux = mux.NewRouter()
		srv.RegisterRoutes(gmux)
		RunWeb(gmux)
		WaitExit()
		srv.WaitHandlers()
		Done()
		return nil
	},
}

var (
	fSlot bool
)

func init() {
	rootCmd.AddCommand(webCmd)

	webCmd.Flags().BoolVarP(&fSlot, "slot", "s", false, "'Slotopol' Megajack 5x3 slots")
}
