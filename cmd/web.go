package cmd

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	cfg "github.com/schwarzlichtbezirk/hms/config"
	srv "github.com/schwarzlichtbezirk/hms/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const webShort = "Starts web-server"
const webLong = ``
const webExmp = `Start web-server without settings overwriting:
  %s web
Start web-server with 3 opened ports for non-encrypted connections:
  %s web -p=:80 -p=:8088 -p=:8888`

// webCmd represents the web command
var webCmd = &cobra.Command{
	Use:     "web",
	Aliases: []string{"srv"},
	Short:   webShort,
	Long:    webLong,
	Example: fmt.Sprintf(webExmp, cfg.AppName, cfg.AppName),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if cfg.DevMode {
			gin.SetMode(gin.DebugMode)
		} else {
			gin.SetMode(gin.ReleaseMode)
		}

		var exitctx context.Context
		if exitctx, err = Init(); err != nil {
			return
		}

		var r = gin.New()
		r.SetTrustedProxies(Cfg.TrustedProxies)
		srv.Router(r)

		RunWeb(exitctx, r)
		srv.WaitHandlers()
		err = Done()
		return
	},
}

func init() {
	rootCmd.AddCommand(webCmd)

	var flags = webCmd.Flags()
	flags.StringSliceP("port", "p", []string{":80"}, "List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains. Binded to PORTHTTP variable.")
	viper.BindPFlag("web-server.port-http", flags.Lookup("port"))
	viper.BindEnv("web-server.port-http", "PORTHTTP")
	flags.StringSliceP("tls", "s", nil, "List of address:port values for encrypted connections. Address is skipped in most common cases, port only remains. Binded to PORTTLS variable.")
	viper.BindPFlag("web-server.port-tls", flags.Lookup("tls"))
	viper.BindEnv("web-server.port-tls", "PORTTLS")
	flags.StringP("xorm", "x", "sqlite3", "Provides XORM driver name. Binded to XORMDRIVER variable.")
	viper.BindPFlag("xorm.xorm-driver-name", flags.Lookup("xorm"))
	viper.BindEnv("xorm.xorm-driver-name", "XORMDRIVER")
}
