package darkman

import (
	"context"
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
	if err = json.Unmarshal(data, location); err != nil {
		log.Printf("Error parsing data from cache file path: %v\n", err)
		return nil
	}

	return
}

// Initialise geoclue. Note that we have our own channel where we yield
// locations, and Geoclient has its own. We act as middleman here since we also
// keep the last location cached.
func initGeoclue(ctx context.Context, onLocation func(context.Context, geoclue.Location)) (client *geoclue.Geoclient, err error) {
	client, err = geoclue.NewClient(ctx, "darkman", time.Minute, 40000, 3600*4)
	if err != nil {
		return nil, err
	}

	go func() {
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case loc := <-client.Locations:
				if err := saveLocationToCache(loc); err != nil {
					log.Println("Error saving location to cache: ", loc)
				} else {
					log.Println("Saved location to cache.")
				}

				onLocation(ctx, loc)
			}
		}
	}()

	return
}

// Periodically fetch the current location.
//
// By default, we indicate set geoclue in a rather passive mode; it'll ignore
// location changes that occurr in less than four hours, or of less than 40km.
func GetLocations(ctx context.Context, onLocation func(context.Context, geoclue.Location)) (err error) {
	if _, err = initGeoclue(ctx, onLocation); err != nil {
		return fmt.Errorf("error initialising geoclue: %v", err)
	}

	return nil

}
