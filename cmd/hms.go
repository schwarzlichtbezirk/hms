package main

import (
	"github.com/schwarzlichtbezirk/hms"
)

func main() {
	hms.Init()
	var gmux = hms.NewRouter()
	hms.RegisterRoutes(gmux)
	hms.Run(gmux)
	hms.WaitExit()
	hms.Shutdown()
}

// The End.
