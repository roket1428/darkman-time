// Package darkman implements darkman's service itself.
//
// This package is used by gitlab.com/WhyNotHugo/darkman/cmd, which is the cli
// that wraps around the service and the client.
package darkman

import (
	"fmt"
	"log"
	"os"

	"github.com/adrg/xdg"
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

func saveModeToCache(mode Mode) {
	cacheFilePath, err := xdg.CacheFile("darkman/mode.txt")
	if err != nil {
		fmt.Println("Failed find location for mode cache file:", err)
		return
	}

	err = os.WriteFile(cacheFilePath, []byte(mode), os.FileMode(0600))
	if err != nil {
		fmt.Println("Failed to save mode to cache:", err)
		return
	}
}

func readModeFromCache() (Mode, error) {
	cacheFilePath, err := xdg.CacheFile("darkman/location.txt")
	if err != nil {
		return NULL, fmt.Errorf("error determining cache file path: %v", err)
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return NULL, fmt.Errorf("error reading cache file path: %v", err)
	}

	var tmp interface{} = string(data[:])
	if tmp == DARK || tmp == LIGHT {
		return tmp.(Mode), nil
	} else {
		return NULL, nil

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
	service.AddListener(saveModeToCache)

	if config.DBusServer {
		log.Println("Running with D-Bus server.")
		_, dbusCallback := NewDbusServer(service.ChangeMode)
		service.AddListener(dbusCallback)
	} else {
		log.Println("Running without D-Bus server.")
	}

	if config.Portal {
		log.Println("Running with XDG portal.")
		_, portalCallback := NewPortal()
		service.AddListener(portalCallback)
	} else {
		log.Println("Running without XDG portal.")
	}

	if initialLocation != nil || config.UseGeoclue {
		// Start after registering all callbacks, so that the first changes
		// are triggered after they're all listening.
		err = NewScheduler(initialLocation, service.ChangeMode, config.UseGeoclue)
		if err != nil {
			log.Panicln("Failed to initialise the scheduler:", err)
		}
	} else {
		log.Println("Not using geoclue and no configured location.")
		log.Println("No automatic transitions will be scheduled.")

		mode, err := readModeFromCache()
		if err != nil {
			log.Println("Couldn't load previous mode from cache:", err)
			mode = NULL
		}
		service.ChangeMode(mode)
	}

	// Sleep silently forever...
	select {}
}
