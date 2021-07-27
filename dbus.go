package main

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

type ServerHandle struct {
	conn *dbus.Conn
	mode string
	prop *prop.Properties
}

func (handle *ServerHandle) ChangeMode(newMode string) error {
	handle.mode = newMode
	err := handle.conn.Emit("/nl/whynothugo/darkman", "nl.whynothugo.darkman.ModeChanged", newMode)
	handle.prop.SetMust("nl.whynothugo.darkman", "Mode", newMode)

	if err != nil {
		return fmt.Errorf("couldn't emit signal: %v", err)
	}

	return nil
}

func (handle *ServerHandle) Close() error {
	return handle.conn.Close()
}

func RunDbusServer(initialMode string) (*ServerHandle, error) {
	handle := ServerHandle{mode: initialMode}

	var err error
	handle.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("could not connect to session D-Bus: %v", err)
	}

	propsSpec := map[string]map[string]*prop.Prop{
		"nl.whynothugo.darkman": {
			"Mode": {
				Value:    initialMode,
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
	handle.prop, err = prop.Export(handle.conn, "/nl/whynothugo/darkman", propsSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to export D-Bus prop: %v", err)
	}

	err = handle.conn.Export(handle, "/nl/whynothugo/darkman", "nl.whynothugo.darkman")
	if err != nil {
		return nil, fmt.Errorf("failed to export interface: %v", err)
	}

	signal := introspect.Signal{
		Name: "ModeChanged",
		Args: []introspect.Arg{
			{
				Name: "NewMode",
				Type: "s",
			},
		},
	}

	n := &introspect.Node{
		Name: "/nl/whynothugo/darkman",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			{
				Name:       "nl.whynothugo.darkman",
				Signals:    []introspect.Signal{signal},
				Properties: handle.prop.Introspection("nl.whynothugo.darkman"),
			},
		},
	}

	err = handle.conn.Export(
		introspect.NewIntrospectable(n),
		"/nl/whynothugo/darkman",
		"org.freedesktop.DBus.Introspectable",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to export dbus name: %v", err)
	}

	reply, err := handle.conn.RequestName("nl.whynothugo.darkman", dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, fmt.Errorf("failed to register dbus name: %v", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf("can't register D-Bus name: name already taken")
	}

	return &handle, nil
}

func initDbusServer() {
	var err error
	dbusServer, err = RunDbusServer(string(currentMode))
	if err != nil {
		log.Printf("Failed to start D-Bus server: %v.\n", err)
	} else {
		log.Println("Listening on D-Bus `nl.whynothugo.darkman`...")
	}
}
