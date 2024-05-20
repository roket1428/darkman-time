package darkman

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rxwycdh/rxhash"
	"gitlab.com/WhyNotHugo/darkman/geoclue"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Lat        *float64
	Lng        *float64
	Sunrise    *string
	Sunset     *string
	UseGeoclue bool
	DBusServer bool
	Portal     bool
}

type Time struct {
	Sunrise time.Time
	Sunset  time.Time
}

// Returns a new Config with the default values.
func Default() Config {
	return Config{
		Lat:        nil,
		Lng:        nil,
		Sunrise:    nil,
		Sunset:     nil,
		UseGeoclue: false,
		DBusServer: true,
		Portal:     true,
	}
}

// Returns nil if the environment variable is unset.
func readFloatEnvVar(name string) (*float64, error) {
	if raw, ok := os.LookupEnv(name); ok {
		if value, err := strconv.ParseFloat(raw, 64); err != nil {
			return nil, fmt.Errorf("%v is not a valid number: %v", name, err)
		} else {
			return &value, nil
		}
	}
	return nil, nil
}

func readStringEnvVar(name string) *string {
	if raw, ok := os.LookupEnv(name); ok {
		if raw != "" {
			return &raw
		}
	}
	return nil
}

// Returns nil if the environment variable is unset.
func readBoolEnvVar(name string) (*bool, error) {
	if raw, ok := os.LookupEnv(name); ok {
		if value, err := strconv.ParseBool(raw); err != nil {
			return nil, fmt.Errorf("%v is not a valid boolean: %v", name, err)
		} else {
			return &value, nil
		}
	}
	return nil, nil
}

// Loads and updates configuration in place.
//
// Returns error for invalid settings. All fields are considered optional.
func (config *Config) LoadFromEnv() error {
	if lat, err := readFloatEnvVar("DARKMAN_LAT"); err != nil {
		return err
	} else if lat != nil {
		config.Lat = lat
	}

	if lng, err := readFloatEnvVar("DARKMAN_LNG"); err != nil {
		return err
	} else if lng != nil {
		config.Lng = lng
	}

	if sunrise := readStringEnvVar("DARKMAN_SUNRISE"); sunrise != nil {
		config.Sunrise = sunrise
	}

	if sunset := readStringEnvVar("DARKMAN_SUNSET"); sunset != nil {
		config.Sunset = sunset
	}

	if usegeoclue, err := readBoolEnvVar("DARKMAN_USEGEOCLUE"); err != nil {
		return err
	} else if usegeoclue != nil {
		config.UseGeoclue = *usegeoclue
	}

	if dbusserver, err := readBoolEnvVar("DARKMAN_DBUSSERVER"); err != nil {
		return err
	} else if dbusserver != nil {
		config.DBusServer = *dbusserver
	}

	if portal, err := readBoolEnvVar("DARKMAN_PORTAL"); err != nil {
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
func (config *Config) LoadFromYaml(file *os.File) error {
	yamlDecoder := yaml.NewDecoder(file)
	yamlDecoder.KnownFields(true)

	if err := yamlDecoder.Decode(&config); err != nil {
		return fmt.Errorf("error parsing configuration file: %s", err)
	}

	return nil
}

func openConfig() (*os.File, error) {
	var configHome string
	if configHomeEnv, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		configHome = filepath.Join(configHomeEnv, "darkman/config.yaml")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user home dir: %v", err)
		}
		configHome = filepath.Join(home, ".config/darkman/config.yaml")
	}
	file, err := os.Open(configHome)
	if err == nil {
		return file, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	var configDirs []string
	if configDirsEnv, ok := os.LookupEnv("XDG_CONFIG_DIRS"); ok {
		configDirs = strings.Split(configDirsEnv, ":")
	} else {
		configDirs = []string{"/etc/xdg/"}
	}
	for _, configDir := range configDirs {
		file, err := os.Open(filepath.Join(configDir, "darkman/config.yaml"))
		if err == nil {
			return file, nil
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("no configuration file found anywhere")
}

func ReadConfig(config *Config) error {
	configFile, err := openConfig()
	if err != nil {
		return fmt.Errorf("error opening config file: %s", err)
	}

	if err := config.LoadFromYaml(configFile); err != nil {
		log.Println(err)
		// TODO: Should actually bail here, but this is a breaking change:
		// return nil, fmt.Errorf("error reading configuration file: %v", configFile)
	} else {
		log.Println("Using config file:", configFile)
	}

	if err := config.LoadFromEnv(); err != nil {
		return fmt.Errorf("error reading environment variables: %v", err)
	}

	log.Printf("Loaded configuration: %v\n", &config)
	return nil
}

func (config *Config) GetLocation() (*geoclue.Location, *Time, error) {
	if ((config.Lat == nil && config.Lng != nil) ||
		(config.Lat != nil && config.Lng == nil)) &&
		((config.Sunrise == nil && config.Sunset != nil) ||
			(config.Sunrise != nil && config.Sunset == nil)) {
		return nil, nil, fmt.Errorf("no valid location / time in the config")
	}

	if config.Sunrise != nil {
		now := time.Now()
		sunriseHour, err := strconv.Atoi(strings.Split(*config.Sunrise, ":")[0])
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing time sunrise: %v", err)
		}
		sunriseMinute, err := strconv.Atoi(strings.Split(*config.Sunrise, ":")[1])
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing time sunrise: %v", err)
		}
		sunrise := time.Date(now.Year(), now.Month(), now.Day(), int(sunriseHour), int(sunriseMinute), 0, 0, now.Location())
		sunsetHour, err := strconv.ParseInt(strings.TrimLeft(strings.Split(*config.Sunset, ":")[0], "0"), 10, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing time sunset: %v", err)
		}
		sunsetMinute, err := strconv.ParseInt(strings.Split(*config.Sunset, ":")[1], 10, 0)
		sunset := time.Date(now.Year(), now.Month(), now.Day(), int(sunsetHour), int(sunsetMinute), 0, 0, now.Location())
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing time sunset: %v", err)
		}
		location := Time{
			Sunrise: sunrise,
			Sunset:  sunset,
		}
		return nil, &location, nil
	} else {
		location := geoclue.Location{
			Lat: *config.Lat,
			Lng: *config.Lng,
		}
		return &location, nil, nil
	}
}

func (config *Config) Hash() (string, error) {
	return rxhash.HashStruct(config)
}

func (config *Config) String() string {
	return fmt.Sprintf(
		"{lat: %v, lng: %v, usegeoclue: %v, dbusserver: %v, portal: %v}",
		*config.Lat,
		*config.Lng,
		config.UseGeoclue,
		config.DBusServer,
		config.Portal,
	)
}
