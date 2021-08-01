package main

import (
	"fmt"
	"log"
	"time"

	"github.com/kelvins/sunrisesunset"
)

type Mode string

const (
	NULL  Mode = "null" // Only used while still initialising.
	LIGHT Mode = "light"
	DARK  Mode = "dark"
)

var (
	locations       chan Location
	transitions     chan Mode
	currentLocation *Location
	locationService LocationService
	dbusServer      *ServerHandle = NewDbusServer()
)

func NextSunriseAndSundown(loc Location) (sunrise time.Time, sundown time.Time, err error) {
	now := time.Now().UTC()
	p := sunrisesunset.Parameters{
		Latitude:  loc.Lat,
		Longitude: loc.Lng,
		UtcOffset: 0,
		Date:      now,
	}
	sunrise, sundown, err = p.GetSunriseSunset()
	if err != nil {
		return
	}

	// If sunrise has passed today, the next one is tomorrow:
	if sunrise.Before(now) {
		var sundownTomorrow time.Time

		p = sunrisesunset.Parameters{
			Latitude:  loc.Lat,
			Longitude: loc.Lng,
			UtcOffset: 0,
			Date:      now.Add(time.Hour * 24),
		}
		sunrise, sundownTomorrow, err = p.GetSunriseSunset()
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

func setNextAlarm(loc Location) {
	sunrise, sundown, err := NextSunriseAndSundown(loc)

	if err != nil {
		log.Printf("An error ocurred trying to calculate sundown/sunrise: %v", err)
		return
	}

	// If no error has occurred, print the results
	log.Println("Next sunrise:", sunrise)
	log.Println("Next sundown:", sundown)

	var nextTick time.Time
	if sunrise.Before(sundown) {
		nextTick = sunrise
	} else {
		nextTick = sundown
	}

	now := time.Now().UTC()
	sleepFor := nextTick.Sub(now)

	SetTimer(sleepFor)
}

func GetCurrentMode(location Location) (Mode, error) {
	p := sunrisesunset.Parameters{
		Latitude:  location.Lat,
		Longitude: location.Lng,
		UtcOffset: 0,
		Date:      time.Now().UTC(),
	}
	sunrise, sundown, err := p.GetSunriseSunset()
	if err != nil {
		return NULL, fmt.Errorf("an error ocurred trying to calculate sundown/sunrise: %v", err)
	}

	// Add one minute here to compensate for rounding.
	// When woken up by the clock, it might be a few milliseconds too early
	// due to rounding. Rather than seek to be more precise (which is
	// unnecessary), just do what we'd do in a minute.
	now := time.Now().UTC().Add(time.Minute)

	if now.Before(sunrise) {
		log.Println("It's before sunrise.")
		return DARK, nil
	} else if now.Before(sundown) {
		log.Println("It's past sunrise and before sundown.")
		return LIGHT, nil
	} else {
		log.Println("It's past sundown.")
		return DARK, nil
	}
}

// A single tick.
//
// Update the mode based on the current time, execute transition, and set the
// timer for the next tick.
func Tick() {
	mode, err := GetCurrentMode(*currentLocation)
	if err != nil {
		log.Printf("Error determining current mode transition: %v", err)
		return
	}
	transitions <- mode
	setNextAlarm(*currentLocation)
}

func main() {
	log.SetFlags(log.Lshortfile)

	locations = make(chan Location)
	transitions = make(chan Mode)

	// Set timer based on location updates:
	go func() {
		for {
			loc := <-locations
			log.Printf("Now using location %v.\n", loc)
			if currentLocation != nil && loc == *currentLocation {
				log.Println("Location has not changed, nothing to do.")
			} else {
				currentLocation = &loc
				Tick()
			}
		}
	}()

	// Initialise the location services:
	locationService = *StartLocationService(locations)

	// Listen for the alarm that wakes us up:
	go func() {
		for {
			<-Alarms
			// On wakeup, poll location again.
			// This'll generally be just twice a day.
			err := locationService.Poll()
			if err != nil {
				log.Printf("Failed to poll location: %v\n", err)
			}
			Tick()
		}
	}()

	// Do things when we get mode transitions:
	go func() {
		previousMode := NULL
		for {
			mode := <-transitions

			if mode != previousMode {
				RunScripts(mode)
				dbusServer.ChangeMode(string(mode))
			}
		}
	}()

	// Sleep silently forever...
	select {}
}
