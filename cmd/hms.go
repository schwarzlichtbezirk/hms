package main

import (
	"github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.7.7"
const builddate = "2021.12.04"

var log = hms.Log

func main() {
	log.Printf("version: %s, builton: %s\n", buildvers, builddate)
	hms.MakeServerLabel("hms", buildvers)
	hms.Init()
	var gmux = hms.NewRouter()
	hms.RegisterRoutes(gmux)
	hms.Run(gmux)
	log.Println("hint: Open http://localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use http://localhost/stat for server state monitoring.")
	hms.WaitExit()
	hms.Shutdown()
}

// The End.
