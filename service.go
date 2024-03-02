// Package darkman implements darkman's service itself.
//
// This package is used by gitlab.com/WhyNotHugo/darkman/cmd, which is the cli
// that wraps around the service and the client.
package darkman

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/adrg/xdg"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type Mode string
type Service struct {
	currentMode Mode
	listeners   *[]func(Mode) error
}

const (
	NULL  Mode = "null" // Only used while still initialising.
	LIGHT Mode = "light"
	DARK  Mode = "dark"
)

// Creates a new Service instance.
func NewService(initialMode Mode) Service {
	return Service{
		currentMode: initialMode,
		listeners:   &[]func(Mode) error{},
	}
}

// Add a callback to be run each time the current mode changes.
func (service *Service) AddListener(listener func(Mode) error) {
	*service.listeners = append(*service.listeners, listener)
	// Apply once with the initial mode.
	if err := listener(service.currentMode); err != nil {
		fmt.Println("error applying initial mode:", err)
	}
}

// Change the current mode (and run all callbacks).
func (service *Service) ChangeMode(mode Mode) {
	log.Printf("Wanted mode is: %v mode.\n", mode)
	if mode == service.currentMode {
		log.Println("No transition necessary")
		return
	}

	log.Println("Notifying all transition handlers of new mode.")
	service.currentMode = mode
	for _, listener := range *service.listeners {
		go func(listener func(Mode) error, mode Mode) {
			if err := listener(mode); err != nil {
				fmt.Println("Error notifying listener:", err)
			}
		}(listener, mode)
	}
}

func saveModeToCache(mode Mode) error {
	cacheFilePath, err := xdg.CacheFile("darkman/mode.txt")
	if err != nil {
		return fmt.Errorf("failed determine location for mode cache file: %v", err)
	}

	if err = os.WriteFile(cacheFilePath, []byte(mode), os.FileMode(0600)); err != nil {
		return fmt.Errorf("failed to save mode to cache: %v", err)
	}
	return nil
}

func readModeFromCache() (Mode, error) {
	cacheFilePath, err := xdg.CacheFile("darkman/mode.txt")
	if err != nil {
		return NULL, fmt.Errorf("error determining cache file path: %v", err)
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return NULL, fmt.Errorf("error reading cache file path: %v", err)
	}

	var tmp interface{} = string(data[:])
	if tmp == string(DARK) {
		return DARK, nil
	} else if tmp == string(LIGHT) {
		return LIGHT, nil
	} else {
		return NULL, nil

	}
}

// Gets the initial mode.
//
// If the location is known, the mode is computed for that location. This
// should be the default value unless the local timezone has changed or the
// device has travelled across timezones.
//
// If no location is known, load the last-known mode. This work well for
// manually controlled devices, which are unlikely to have a "last known
// location".
func GetInitialMode(location *geoclue.Location) Mode {
	if location != nil {
		if mode, err := DetermineModeForRightNow(*location); err != nil {
			log.Println("Could not determine mode for location:", err)
			return NULL
		} else {
			return mode
		}
	} else if mode, err := readModeFromCache(); err != nil {
		log.Println("Could not load previous mode from cache:", err)
		return NULL
	} else {
		return mode
	}
}

// Run the darkman service.
func ExecuteService(ctx context.Context, readyFd *os.File) error {
	log.SetFlags(log.Lshortfile)

	config, err := ReadConfig()
	if err != nil {
		log.Println("Could not read configuration file:", err)
	}

	initialLocation := readLocationFromCache()
	if initialLocation != nil {
		log.Println("Read location from cache:", initialLocation)
	} else {
		initialLocation, err = config.GetLocation()
		if err != nil {
			log.Println("No location found via config.")
		} else {
			log.Println("Found location in config:", initialLocation)
		}
	}

	initialMode := GetInitialMode(initialLocation)
	log.Println("Initial mode set to:", initialMode)

	service := NewService(initialMode)
	service.AddListener(RunScripts)
	service.AddListener(saveModeToCache)

	if config.DBusServer {
		log.Println("Running with D-Bus server.")
		dbus, err := NewDbusServer(ctx, initialMode, service.ChangeMode)
		if err != nil {
			return err
		}
		service.AddListener(dbus.ChangeMode)
	} else {
		log.Println("Running without D-Bus server.")
	}

	if config.Portal {
		log.Println("Running with XDG portal.")
		portal, err := NewPortal(ctx, initialMode)
		if err != nil {
			return err
		}
		service.AddListener(portal.ChangeMode)
	} else {
		log.Println("Running without XDG portal.")
	}

	if initialLocation != nil || config.UseGeoclue {
		// Start after registering all callbacks, so that the first changes
		// are triggered after they're all listening.
		err = NewScheduler(ctx, initialLocation, service.ChangeMode, config.UseGeoclue)
		if err != nil {
			return fmt.Errorf("failed to initialise service scheduler: %v", err)
		}
	} else {
		log.Println("Not using geoclue and no configured location.")
		log.Println("No automatic transitions will be scheduled.")
	}

	if readyFd != nil {
		if _, err = readyFd.Write([]byte("\n")); err != nil {
			return fmt.Errorf("error writing to ready-fd: %v", err)
		}
		if err = readyFd.Close(); err != nil {
			return fmt.Errorf("error closing ready-fd: %v", err)
		}
	}

	// Run until explicitly stopped.
	<-ctx.Done()

	return nil
}
