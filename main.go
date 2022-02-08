// Package darkman implements darkman's service itself.
//
// This package is used by gitlab.com/WhyNotHugo/darkman/cmd, which is the cli
// that wraps around the service and the client.
package darkman

import (
	"log"
	"time"

	"github.com/sj14/astral"

	"gitlab.com/WhyNotHugo/darkman/boottimer"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type Mode string

const (
	NULL  Mode = "null" // Only used while still initialising.
	LIGHT Mode = "light"
	DARK  Mode = "dark"
)

var (
	config *Config
)

// Return the time for sunrise and sundown for a given day and location.
func SunriseAndSundown(loc geoclue.Location, now time.Time) (sunrise time.Time, sundown time.Time, err error) {
	obs := astral.Observer{
		Latitude:  loc.Lat,
		Longitude: loc.Lng,
		Elevation: loc.Alt,
	}
	sunrise, err = astral.Sunrise(obs, now)
	if err != nil {
		return
	}

	sundown, err = astral.Sunset(obs, now)
	return
}

// Returns the time of the next sunrise and the next sundown.
// Note that they next sundown may be before the next sunrise or viceversa.
func NextSunriseAndSundown(loc geoclue.Location, now time.Time) (sunrise time.Time, sundown time.Time, err error) {
	sunrise, sundown, err = SunriseAndSundown(loc, now)

	// If sunrise has passed today, the next one is tomorrow:
	if sunrise.Before(now) {
		var sundownTomorrow time.Time

		sunrise, sundownTomorrow, err = SunriseAndSundown(loc, now.Add(time.Hour*24))
		if err != nil {
			return
		}

		// It might also be past sundown today:
		if sundown.Before(now) {
			sundown = sundownTomorrow
		}
	}

	return
}

func CalculateCurrentMode(nextSunrise time.Time, nextSundown time.Time) Mode {
	if nextSunrise.Before(nextSundown) {
		log.Println("Sunrise comes first; so it's night time.")
		return DARK
	} else {
		log.Println("Sundown comes first; so it's day time.")
		return LIGHT
	}
}

func ExecuteService() {
	log.SetFlags(log.Lshortfile)

	var err error // Declared to avoid creating a local variable `config`.
	config, err = ReadConfig()
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

			if err != nil {
				log.Printf("Failed to poll location: %v\n", err)
			}

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
