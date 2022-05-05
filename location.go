package darkman

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/adrg/xdg"

	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

// NOTE: Geoclue continues polling in the background every few minutes, so if
// we fail to start or stop it, we hard fail and exit. Geoclue will detect that
// we (the client) has existed, and stop polling.
//
// Errors here are hard to handle, since we can't know geoclue's state, and we
// can't control it and tell it to stop either.

func saveLocationToCache(loc geoclue.Location) error {
	cacheFilePath, err := xdg.CacheFile("darkman/location.json")
	if err != nil {
		return err
	}

	marshalled, err := json.Marshal(loc)
	if err != nil {
		return err
	}

	err = os.WriteFile(cacheFilePath, marshalled, os.FileMode(0600))

	return err
}

func readLocationFromCache() (location *geoclue.Location) {
	cacheFilePath, err := xdg.CacheFile("darkman/location.json")
	if err != nil {
		log.Printf("Error determining cache file path: %v\n", err)
		return
	}

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		log.Printf("Error reading cache file path: %v\n", err)
		return
	}

	location = &geoclue.Location{}
	err = json.Unmarshal(data, location)
	if err != nil {
		log.Printf("Error parsing data from cache file path: %v\n", err)
		return nil
	}

	return
}

// Initialise geoclue. Note that we have our own channel where we yield
// locations, and Geoclient has its own. We act as middleman here since we also
// keep the last location cached.
func initGeoclue(onLocation func(geoclue.Location)) (client *geoclue.Geoclient, err error) {
	client, err = geoclue.NewClient("darkman", time.Minute, 40000, 3600*4)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			loc := <-client.Locations

			err := saveLocationToCache(loc)
			if err != nil {
				log.Println("Error saving location to cache: ", loc)
			} else {
				log.Println("Saved location to cache.")
			}

			onLocation(loc)
		}
	}()

	return
}

// Periodically fetch the current location.
//
// When initialised, it'll yield an initial location, or fall back to a cached
// one if none is passed.
//
// It'll then initialise geoclue, and yield locations that it sends.
// My default, we indicate set geoclue in a rather passive mode; it'll ignore
// location changes that occurr in less than four hours, or of less than 40km.
func GetLocations(initial *geoclue.Location, onLocation func(geoclue.Location)) (err error) {
	// TODO: Should take a context to kill the client.
	if initial != nil {
		onLocation(*initial)
	}

	_, err = initGeoclue(onLocation)
	if err != nil {
		return fmt.Errorf("error initialising geoclue: %v", err)
	}

	return nil

}
