package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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

// Resolves locations.
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

func initGeoclue(c chan geoclue.Location) (client *geoclue.Geoclient, err error) {
	client, err = geoclue.NewClient("darkman")
	if err != nil {
		return nil, err
	}
	log.Println("Geoclue initialised.")

	go func() {
		for {
			loc := <-client.Locations

			err := saveLocationToCache(loc)
			if err != nil {
				log.Println("Error saving location to cache: ", loc)
			} else {
				log.Println("Saved location to cache.")
			}

			c <- loc

			err = client.StopClient()
			if err != nil {
				log.Fatalln("Error stopping client.", err)
			}
		}
	}()

	return
}

type LocationService struct {
	locations chan geoclue.Location
	geoclue   geoclue.Geoclient
}

// Update the location once, and go back to sleep.
func (service LocationService) Poll() error {
	return service.geoclue.StartClient()
}

func StartLocationService(c chan geoclue.Location) (*LocationService, error) {
	location := readLocationFromCache()
	if location != nil {
		log.Println("Read location from cache:", location)
		c <- *location
	}

	// TODO: read from env var in the same manner we read from the cache.

	// TODO: allow disabling geoclue via an env var.
	geoclue, err := initGeoclue(c)
	if err != nil {
		return nil, fmt.Errorf("error initialising geoclue: %v", err)
	}

	service := LocationService{
		locations: c,
		geoclue:   *geoclue,
	}

	err = geoclue.StartClient()
	if err != nil {
		return nil, fmt.Errorf("error initialising geoclue: %v", err)
	}

	return &service, nil

}
