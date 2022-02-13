// Package darkman implements darkman's service itself.
//
// This package is used by gitlab.com/WhyNotHugo/darkman/cmd, which is the cli
// that wraps around the service and the client.
package darkman

import (
	"log"
)

type Mode string

const (
	NULL  Mode = "null" // Only used while still initialising.
	LIGHT Mode = "light"
	DARK  Mode = "dark"
)

func ExecuteService() {
	log.SetFlags(log.Lshortfile)

	config, err := ReadConfig()
	if err != nil {
		log.Println("Could not read configuration file:", err)
	}

	initialLocation, err := config.GetLocation()
	if err != nil {
		log.Println("No location found via config.")
	} else {
		log.Println("Found location in config:", initialLocation)
	}

	scheduler := NewScheduler(initialLocation)
	scheduler.AddListener(RunScripts)

	if config.DBusServer {
		log.Println("Running with D-Bus server.")
		_, dbusCallback := NewDbusServer()
		scheduler.AddListener(dbusCallback)
	} else {
		log.Println("Running without D-Bus server.")
	}

	// Sleep silently forever...
	select {}
}
