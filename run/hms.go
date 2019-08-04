package main

import (
	"flag"
	"github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.1.1"
const builddate = "2019.08.04"

var log = hms.Log

func main() {
	flag.BoolVar(&hms.DevMode, "dev", false, "starts web-server in developer mode")
	flag.Parse()

	log.Printf("version: %s, date: %s", buildvers, builddate)
	if hms.DevMode {
		log.Println("starts in developer mode")
	} else {
		log.Println("starts")
	}
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
