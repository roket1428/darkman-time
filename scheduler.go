package darkman

import (
	"log"
	"time"

	"github.com/sj14/astral"

	"gitlab.com/WhyNotHugo/darkman/boottimer"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
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

type Scheduler struct {
	currentMode     Mode
	currentLocation *geoclue.Location
	listeners       *[]chan Mode
}

// The scheduler schedules timer to wake up in time for the next sundown/sunrise.
func NewScheduler() Scheduler {
	handler := Scheduler{
		currentMode: NULL,
		listeners:   &[]chan Mode{},
	}

	return handler
}

func (handler *Scheduler) AddListener(c chan Mode) {
	*handler.listeners = append(*handler.listeners, c)
}

func (handler *Scheduler) UpdateLocation(newLocation geoclue.Location) {
	if handler.currentLocation != nil && newLocation == *handler.currentLocation {
		log.Println("Location has not changed, nothing to do.")
		return
	}

	handler.currentLocation = &newLocation
	handler.Tick()
}

// A single tick.
//
// Update the mode based on the current time, execute transition, and set the
// timer for the next tick.
func (handler *Scheduler) Tick() {
	if handler.currentLocation == nil {
		log.Println("No location yet, nothing to do.")
		return
	}

	now := time.Now().UTC()

	// Add one minute here to compensate for rounding.
	//
	// When woken up by the clock, it might be a few milliseconds too early
	// due to rounding. Rather than seek to be more precise (which is
	// unnecessary), just do what we'd do in a minute.
	//
	// TODO: with recent changes, this might no longer be necessary, but
	// needs to be well tested.
	sunrise, sundown, err := NextSunriseAndSundown(*handler.currentLocation, now.Add(time.Minute))
	if err != nil {
		log.Printf("Error calculating next sundown/sunrise: %v", err)
		return
	}

	mode := CalculateCurrentMode(sunrise, sundown)
	handler.notifyListeners(mode)

	setNextAlarm(now, mode, sunrise, sundown)
}

/// Apply a transition if applicable.
func (handler *Scheduler) notifyListeners(mode Mode) {
	log.Printf("Mode should now be: %v mode.\n", mode)
	if mode == handler.currentMode {
		log.Println("No transition necessary")
		return
	}

	handler.currentMode = mode
	for _, c := range *handler.listeners {
		c <- mode
	}
}

func setNextAlarm(now time.Time, curMode Mode, sunrise time.Time, sundown time.Time) {
	log.Println("Next sunrise:", sunrise)
	log.Println("Next sundown:", sundown)

	var nextTick time.Time
	if sunrise.Before(sundown) {
		nextTick = sunrise
		log.Println("Will set an alarm for sunrise")
	} else {
		nextTick = sundown
		log.Println("Will set an alarm for sundown")
	}

	sleepFor := nextTick.Sub(now)
	boottimer.SetTimer(sleepFor)
}
