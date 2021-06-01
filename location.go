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
func cacheLocationService(c chan Location) {
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

	var location Location
	err = json.Unmarshal(data, &location)
	if err != nil {
		log.Printf("Error parsing data from cache file path: %v\n", err)
		return
	}

	c <- location
}

func manualLocationService(c chan Location) {
	// TODO: Read from an environment variable.
}

func geoclueLocationService(c chan Location) {
	// TODO: Allow disabling geoclue via an env var.

	client, err := NewClient("darkman", c)
	if err != nil {
		log.Println("Fatal error initialising geoclue: ", err)
		return
	}
	log.Println("Geoclue initialised.")

	err = client.StartClient()
	if err != nil {
		// Exit here on error, since having geoclue in a broken state
		// can be dangerous: if the service actually DID start, it will
		// start continuously polling location and we won't be able to
		// control it.
		log.Fatalln("Fatal error starting geoclue: ", err)
	}

	log.Println("Geoclue client started.")
}

func LocationService(c chan Location) {
	proxyChan := make(chan Location, 10)

	cacheLocationService(c)
	manualLocationService(proxyChan)
	geoclueLocationService(proxyChan)

	// Intercept non-cache values here and put them in the cache:
	for {
		loc := <-proxyChan

		err := saveLocationToCache(loc)
		if err != nil {
			log.Println("Error saving location to cache: ", loc)
		} else {
			log.Println("Saved location to cache.")
		}

		c <- loc
	}

}
