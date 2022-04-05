package darkman

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const PORTAL_NAMESPACE = "org.freedesktop.appearance"
const PORTAL_KEY = "color-scheme"
const PORTAL_BUS_NAME = "org.freedesktop.impl.portal.desktop.darkman"
const PORTAL_OBJ_PATH = "/org/freedesktop/portal/desktop"
const PORTAL_INTERFACE = "org.freedesktop.impl.portal.Settings"

type PortalHandle struct {
	conn *dbus.Conn
	mode uint
}

func modeToPortalValue(mode Mode) uint {
	switch mode {
	case NULL:
		return 0
	case DARK:
		return 1
	case LIGHT:
		return 2
	}

	// Should never happen: it's a fatal programming errror.
	log.Println("Got an invalid mode to convert to a D-Bus value!!")
	return 255
}

func (portal *PortalHandle) changeMode(newMode Mode) {
	if portal.conn == nil {
		if err := portal.start(); err != nil {
			log.Printf("Could not start D-Bus server: %v", err)
			return
		}
	}

	portal.mode = modeToPortalValue(newMode)
	err := portal.conn.Emit(
		PORTAL_OBJ_PATH,
		"org.freedesktop.portal.Settings.SettingChanged",
		PORTAL_NAMESPACE,
		PORTAL_KEY,
		dbus.MakeVariant(portal.mode),
	)
	if err != nil {
		log.Printf("couldn't emit signal: %v", err)
	}
}

/// Create a new D-Bus server instance for the XDG portal API.
///
/// Returns a callback function which should be called each time the current
/// mode changes.
func NewPortal() (PortalHandle, func(Mode)) {
	portal := PortalHandle{}

	return portal, portal.changeMode
}

func (portal *PortalHandle) start() (err error) {
	portal.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("could not connect to session D-Bus: %v", err)
	}

	// Define the "Version" prop (its value will be static).
	propsSpec := map[string]map[string]*prop.Prop{
		"org.freedesktop.impl.portal.Settings": {
			"Version": {
				Value:    uint(1),
				Writable: false,
				Emit:     prop.EmitTrue,
			},
		},
	}
	// Export the "Version" prop.
	versionProp, err := prop.Export(portal.conn, PORTAL_OBJ_PATH, propsSpec)
	if err != nil {
		return fmt.Errorf("failed to export D-Bus prop: %v", err)
	}

	// Exoprt the D-Bus object.
	err = portal.conn.Export(portal, PORTAL_OBJ_PATH, PORTAL_INTERFACE)
	if err != nil {
		return fmt.Errorf("failed to export interface: %v", err)
	}

	// Declare change signal (for introspection only).
	settingChanged := introspect.Signal{
		Name: "SettingChanged",
		Args: []introspect.Arg{
			{
				Name: "namespace",
				Type: "s",
			},
			{
				Name: "key",
				Type: "s",
			},
			{
				Name: "value",
				Type: "v",
			},
		},
	}

	// Declare read method (for introspection only).
	readMethod := introspect.Method{
		Name: "Read",
		Args: []introspect.Arg{
			{
				Name:      "namespace",
				Type:      "s",
				Direction: "in",
			},
			{
				Name:      "key",
				Type:      "s",
				Direction: "in",
			},
			{
				Name:      "value",
				Type:      "v",
				Direction: "out",
			},
		},
	}
	readAllMethod := introspect.Method{
		Name: "ReadAll",
		Args: []introspect.Arg{
			{
				Name:      "namespaces",
				Type:      "as",
				Direction: "in",
			},
			{
				Name:      "value",
				Type:      "a{sa{sv}}",
				Direction: "out",
			},
		},
	}

	portalInterface := introspect.Interface{
		Name:       PORTAL_INTERFACE,
		Signals:    []introspect.Signal{settingChanged},
		Properties: versionProp.Introspection(PORTAL_INTERFACE),
		Methods:    []introspect.Method{readMethod, readAllMethod},
	}

	n := &introspect.Node{
		Name: PORTAL_OBJ_PATH,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			portalInterface,
		},
	}

	err = portal.conn.Export(
		introspect.NewIntrospectable(n),
		PORTAL_OBJ_PATH,
		"org.freedesktop.DBus.Introspectable",
	)
	if err != nil {
		return fmt.Errorf("failed to export dbus name: %v", err)
	}

	reply, err := portal.conn.RequestName(PORTAL_BUS_NAME, dbus.NameFlagDoNotQueue)
	if err != nil {
		return fmt.Errorf("failed to register dbus name: %v", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("can't register D-Bus name: name already taken")
	}

	log.Println("Listening on D-Bus:", PORTAL_BUS_NAME)
	return nil
}

func (portal *PortalHandle) Read(namespace string, key string) (dbus.Variant, *dbus.Error) {
	if namespace != PORTAL_NAMESPACE || key != PORTAL_KEY {
		log.Println("Got request for unknown setting:", namespace, key)
		return dbus.Variant{}, dbus.NewError("org.freedesktop.portal.Error.NotFound", []interface{}{"Requested setting not found"})
	}

	return dbus.MakeVariant(portal.mode), nil
}

func (portal *PortalHandle) ReadAll(namespaces []string) (map[string]map[string]dbus.Variant, *dbus.Error) {
	values := map[string]map[string]dbus.Variant{}

	for _, namespace := range namespaces {
		if namespace == PORTAL_NAMESPACE {
			values[PORTAL_NAMESPACE] = map[string]dbus.Variant{
				PORTAL_KEY: dbus.MakeVariant(portal.mode),
			}
		}
	}

	return values, nil
}
