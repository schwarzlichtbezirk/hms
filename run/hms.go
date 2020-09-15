package main

import (
	"github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.4.8"
const builddate = "2020.09.14"

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
	log.Printf("hint: Open localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown.")
	hms.WaitBreak()
	log.Println("shutting down")
	hms.Done()
	log.Println("server stopped")
}

// The End.
