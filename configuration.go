package darkman

import (
	"fmt"
	"log"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
)

type Config struct {
	Lat        *float64
	Lng        *float64
	UseGeoclue bool
	DBusServer bool
	Portal     bool
}

func ReadConfig() (*Config, error) {
	filePath, err := xdg.ConfigFile("darkman")
	if err != nil {
		return nil, err
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filePath)

	viper.SetDefault("Lat", nil)
	viper.SetDefault("Lng", nil)
	viper.SetDefault("UseGeoclue", true)
	viper.SetDefault("DBusServer", true)
	viper.SetDefault("Portal", true)

	err = viper.ReadInConfig()
	if err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Println(err)
	}

	// Load env vars (e.g.: DARKMAN_LAT) too.
	viper.SetEnvPrefix("darkman")
	viper.AutomaticEnv()

	config := &Config{}
	err = viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("could not properly decode config, %v", err)
	}

	return config, nil
}

func (config *Config) GetLocation() (*geoclue.Location, error) {
	if config.Lat == nil || config.Lng == nil {
		return nil, fmt.Errorf("no valid location in the config")
	}

	location := geoclue.Location{
		Lat: *config.Lat,
		Lng: *config.Lng,
	}

	return &location, nil
}
