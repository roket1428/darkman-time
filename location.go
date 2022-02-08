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
	client, err = geoclue.NewClient("darkman", time.Minute)
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
	C       chan geoclue.Location
	geoclue *geoclue.Geoclient
}

// Update the location once, and go back to sleep.
func (service LocationService) Poll() error {
	if service.geoclue == nil {
		var err error
		service.geoclue, err = initGeoclue(service.C)

		if err != nil {
			return fmt.Errorf("error initialising geoclue: %v", err)
		}

	}

	return service.geoclue.StartClient()
}

func NewLocationService(initial *geoclue.Location) LocationService {
	service := LocationService{
		C:       make(chan geoclue.Location, 1),
		geoclue: nil,
	}

	if initial == nil {
		initial = readLocationFromCache()
		if initial != nil {
			log.Println("Read location from cache:", initial)
		}
	}

	if initial != nil {
		service.C <- *initial
	}

	return service

}
