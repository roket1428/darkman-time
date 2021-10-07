package main

import (
	"log"
	"time"

	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type TransitionHandler struct {
	currentLocation *geoclue.Location
	dbusServer      ServerHandle
	transitions     chan Mode
}

func NewTransitionHandler(dbusServer ServerHandle) TransitionHandler {
	handler := TransitionHandler{
		dbusServer:  dbusServer,
		transitions: make(chan Mode),
	}

	go handler.waitForTransitions()

	return handler
}

func (handler *TransitionHandler) UpdateLocation(newLocation geoclue.Location) {
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
func (handler *TransitionHandler) Tick() {
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
	handler.transitions <- mode

	sunrise, sundown, err = NextSunriseAndSundown(*handler.currentLocation, now, sunrise, sundown)
	if err != nil {
		log.Printf("Error calculating next sundown/sunrise: %v", err)
		return
	}
	setNextAlarm(now, mode, sunrise, sundown)
}

/// Waits for transitions to happen and executes necessary actions.
func (handler *TransitionHandler) waitForTransitions() {
	previousMode := NULL
	for {
		mode := <-handler.transitions

		log.Printf("Mode should now be: %v mode.\n", mode)
		if mode == previousMode {
			log.Println("No transition necessary")
			continue
		}

		RunScripts(mode)
		handler.dbusServer.ChangeMode(string(mode))
		previousMode = mode
	}
}
