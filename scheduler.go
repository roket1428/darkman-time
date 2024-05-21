package darkman

import (
	"context"
	"fmt"
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

func NextSunriseAndSundownTime(configTime Time, now time.Time) (sunrise time.Time, sundown time.Time, err error) {
	sunrise = configTime.Sunrise
	sundown = configTime.Sunset
	// If sunrise has passed today, the next one is tomorrow:
	if sunrise.Before(now) {
		var sundownTomorrow time.Time
		sunrise = sunrise.Add(time.Hour * 24)
		sundownTomorrow = sundown.Add(time.Hour * 24)
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

// Scheduler handles setting timers based on the current location, and
// trigering changes based on the current location and sun position.
type Scheduler struct {
	currentLocation *geoclue.Location
	currentTime     *Time
	changeCallback  func(Mode)
	latestTimer     *boottimer.Timer
}

// The scheduler schedules timer to wake up in time for the next sundown/sunrise.
func NewScheduler(ctx context.Context, initialLocation *geoclue.Location, initialTime *Time, changeCallback func(Mode), useGeoclue bool) error {
	scheduler := Scheduler{
		changeCallback: changeCallback,
	}

	newLocations := make(chan (geoclue.Location))
	chanTime := make(chan (Time))
	// Alarms wake us up when it's time for the next transition.
	go func() {
		for {
			select {
			case <-boottimer.Alarms:
				scheduler.Tick(ctx)
			case <-ctx.Done():
				scheduler.stop()
				return
				// The timer itself also has ctx.
			case loc := <-newLocations:
				if scheduler.currentLocation != nil && loc == *scheduler.currentLocation {
					log.Println("Location has not changed, nothing to do.")
				} else {
					scheduler.currentLocation = &loc
					scheduler.Tick(ctx)
				}
			case tm := <-chanTime:
				scheduler.currentTime = &tm
				scheduler.Tick(ctx)
			}
		}
	}()

	if useGeoclue {
		if err := GetLocations(ctx, newLocations); err != nil {
			return fmt.Errorf("could not start location service: %v", err)
		}
		return nil
	}

	if initialLocation != nil {
		log.Println("Not using geoclue; using static location.")
		newLocations <- *initialLocation
		return nil
	}

	if initialTime != nil {
		log.Println("Not using geoclue or static location; using custom sunrise and sunset.")
		chanTime <- *initialTime
		return nil
	}

	return fmt.Errorf("no location source available")
}

// A single tick.
//
// Update the mode based on the current time, execute transition, and set the
// timer for the next tick.
func (handler *Scheduler) Tick(ctx context.Context) {
	if handler.currentLocation == nil && handler.currentTime == nil {
		log.Println("No location or time yet, nothing to do.")
		return
	}

	now := time.Now()

	// Add one minute here to compensate for rounding.
	//
	// When woken up by the clock, it might be a few milliseconds too early
	// due to rounding. Rather than seek to be more precise (which is
	// unnecessary), just do what we'd do in a minute.
	//
	// TODO: with recent changes, this might no longer be necessary, but
	// needs to be well tested.
	var err error
	var sunrise, sundown time.Time
	if handler.currentTime != nil {
		sunrise, sundown, err = NextSunriseAndSundownTime(*handler.currentTime, now.Add(time.Minute))
	} else {
		sunrise, sundown, err = NextSunriseAndSundown(*handler.currentLocation, now.Add(time.Minute))
	}
	if err != nil {
		log.Printf("Error calculating next sundown/sunrise: %v", err)
		return
	}

	mode := CalculateCurrentMode(sunrise, sundown)
	handler.changeCallback(mode)

	handler.setNextAlarm(ctx, now, mode, sunrise, sundown)
}

func DetermineModeForRightNow(location geoclue.Location) (Mode, error) {
	now := time.Now()
	sunrise, sundown, err := NextSunriseAndSundown(location, now.Add(time.Minute))
	if err != nil {
		return NULL, fmt.Errorf("error calculating next sundown/sunrise: %v", err)
	}

	return CalculateCurrentMode(sunrise, sundown), nil
}

func DetermineModeForRightNowTime(configTime Time) (Mode, error) {
	now := time.Now()
	sunrise, sundown, err := NextSunriseAndSundownTime(configTime, now.Add(time.Minute))
	if err != nil {
		return NULL, fmt.Errorf("error calculating next sundown/sunrise: %v", err)
	}

	return CalculateCurrentMode(sunrise, sundown), nil
}

func (handler *Scheduler) setNextAlarm(ctx context.Context, now time.Time, curMode Mode, sunrise time.Time, sundown time.Time) {
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

	// Need to move the timer into the heap before assigning.
	timer := boottimer.SetTimer(sleepFor)
	handler.latestTimer = &timer
}

func (handler *Scheduler) stop() {
	if handler.latestTimer != nil {
		handler.latestTimer.Delete()
	}
}
