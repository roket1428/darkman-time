// Package implementing a wrapper around darkman's D-Bus API.
//
// It may be used by client applications needing to query or change the current
// mode.
package libdarkman

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const prop = "nl.whynothugo.darkman.Mode"

func getDBusObj() (*dbus.BusObject, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("could not connect to d-bus: %v", err)
	}
	// FIXME: does the connection get leaked on long-running processes?

	obj := conn.Object("nl.whynothugo.darkman", "/nl/whynothugo/darkman")

	return &obj, nil
}

func validateMode(mode string) error {
	if mode != "light" && mode != "dark" {
		return fmt.Errorf("%s is not a valid mode", mode)
	}
	return nil
}

// Set the current mode. Mode MUST be either "light" or "dark".
func SetMode(mode string) error {
	if err := validateMode(mode); err != nil {
		return err
	}

	obj, err := getDBusObj()
	if err != nil {
		return err
	}

	if err = (*obj).SetProperty(prop, dbus.MakeVariant(mode)); err != nil {
		return fmt.Errorf("error setting property: %v", err)
	}

	return nil
}

// Returns the current mode, either "light" or "dark".
func GetMode() (string, error) {
	var mode string

	obj, err := getDBusObj()
	if err != nil {
		return "", err
	}

	if err = (*obj).StoreProperty(prop, &mode); err != nil {
		return "", fmt.Errorf("error reading property: %v", err)
	}

	return mode, nil
}

// Toggle the current mode (e.g.: switch to light mode if the current mode is
// dark mode or viceversa).
// Returns the current mode, either "light" or "dark".
func ToggleMode() (string, error) {
	var mode string

	obj, err := getDBusObj()
	if err != nil {
		return "", err
	}

	if err = (*obj).StoreProperty(prop, &mode); err != nil {
		return "", fmt.Errorf("error reading property: %v", err)
	}

	if mode == "light" {
		mode = "dark"
	} else {
		mode = "light"
	}

	if err = (*obj).SetProperty(prop, dbus.MakeVariant(mode)); err != nil {
		return "", fmt.Errorf("error setting property: %v", err)
	}

	return mode, nil
}
