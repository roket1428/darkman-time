package main

import (
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
	transitions     chan Location
	timers          chan struct{}
	currentLocation *Location
	currentMode     Mode
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

	timeUntilDark := sundown.Sub(time.Now().UTC())
	timeUntilLight := sunrise.Sub(time.Now().UTC())

	var timeUntilNext time.Duration
	if timeUntilDark < timeUntilLight {
		timeUntilNext = timeUntilDark
	} else {
		timeUntilNext = timeUntilLight
	}

	SetTimer(timeUntilNext, timers)
}

func UpdateCurrentMode() {
	if currentLocation == nil {
		log.Println("Cannot transition because we don't have a location yet.")
		// XXX: Maybe this should be fatal? It should really never happen.
		return
	}

	p := sunrisesunset.Parameters{
		Latitude:  currentLocation.Lat,
		Longitude: currentLocation.Lng,
		UtcOffset: 0,
		Date:      time.Now().UTC(),
	}
	sunrise, sundown, err := p.GetSunriseSunset()
	if err != nil {
		log.Printf("An error ocurred trying to calculate sundown/sunrise: %v", err)
		return
	}

	now := time.Now().UTC()

	if now.Before(sunrise) {
		log.Println("It's before sunrise.")
		currentMode = DARK
	} else if now.Before(sundown) {
		log.Println("It's past sunrise and before sundown.")
		currentMode = LIGHT
	} else {
		log.Println("It's past sundown.")
		currentMode = DARK
	}
}

// A single tick.
//
// Update the mode based on the current time, execute transition, and set the
// timer for the next tick.
func Tick() {
	UpdateCurrentMode()
	Transition(currentMode)
	setNextAlarm(*currentLocation)
}

func main() {
	log.SetFlags(log.Lshortfile)

	locations = make(chan Location)
	transitions = make(chan Location)
	timers = make(chan struct{})
	currentMode = NULL

	// Set timer based on locaiton updates:
	go func() {
		for {
			loc := <-locations
			log.Printf("Got a location update: %v.\n", loc)
			if currentLocation != nil && loc == *currentLocation {
				log.Println("Location has not changed, nothing to do.")
			} else {
				currentLocation = &loc
				Tick()
			}
		}
	}()

	go func() {
		for {
			<-timers
			Tick()
		}
	}()

	// Initialise the location services:
	go CacheLocationService(locations)

	// Sleep silently forever...
	select {}
}
