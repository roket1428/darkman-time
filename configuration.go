package darkman

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/adrg/xdg"
	"github.com/rxwycdh/rxhash"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Lat        *float64
	Lng        *float64
	UseGeoclue bool
	DBusServer bool
	Portal     bool
}

// Returns a new Config with the default values.
func New() Config {
	return Config{
		Lat:        nil,
		Lng:        nil,
		UseGeoclue: false,
		DBusServer: true,
		Portal:     true,
	}
}

// Returns nil if the environment variable is unset.
func ReadFloatEnvVar(name string) (*float64, error) {
	if raw, ok := os.LookupEnv(name); ok {
		if value, err := strconv.ParseFloat(raw, 64); err != nil {
			return nil, fmt.Errorf("%v is not a valid number: %v", name, err)
		} else {
			return &value, nil
		}
	}
	return nil, nil
}

// Returns nil if the environment variable is unset.
func ReadBoolEnvVar(name string) (*bool, error) {
	if raw, ok := os.LookupEnv(name); ok {
		if value, err := strconv.ParseBool(raw); err != nil {
			return nil, fmt.Errorf("%v is not a valid boolean: %v", name, err)
		} else {
			return &value, nil
		}
	}
	return nil, nil
}

// Returns nil if the variable is unset.
func FloatFromYaml(yamlConfig map[interface{}]interface{}, key string) (*float64, error) {
	if value, ok := yamlConfig[key]; ok {
		if value, ok := (value).(float64); ok {
			return &value, nil
		} else {
			return nil, fmt.Errorf("%v is not a valid number", key)
		}
	}
	return nil, nil
}

// Returns nil if the variable is unset.
func BoolFromYaml(yamlConfig map[interface{}]interface{}, key string) (*bool, error) {
	if value, ok := yamlConfig[key]; ok {
		if value, ok := (value).(bool); ok {
			return &value, nil
		} else {
			return nil, fmt.Errorf("%v is not a valid boolean", key)
		}
	}
	return nil, nil
}

// Loads and updates configuration in place.
//
// Returns error for invalid settings. All fields are considered optional.
func (config *Config) LoadFromEnv() error {
	if lat, err := ReadFloatEnvVar("DARKMAN_LAT"); err != nil {
		return err
	} else if lat != nil {
		config.Lat = lat
	}

	if lng, err := ReadFloatEnvVar("DARKMAN_LNG"); err != nil {
		return err
	} else if lng != nil {
		config.Lng = lng
	}

	if usegeoclue, err := ReadBoolEnvVar("DARKMAN_USEGEOCLUE"); err != nil {
		return err
	} else if usegeoclue != nil {
		config.UseGeoclue = *usegeoclue
	}

	if dbusserver, err := ReadBoolEnvVar("DARKMAN_DBUSSERVER"); err != nil {
		return err
	} else if dbusserver != nil {
		config.DBusServer = *dbusserver
	}

	if portal, err := ReadBoolEnvVar("DARKMAN_PORTAL"); err != nil {
		return err
	} else if portal != nil {
		config.Portal = *portal
	}

	return nil
}

// Loads a new configuration.
//
// Returns error for invalid settings. All fields are considered optional.
// Fails is any unknown fields are found (this usually indicates a typy). Does
// not overwrite any values already defined in `config`.
func (config *Config) LoadFromYamlFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	yamlDecoder := yaml.NewDecoder(file)
	yamlDecoder.KnownFields(true)

	if err := yamlDecoder.Decode(&config); err != nil {
		return fmt.Errorf("error parsing configuration file: %s", err)
	}

	return nil
}

func ReadConfig() (*Config, error) {
	config := New()

	configDir, err := xdg.ConfigFile("darkman")
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(configDir, "config.yaml")

	if err := config.LoadFromYamlFile(configFile); err != nil {
		log.Println(err)
		// TODO: Should actually bail here, but this is a breaking change:
		// return nil, fmt.Errorf("error reading configuration file: %v", configFile)
	} else {
		log.Println("Using config file:", configFile)
	}

	if err := config.LoadFromEnv(); err != nil {
		return nil, fmt.Errorf("error reading environment variables: %v", err)
	}

	log.Printf("Loaded configuration: %v\n", config)
	return &config, nil
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

func (config *Config) Hash() (string, error) {
	return rxhash.HashStruct(config)
}
