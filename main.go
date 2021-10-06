package main

import (
	"log"
	"time"

	"github.com/kelvins/sunrisesunset"

	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type Mode string

const (
	NULL  Mode = "null" // Only used while still initialising.
	LIGHT Mode = "light"
	DARK  Mode = "dark"
)

var (
	config          *Config
	locations       chan geoclue.Location
	transitions     chan Mode
	currentLocation *geoclue.Location
	locationService *LocationService
	dbusServer      *ServerHandle = NewDbusServer()
)

func SunriseAndSundown(loc geoclue.Location, now time.Time) (sunrise time.Time, sundown time.Time, err error) {
	p := sunrisesunset.Parameters{
		Latitude:  loc.Lat,
		Longitude: loc.Lng,
		UtcOffset: 0,
		Date:      now,
	}
	sunrise, sundown, err = p.GetSunriseSunset()

	return
}

func NextSunriseAndSundown(loc geoclue.Location, now time.Time, curSunrise time.Time, curSundown time.Time) (sunrise time.Time, sundown time.Time, err error) {
	sunrise = curSunrise
	sundown = curSundown

	// If sunrise has passed today, the next one is tomorrow:
	if sunrise.Before(now) {
		var sundownTomorrow time.Time

		p := sunrisesunset.Parameters{
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

func setNextAlarm(now time.Time, curMode Mode, sunrise time.Time, sundown time.Time) {
	log.Println("Next sunrise:", sunrise)
	log.Println("Next sundown:", sundown)

	var nextTick time.Time
	if curMode == DARK {
		nextTick = sunrise
	} else {
		nextTick = sundown
	}

	sleepFor := nextTick.Sub(now)

	SetTimer(sleepFor)
}

func GetCurrentMode(now time.Time, sunrise time.Time, sundown time.Time) (Mode) {
	// Add one minute here to compensate for rounding.
	// When woken up by the clock, it might be a few milliseconds too early
	// due to rounding. Rather than seek to be more precise (which is
	// unnecessary), just do what we'd do in a minute.
	now = now.Add(time.Minute)

	if now.Before(sunrise) {
		log.Println("It's before sunrise.")
		return DARK
	} else if now.Before(sundown) {
		log.Println("It's past sunrise and before sundown.")
		return LIGHT
	} else {
		log.Println("It's past sundown.")
		return DARK
	}
}

// A single tick.
//
// Update the mode based on the current time, execute transition, and set the
// timer for the next tick.
func Tick() {
	now := time.Now().UTC()
	sunrise, sundown, err := SunriseAndSundown(*currentLocation, now)
	if err != nil {
		log.Printf("An error occurred trying to calculate sundown/sunrise: %v", err)
		return
	}

	mode := GetCurrentMode(now, sunrise, sundown)
	transitions <- mode

	sunrise, sundown, err = NextSunriseAndSundown(*currentLocation, now, sunrise, sundown)
	if err != nil {
		log.Printf("An error occurred trying to calculate next sundown/sunrise: %v", err)
		return
	}
	setNextAlarm(now, mode, sunrise, sundown)
}

func init() {
	log.SetFlags(log.Lshortfile)

	var err error
	config, err = ReadConfig()
	if err != nil {
		log.Println("Could not read configuration file:", err)
	}
}

func main() {
	locations = make(chan geoclue.Location)
	transitions = make(chan Mode)

	initialLocation, err := config.GetLocation()
	if err != nil {
		log.Println("No location found via config.")
	} else {
		log.Println("Found location in config:", initialLocation)
		currentLocation = initialLocation
	}

	// Set timer based on location updates:
	go func() {
		for {
			loc := <-locations
			log.Printf("Now using location %v.\n", loc)

			if currentLocation != nil && loc == *currentLocation {
				log.Println("Location has not changed, nothing to do.")
				continue
			}

			currentLocation = &loc
			Tick()
		}
	}()

	// Initialise the location services:
	locationService, err = StartLocationService(locations)
	if err != nil {
		log.Printf("Could not start location service: %v.\n", err)
	}

	// Listen for the alarm that wakes us up:
	go func() {
		for {
			<-Alarms
			// On wakeup, poll location again.
			// This'll generally be just twice a day.
			if locationService != nil {
				err = locationService.Poll()
				if err != nil {
					log.Printf("Failed to poll location: %v\n", err)
				}
			}
			Tick()
		}
	}()

	// Do things when we get mode transitions:
	go func() {
		previousMode := NULL
		for {
			mode := <-transitions

			log.Printf("Mode should now be: %v mode.\n", mode)
			if mode == previousMode {
				log.Println("No transition necessary")
				continue
			}

			RunScripts(mode)
			dbusServer.ChangeMode(string(mode))
			previousMode = mode
		}
	}()

	// Sleep silently forever...
	select {}
}
