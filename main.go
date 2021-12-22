package main

import (
	"log"
	"time"

	"github.com/kelvins/sunrisesunset"

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

	boottimer.SetTimer(sleepFor)
}

func GetCurrentMode(now time.Time, sunrise time.Time, sundown time.Time) Mode {
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

func init() {
	log.SetFlags(log.Lshortfile)

	var err error
	config, err = ReadConfig()
	if err != nil {
		log.Println("Could not read configuration file:", err)
	}
}

func main() {
	initialLocation, err := config.GetLocation()
	if err != nil {
		log.Println("No location found via config.")
	} else {
		log.Println("Found location in config:", initialLocation)
	}

	scheduler := NewScheduler()
	locationService := NewLocationService(initialLocation)
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
			newLocation := <-locationService.C
			log.Println("Location service has yielded:", newLocation)
			scheduler.UpdateLocation(newLocation)
		}
	}()

	// Alarms wake us up when it's time for the next transition.
	go func() {
		for {
			<-boottimer.Alarms
			// On wakeup, poll location again.
			// This'll generally be just twice a day.
			err = locationService.Poll()
			if err != nil {
				log.Printf("Failed to poll location: %v\n", err)
			}

			scheduler.Tick()
		}
	}()

	err = locationService.Poll()
	if err != nil {
		log.Println("Could not start location service:", err)
	}

	// Sleep silently forever...
	select {}
}
