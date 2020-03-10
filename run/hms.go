package main

import (
	"github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.2.4"
const builddate = "2020.03.11"

var log = hms.Log

func main() {
	log.Printf("version: %s, builton: %s", buildvers, builddate)
	log.Println("starts")
	hms.Init()
	var gmux = hms.NewRouter()
	hms.RegisterRoutes(gmux)
	hms.Run(gmux)
	log.Printf("ready")
	log.Printf("hint: Open localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown.")
	hms.WaitBreak()
	log.Println("shutting down")
	hms.Done()
	log.Println("server stopped")
}

// The End.
