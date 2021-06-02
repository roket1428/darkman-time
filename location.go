package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/adrg/xdg"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// NOTE: Geoclue continues polling in the background every few minutes, so if
// we fail to start or stop it, we hard fail and exit. Geoclue will detect that
// we (the client) has existed, and stop polling.
//
// Errors here are hard to handle, since we can't know geoclue's state, and we
// can't control it and tell it to stop either.

func saveLocationToCache(loc Location) error {
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
func readLocationFromCache() (location *Location) {
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

	location = &Location{}
	err = json.Unmarshal(data, location)
	if err != nil {
		log.Printf("Error parsing data from cache file path: %v\n", err)
		return nil
	}

	return
}

func manualLocationService(c chan Location) {
	// TODO: Read from an environment variable.
}

func initGeoclue(c chan Location) (geoclue *Geoclient, err error) {
	// Intercept non-cache values here and put them in the cache:
	proxy := make(chan Location, 10)

	geoclue, err = NewClient("darkman", proxy)
	if err != nil {
		log.Println("Fatal error initialising geoclue: ", err)
		return 
	}
	log.Println("Geoclue initialised.")

	go func() {
		for {

			loc := <-proxy

			err := saveLocationToCache(loc)
			if err != nil {
				log.Println("Error saving location to cache: ", loc)
			} else {
				log.Println("Saved location to cache.")
			}

			c <- loc

			err = geoclue.StopClient()
			if err != nil {
				log.Fatalln("Error stopping client.", err)
			}
		}
	}()

	return
}

type LocationService struct {
	locations chan Location
	geoclue   Geoclient
}

// Update the location once, and go back to sleep.
func (service LocationService) Poll() {
	service.geoclue.StartClient()
}

func StartLocationService(c chan Location) *LocationService {
	location := readLocationFromCache()
	if location != nil {
		log.Println("Read location from cache.")
		c <- *location
	}

	// TODO: read from env var in the same manner we read from the cache.

	// TODO: allow disabling geoclue via an env var.
	geoclue, err := initGeoclue(c)
	if err != nil {
		log.Println("Fatal error initialising geoclue: ")
		return nil
	}

	service := LocationService{
		locations: c,
		geoclue:   *geoclue,
	}

	err = geoclue.StartClient()
	if err != nil {
		log.Fatalln("Fatal error starting geoclue: ", err)
	}

	return &service
}
