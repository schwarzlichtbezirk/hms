package main

import (
	"github.com/gorilla/mux"
	"github.com/schwarzlichtbezirk/hms"
)

func main() {
	Init()
	var gmux = mux.NewRouter()
	hms.RegisterRoutes(gmux)
	Run(gmux)
	WaitExit()
	Shutdown()
}

// The End.
