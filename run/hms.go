package main

import (
	. "github.com/schwarzlichtbezirk/hms"
)

const buildvers = "0.6.6"
const builddate = "2020.11.22"

func main() {
	Log.Printf("version: %s, builton: %s", buildvers, builddate)
	MakeServerLabel("hms", buildvers)
	Log.Println("starts")
	Init()
	var gmux = NewRouter()
	RegisterRoutes(gmux)
	Run(gmux)
	Log.Printf("ready")
	Log.Printf("hint: Open localhost page in browser to view the player. If you want to stop the server, press 'Ctrl+C' for graceful network shutdown. Use localhost/stat for server state monitoring.")
	WaitBreak()
	Log.Println("shutting down begin")
	Done()
	Log.Println("shutting down complete.")
}

// The End.
