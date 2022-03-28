package main

import (
	"github.com/schwarzlichtbezirk/hms"
)

var log = hms.Log

func main() {
	hms.Init()
	var gmux = hms.NewRouter()
	hms.RegisterRoutes(gmux)
	hms.Run(gmux)
	log.Infoln("hint: Open http://localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use http://localhost/stat for server state monitoring.")
	hms.WaitExit()
	hms.Shutdown()
}

// The End.
