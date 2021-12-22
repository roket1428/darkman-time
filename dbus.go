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
	C    chan Mode
}

func (handle *ServerHandle) changeMode(newMode string) {
	if handle.conn == nil {
		if err := handle.start(); err != nil {
			log.Printf("Could not start D-Bus server: %v", err)
			return
		}
	}

	handle.mode = newMode
	err := handle.conn.Emit("/nl/whynothugo/darkman", "nl.whynothugo.darkman.ModeChanged", newMode)
	handle.prop.SetMust("nl.whynothugo.darkman", "Mode", newMode)

	if err != nil {
		log.Printf("couldn't emit signal: %v", err)
	}
}

func (handle *ServerHandle) handleChangeMode(c *prop.Change) *dbus.Error {
	newMode := Mode(c.Value.(string))
	if newMode != DARK && newMode != LIGHT {
		log.Printf("Mode %s is invalid", newMode)
		return prop.ErrInvalidArg
	}

	handle.mode = c.Value.(string)
	err := handle.conn.Emit("/nl/whynothugo/darkman", "nl.whynothugo.darkman.ModeChanged", handle.mode)
	if err != nil {
		log.Printf("Couldn't emit signal")
		return nil
	}
	RunScripts(newMode)
	return nil
}

func (handle *ServerHandle) Close() error {
	return handle.conn.Close()
}

func NewDbusServer() ServerHandle {
	handle := ServerHandle{
		C: make(chan Mode),
	}

	go func() {
		mode := <-handle.C
		handle.changeMode(string(mode))
	}()

	// TODO: When implement new tri-state API, start at this point.

	return handle
}

func (handle *ServerHandle) start() (err error) {
	handle.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("could not connect to session D-Bus: %v", err)
	}

	propsSpec := map[string]map[string]*prop.Prop{
		"nl.whynothugo.darkman": {
			"Mode": {
				Value:    handle.mode,
				Writable: true,
				Emit:     prop.EmitTrue,
				Callback: handle.handleChangeMode,
			},
		},
	}
	handle.prop, err = prop.Export(handle.conn, "/nl/whynothugo/darkman", propsSpec)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus prop: %v", err)
	}

	err = handle.conn.Export(handle, "/nl/whynothugo/darkman", "nl.whynothugo.darkman")
	if err != nil {
		return fmt.Errorf("failed to export interface: %v", err)
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
		return fmt.Errorf("failed to export dbus name: %v", err)
	}

	reply, err := handle.conn.RequestName("nl.whynothugo.darkman", dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to register dbus name: %v", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("can't register D-Bus name: name already taken")
	}

	log.Println("Listening on D-Bus `nl.whynothugo.darkman`...")
	return nil
}
