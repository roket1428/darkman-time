package geoclue

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"log"
)

type Geoclient struct {
	Id         string
	Locations  chan Location
	conn       *dbus.Conn
	clientPath dbus.ObjectPath
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
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

	return
}

func (client *Geoclient) listerForLocation(c chan Location) error {
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
			// Name should always be org.freedesktop.GeoClue2.Client.LocationUpdated

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
			c <- *location
		}
	}()

	return nil
}

func NewClient(id string) (*Geoclient, error) {
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
	}

	err = client.listerForLocation(client.Locations)
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

	return err
}

func (client Geoclient) StopClient() error {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err := obj.Call("org.freedesktop.GeoClue2.Client.Stop", 0).Err

	if err == nil {
		log.Println("Client stopped.")
	}

	return err
}
