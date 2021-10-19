package main

import (
	"log"
	"time"

	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type Scheduler struct {
	currentMode     Mode
	currentLocation *geoclue.Location
	listeners       []chan Mode
}

func NewScheduler() Scheduler {
	handler := Scheduler{
		currentMode: NULL,
	}

	return handler
}

func (handler *Scheduler) AddListener(c chan Mode) {
	handler.listeners = append(handler.listeners, c)
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
	sunrise, sundown, err := SunriseAndSundown(*handler.currentLocation, now)
	if err != nil {
		// This is fatal; there's nothing we can do if this fails.
		log.Fatalln("Error calculating today's sundown/sunrise", err)
	}

	mode := GetCurrentMode(now, sunrise, sundown)
	handler.notifyListeners(mode)

	sunrise, sundown, err = NextSunriseAndSundown(*handler.currentLocation, now, sunrise, sundown)
	if err != nil {
		log.Printf("Error calculating next sundown/sunrise: %v", err)
		return
	}
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
	for _, c := range handler.listeners {
		c <- mode
	}
}
