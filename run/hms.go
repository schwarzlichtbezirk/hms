package main

import (
	. "github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.6.0"
const builddate = "2020.11.10"

func main() {
	Log.Printf("version: %s, builton: %s", buildvers, builddate)
	MakeServerLabel("hms", buildvers)
	Log.Println("starts")
	Init()
	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)
	Log.Printf("ready")
	Log.Printf("hint: Open localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown.")
	WaitBreak()
	Log.Println("shutting down")
	Done()
	Log.Println("server stopped")
}

// The End.
