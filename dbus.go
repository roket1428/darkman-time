package darkman

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
	c    chan Mode
}

func (handle *ServerHandle) changeMode(newMode Mode) {
	if handle.conn == nil {
		if err := handle.start(); err != nil {
			log.Printf("Could not start D-Bus server: %v", err)
			return
		}
	}

	handle.mode = string(newMode)
	err := handle.conn.Emit("/nl/whynothugo/darkman", "nl.whynothugo.darkman.ModeChanged", handle.mode)
	handle.prop.SetMust("nl.whynothugo.darkman", "Mode", handle.mode)

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

/// Create a new D-Bus server instance for our API.
/// 
/// Returns a callback function which should be called each time the current
/// mode changes.
func NewDbusServer() (ServerHandle, func(Mode)) {
	handle := ServerHandle{
		c: make(chan Mode),
	}

	return handle, handle.changeMode
}

func (handle *ServerHandle) start() (err error) {
	handle.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("could not connect to session D-Bus: %v", err)
	}

	// Define the "Mode" prop.
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

	// Export the "Mode" prop.
	handle.prop, err = prop.Export(handle.conn, "/nl/whynothugo/darkman", propsSpec)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus prop: %v", err)
	}

	// Export the D-Bus object.
	err = handle.conn.Export(handle, "/nl/whynothugo/darkman", "nl.whynothugo.darkman")
	if err != nil {
		return fmt.Errorf("failed to export interface: %v", err)
	}

	// Declare our signal (for introspection only).
	modeChanged := introspect.Signal{
		Name: "ModeChanged",
		Args: []introspect.Arg{
			{
				Name: "NewMode",
				Type: "s",
			},
		},
	}

	darkmanInterface := introspect.Interface{
		Name:       "nl.whynothugo.darkman",
		Signals:    []introspect.Signal{modeChanged},
		Properties: handle.prop.Introspection("nl.whynothugo.darkman"),
	}

	// Declare our whole interface (for introspection only).
	n := &introspect.Node{
		Name: "/nl/whynothugo/darkman",
		Interfaces: []introspect.Interface{
			introspect.IntrospectData, // introspection interface
			prop.IntrospectData,       // prop read/set interface
			darkmanInterface,          // darkman interface
		},
	}

	// Export introspection data.
	err = handle.conn.Export(
		introspect.NewIntrospectable(n),
		"/nl/whynothugo/darkman",
		"org.freedesktop.DBus.Introspectable",
	)
	if err != nil {
		return fmt.Errorf("failed to export dbus name: %v", err)
	}

	// Register our D-Bus name.
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
