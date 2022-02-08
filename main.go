// Package darkman implements darkman's service itself.
//
// This package is used by gitlab.com/WhyNotHugo/darkman/cmd, which is the cli
// that wraps around the service and the client.
package darkman

import (
	"log"

	"gitlab.com/WhyNotHugo/darkman/boottimer"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
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

	scheduler := NewScheduler()
	locations := make(chan geoclue.Location, 3)
	scheduler.AddListener(RunScriptsListener())

	if config.DBusServer {
		log.Println("Running with D-Bus server.")
		NewDbusServer(scheduler)
	} else {
		log.Println("Running without D-Bus server.")
	}

	// Listen for location changes and pass them to the handler.
	go func() {
		for {
			newLocation := <-locations
			log.Println("Location service has yielded:", newLocation)
			scheduler.UpdateLocation(newLocation)
		}
	}()

	// Alarms wake us up when it's time for the next transition.
	go func() {
		for {
			<-boottimer.Alarms
			scheduler.Tick()
		}
	}()

	err = GetLocations(initialLocation, locations)
	if err != nil {
		log.Println("Could not start location service:", err)
	}

	// Sleep silently forever...
	select {}
}
