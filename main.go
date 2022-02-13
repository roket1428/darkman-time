// Package darkman implements darkman's service itself.
//
// This package is used by gitlab.com/WhyNotHugo/darkman/cmd, which is the cli
// that wraps around the service and the client.
package darkman

import (
	"log"
)

type Mode string
type Service struct {
	currentMode Mode
	listeners   *[]func(Mode)
}

const (
	NULL  Mode = "null" // Only used while still initialising.
	LIGHT Mode = "light"
	DARK  Mode = "dark"
)

/// Add a callback to be run each time the current mode changes.
func (service *Service) AddListener(listener func(Mode)) {
	*service.listeners = append(*service.listeners, listener)
}

/// Change the current mode (and run all callbacks).
func (service *Service) ChangeMode(mode Mode) {
	log.Printf("Mode should now be: %v mode.\n", mode)
	if mode == service.currentMode {
		log.Println("No transition necessary")
		return
	}

	service.currentMode = mode
	for _, listener := range *service.listeners {
		go listener(mode)
	}
}

/// Run the darkman service.
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

	service := Service{
		currentMode: NULL,
		listeners:   &[]func(Mode){},
	}
	service.AddListener(RunScripts)

	_ = NewScheduler(initialLocation, service.ChangeMode)

	if config.DBusServer {
		log.Println("Running with D-Bus server.")
		_, dbusCallback := NewDbusServer()
		service.AddListener(dbusCallback)
	} else {
		log.Println("Running without D-Bus server.")
	}

	// Sleep silently forever...
	select {}
}
