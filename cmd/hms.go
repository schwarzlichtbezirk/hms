package main

import (
	"github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.7.5"
const builddate = "2021.07.04"

var log = hms.Log

func main() {
	log.Printf("version: %s, builton: %s", buildvers, builddate)
	hms.MakeServerLabel("hms", buildvers)
	log.Println("starts")
	hms.Init()
	var gmux = hms.NewRouter()
	hms.RegisterRoutes(gmux)
	hms.Run(gmux)
	log.Printf("ready")
	log.Printf("hint: Open localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use localhost/stat for server state monitoring.")
	go func() {
		hms.WaitBreak()
		log.Println("shutting down by break begin")
	}()
	hms.WaitExit()
	hms.Shutdown()
	log.Println("shutting down complete.")
}

// The End.
