// Package geoclue implements a client for Geoclue's D-Bus.
package geoclue

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/godbus/dbus/v5"
)

// A handle for a client connection to Geoclue.
type Geoclient struct {
	Id           string
	Locations    chan Location
	conn         *dbus.Conn
	clientPath   dbus.ObjectPath
	timeoutTimer *time.Timer
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"Alt"`
}

func (client *Geoclient) getUpdatedLocation(path dbus.ObjectPath) (*Location, error) {
	location := &Location{}
	obj := client.conn.Object("org.freedesktop.GeoClue2", path)

	if err := obj.StoreProperty("org.freedesktop.GeoClue2.Location.Latitude", &location.Lat); err != nil {
		return nil, fmt.Errorf("error reading Latitutde: %v", err)
	}

	if err := obj.StoreProperty("org.freedesktop.GeoClue2.Location.Longitude", &location.Lng); err != nil {
		return nil, fmt.Errorf("error reading Longitude: %v", err)
	}

	if err := obj.StoreProperty("org.freedesktop.GeoClue2.Location.Altitude", &location.Alt); err != nil {
		return nil, fmt.Errorf("error reading Altitude: %v", err)
	}

	return location, nil
}

func (client *Geoclient) listerForLocation(ctx context.Context) error {
	if err := client.conn.AddMatchSignal(
		dbus.WithMatchObjectPath(client.clientPath),
		dbus.WithMatchInterface("org.freedesktop.GeoClue2.Client"),
		// Type: signal...?
	); err != nil {
		return fmt.Errorf("error listening for signal: %v", err)
	}

	// TODO: Should handle the connection to geoclue dying.

	go func() {
		c2 := make(chan *dbus.Signal, 3)
		client.conn.Signal(c2)
	loop:
		for {
			select {
			case <-ctx.Done():
				client.conn.Close()
				break loop
			case s := <-c2:
				if s.Name != "org.freedesktop.GeoClue2.Client.LocationUpdated" {
					log.Println("geoclue: Got an unrelated event? ", s)
					continue
				}

				client.timeoutTimer.Stop()

				// Geoclue gives us the path to a new object that has
				// the location data, hence, "newPath".
				newPath, ok := s.Body[1].(dbus.ObjectPath)
				if !ok {
					log.Println("geoclue: failed to parse signal location: ", ok)
					continue
				}

				location, err := client.getUpdatedLocation(newPath)
				if err != nil {
					log.Println("geoclue: failed to obtain updated location: ", err)
					continue
				}

				log.Println("geoclue: resolved a new location: ", location)
				client.Locations <- *location
			}
		}
	}()

	return nil
}

// Initialise a new geoclue client.
//
// The desktopId parameter is passed onto geoclue. It should match the calling
// application's desktop file id (the basename of the desktop file), and is
// used for authorization to work.
//
// If geoclue does not return any location within the specified timeout, a
// warning is emmited.
//
// For details on DistanceThreshold and TimeThreshold see the documentation for
// geoclue's D-Bus API:
//
//   - https://www.freedesktop.org/software/geoclue/docs/gdbus-org.freedesktop.GeoClue2.Client.html#gdbus-property-org-freedesktop-GeoClue2-Client.DistanceThreshold
//   - https://www.freedesktop.org/software/geoclue/docs/gdbus-org.freedesktop.GeoClue2.Client.html#gdbus-property-org-freedesktop-GeoClue2-Client.TimeThreshold
func NewClient(ctx context.Context, desktopId string, timeout time.Duration, distanceThreshold uint32, timeThreshold uint32) (*Geoclient, error) {
	// TODO: take other params passed to the server here.
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}

	obj := conn.Object("org.freedesktop.GeoClue2", "/org/freedesktop/GeoClue2/Manager")

	var clientPath dbus.ObjectPath
	if err = obj.Call(
		"org.freedesktop.GeoClue2.Manager.GetClient",
		0,
	).Store(&clientPath); err != nil {
		return nil, fmt.Errorf("GetClient failed: %v", err)
	}

	obj = conn.Object("org.freedesktop.GeoClue2", clientPath)
	if err = obj.SetProperty(
		"org.freedesktop.GeoClue2.Client.DesktopId",
		dbus.MakeVariant(desktopId),
	); err != nil {
		return nil, fmt.Errorf("setting DesktopId failed: %v", err)
	}

	obj = conn.Object("org.freedesktop.GeoClue2", clientPath)
	if err = obj.SetProperty(
		"org.freedesktop.GeoClue2.Client.DistanceThreshold",
		dbus.MakeVariant(distanceThreshold),
	); err != nil {
		return nil, fmt.Errorf("setting DistanceThreshold failed: %v", err)
	}

	obj = conn.Object("org.freedesktop.GeoClue2", clientPath)
	if err = obj.SetProperty("org.freedesktop.GeoClue2.Client.TimeThreshold", dbus.MakeVariant(timeThreshold)); err != nil {
		return nil, fmt.Errorf("setting TimeThreshold failed: %v", err)
	}

	client := &Geoclient{
		Id:           desktopId,
		Locations:    make(chan Location, 3),
		clientPath:   clientPath,
		conn:         conn,
		timeoutTimer: time.NewTimer(timeout),
	}

	// FIXME: This goroutine is never cleaned up.
	go func() {
		<-client.timeoutTimer.C
		log.Println("geoclue: WARNING! the server hasn't responded; is it working? Timeout is:", timeout)
	}()

	if err = client.listerForLocation(ctx); err != nil {
		return nil, err
	}

	obj = client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	if err = obj.Call("org.freedesktop.GeoClue2.Client.Start", 0).Err; err != nil {
		return nil, err
	}

	log.Println("geoclue: client started.")
	return client, nil
}

// Stop searching for a location.
// This function is safe to call even if the client is not currently running.
func (client Geoclient) StopClient() error {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err := obj.Call("org.freedesktop.GeoClue2.Client.Stop", 0).Err

	if err == nil {
		log.Println("geoclue: client stopped.")
	}

	client.timeoutTimer.Stop()

	return err
}

// Check if the client is currently active or inactive.
func (client Geoclient) IsActive() (bool, error) {
	var active bool

	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	if err := obj.StoreProperty("org.freedesktop.GeoClue2.Client.Active", &active); err != nil {
		return false, fmt.Errorf("error reading Active prop: %v", err)
	}

	return active, nil
}
