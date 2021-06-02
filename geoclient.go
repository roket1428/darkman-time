package main

import (
	"github.com/godbus/dbus/v5"
	"log"
)

type Geoclient struct {
	Id         string
	conn       *dbus.Conn
	c          chan Location
	clientPath dbus.ObjectPath
}

func (client *Geoclient) getUpdatedLocation(path dbus.ObjectPath) (location Location, err error) {
	obj := client.conn.Object("org.freedesktop.GeoClue2", path)
	log.Println(path)

	err = obj.StoreProperty("org.freedesktop.GeoClue2.Location.Latitude", &location.Lat)
	if err != nil {
		return
	}

	err = obj.StoreProperty("org.freedesktop.GeoClue2.Location.Longitude", &location.Lng)
	if err != nil {
		return
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
		return err
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
				log.Println("Failed to parse signal location", ok)
				continue
			}

			location, err := client.getUpdatedLocation(newPath)
			if err != nil {
				log.Println("Failed to obtain updated location", err)
				continue
			}

			log.Println("Resolved a new location: ", location)
			c <- location
		}
	}()

	return nil
}

func NewClient(id string, c chan Location) (client *Geoclient, err error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}

	obj := conn.Object("org.freedesktop.GeoClue2", "/org/freedesktop/GeoClue2/Manager")

	var clientPath dbus.ObjectPath
	err = obj.Call("org.freedesktop.GeoClue2.Manager.GetClient", 0).Store(&clientPath)
	if err != nil {
		return
	}

	obj = conn.Object("org.freedesktop.GeoClue2", clientPath)
	err = obj.SetProperty("org.freedesktop.GeoClue2.Client.DesktopId", dbus.MakeVariant("darkman"))
	if err != nil {
		return
	}

	client = &Geoclient{
		Id:         id,
		conn:       conn,
		c:          c,
		clientPath: clientPath,
	}

	err = client.listerForLocation(c)
	if err != nil {
		return
	}

	return
}

func (client Geoclient) StartClient() error {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err := obj.Call("org.freedesktop.GeoClue2.Client.Start", 0).Err

	if err != nil {
		log.Println("Client started.")
	}

	return err
}

func (client Geoclient) StopClient() error {
	obj := client.conn.Object("org.freedesktop.GeoClue2", client.clientPath)
	err := obj.Call("org.freedesktop.GeoClue2.Client.Stop", 0).Err

	if err != nil {
		log.Println("Client stopped.")
	}

	return err
}
