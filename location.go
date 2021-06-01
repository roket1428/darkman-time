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

// Resolves locations.
func CacheLocationService(c chan Location) { 
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
	// TODO: Read from geoclue.
}

// XXX: I need a way to re-trigger geoclue!!!
func LocationService(c chan Location) {
}
