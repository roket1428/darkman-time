package geoclue

import (
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
	timeout      time.Duration
	timeoutTimer *time.Timer
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"Alt"`
}

func (client *Geoclient) getUpdatedLocation(path dbus.ObjectPath) (location *Location, err error) {
	location = &Location{}

	obj := client.conn.Object("org.freedesktop.GeoClue2", path)

	err = obj.StoreProperty("org.freedesktop.GeoClue2.Location.Latitude", &location.Lat)
	if err != nil {
		return nil, fmt.Errorf("error reading Latitutde: %v", err)
	}

	err = obj.StoreProperty("org.freedesktop.GeoClue2.Location.Longitude", &location.Lng)
	if err != nil {
		return nil, fmt.Errorf("error reading Longitude: %v", err)
	}

	err = obj.StoreProperty("org.freedesktop.GeoClue2.Location.Altitude", &location.Alt)
	if err != nil {
		return nil, fmt.Errorf("error reading Altitude: %v", err)
	}

	return
}

func (client *Geoclient) listerForLocation() error {
	err := client.conn.AddMatchSignal(
		dbus.WithMatchObjectPath(client.clientPath),
		dbus.WithMatchInterface("org.freedesktop.GeoClue2.Client"),
		// Type: signal...?
	)
	if err != nil {
		return fmt.Errorf("error listening for signal: %v", err)
	}

	go func() {
		c2 := make(chan *dbus.Signal, 10)
		client.conn.Signal(c2)
		for {
			s := <-c2
			if s.Name != "org.freedesktop.GeoClue2.Client.LocationUpdated" {
				log.Println("Got an unrelated event? ", s)
				continue
			}

			if client.timeoutTimer != nil {
				client.timeoutTimer.Stop()
			}

			// Geoclue gives us the path to a new object that has
			// the location data, hence, "newPath".
			newPath, ok := s.Body[1].(dbus.ObjectPath)
			if !ok {
				log.Println("Failed to parse signal location: ", ok)
				continue
			}

			location, err := client.getUpdatedLocation(newPath)
			if err != nil {
				log.Println("Failed to obtain updated location: ", err)
				continue
			}

			log.Println("Resolved a new location: ", location)
			client.Locations <- *location
		}
	}()

	return nil
}

// Initialise a new geoclue client.
// If geoclue does not return any location within timeout, a warning is
// emmited.
func NewClient(id string, timeout time.Duration) (*Geoclient, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}

	obj := conn.Object("org.freedesktop.GeoClue2", "/org/freedesktop/GeoClue2/Manager")

	var clientPath dbus.ObjectPath
	err = obj.Call("org.freedesktop.GeoClue2.Manager.GetClient", 0).Store(&clientPath)
	if err != nil {
		return nil, fmt.Errorf("GetClient failed: %v", err)
	}

	obj = conn.Object("org.freedesktop.GeoClue2", clientPath)
	err = obj.SetProperty("org.freedesktop.GeoClue2.Client.DesktopId", dbus.MakeVariant(id))
	if err != nil {
		return nil, fmt.Errorf("setting DesktopId failed: %v", err)
	}

	client := &Geoclient{
		Id:         id,
		Locations:  make(chan Location, 10),
		clientPath: clientPath,
		conn:       conn,
		timeout:    timeout,
	}

	err = client.listerForLocation()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (client Geoclient) StartClient() error {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err := obj.Call("org.freedesktop.GeoClue2.Client.Start", 0).Err

	if err == nil {
		log.Println("Client started.")
	}
	client.timeoutTimer = time.NewTimer(client.timeout)
	go func() {
		<-client.timeoutTimer.C
		log.Println("WARNING! Geoclue server hasn't responded. Is it working? Been waiting for:", client.timeout)
	}()

	return err
}

func (client Geoclient) StopClient() error {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err := obj.Call("org.freedesktop.GeoClue2.Client.Stop", 0).Err

	if err == nil {
		log.Println("Client stopped.")
	}

	if client.timeoutTimer != nil {
		client.timeoutTimer.Stop()
	}

	return err
}

// Check if the client is currently active or inactive.
func (client Geoclient) IsActive() (active bool, err error) {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err = obj.StoreProperty("org.freedesktop.GeoClue2.Client.Active", &active)
	if err != nil {
		return false, fmt.Errorf("error reading Active prop: %v", err)
	}

	return
}
