package main

import (
	"fmt"
	"log"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type Config struct {
	Lat        float64
	Lng        float64
	UseGeoclue bool // TODO: Not yet implemented
}

func ReadConfig() (*Config, error) {
	filePath, err := xdg.ConfigFile("darkman")
	if err != nil {
		return nil, err
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filePath)

	viper.SetDefault("Lat", -1)
	viper.SetDefault("Lng", -1)
	viper.SetDefault("UseGeoclue", true)

	err = viper.ReadInConfig()
	if err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Println("Could not read configuration file, ", err)
	}

	config := &Config{}
	err = viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("could not properly decode config, %v", err)
	}

	return config, nil
}

func (config *Config) GetLocation() (*geoclue.Location, error) {
	if config.Lat < 0 || config.Lng < 0 {
		return nil, fmt.Errorf("no valid location in the config")
	}

	location := geoclue.Location{
		Lat: config.Lat,
		Lng: config.Lng,
	}

	return &location, nil
}