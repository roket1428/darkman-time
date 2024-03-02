package darkman

import (
	"context"
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

type DBusHandle struct {
	conn             *dbus.Conn
	mode             string
	prop             *prop.Properties
	c                chan Mode
	onChangeCallback func(Mode)
}

func (handle *DBusHandle) emitChangeSignal() error {
	return handle.conn.Emit("/nl/whynothugo/darkman", "nl.whynothugo.darkman.ModeChanged", handle.mode)
}

// Changes the current mode to `Mode`. This function is to be called when the
// mode is changed by another / subsystem.
func (handle *DBusHandle) ChangeMode(newMode Mode) error {
	if handle.conn == nil {
		return fmt.Errorf("cannot emit dbus signal; no connection to dbus")
	}

	handle.mode = string(newMode)
	handle.prop.SetMust("nl.whynothugo.darkman", "Mode", handle.mode)
	if err := handle.emitChangeSignal(); err != nil {
		return fmt.Errorf("error emitting mode change dbus signal: %v", err)
	}

	return nil
}

// Called when the mode is changed by writing to the D-Bus prop.
func (handle *DBusHandle) handleChangeMode(c *prop.Change) *dbus.Error {
	newMode := Mode(c.Value.(string))
	if newMode != DARK && newMode != LIGHT {
		log.Printf("Mode %s is invalid", newMode)
		return prop.ErrInvalidArg
	}

	handle.mode = c.Value.(string)
	handle.onChangeCallback(newMode)

	if err := handle.emitChangeSignal(); err != nil {
		fmt.Println("Error emitting mode change dbus signal:", err)
	}
	return nil
}

// Create a new D-Bus server instance for darkman's bespoke API.
//
// Takes as parameter a function that will be called each time the current
// mode is changed via this D-Bus API.
//
// ChangeMode must be called on the returned handle each time that the current
// mode changes by some other mechanism.
func NewDbusServer(ctx context.Context, initial Mode, onChange func(Mode)) (*DBusHandle, error) {
	handle := DBusHandle{
		c:                make(chan Mode),
		onChangeCallback: onChange,
		mode:             string(initial),
	}

	if err := handle.start(ctx); err != nil {
		return nil, fmt.Errorf("could not start D-Bus server: %v", err)
	}

	return &handle, nil
}

func (handle *DBusHandle) start(ctx context.Context) (err error) {
	opts := dbus.WithContext(ctx)
	handle.conn, err = dbus.ConnectSessionBus(opts)
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
