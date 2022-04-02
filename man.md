darkman(1)

# NAME

darkman - a framework for dark-mode and light-mode transitions on Linux desktop

# SYNOPSIS

darkman

# DESCRIPTION

*darkman* runs in the background and turns on night mode at sundown, and turns it off
again at sunrise. darkman is not designed to be used interactively: it's designed to
be set up once, and run in the background.

- At sundown, it will look for scripts in _$XDG_DATA_DIRS/dark-mode.d/_.
- At sunrise, it will look for scripts in _$XDG_DATA_DIRS/light-mode.d/_.

For some sample scripts for common applications and environments, see
https://gitlab.com/WhyNotHugo/darkman

Darkman also implements the Freedesktop dark mode API. Applications using this
API should switch to dark/light mode based on darkman's current preference.

It is also possible disable manual transitions and control darkman manually.

# COMMANDS

*run*
	Runs the darman service. This command is intended to be executed by a
	service manager, init script or alike.

*set* <light|dark>
	Sets the current mode.

*get*
	Prints the current mode.

*toggle*
	Toggle the current mode.

# LOCATION

darkman will automatically determine your location using *geoclue*. If using
geoclue is not an option, the location may be specific explicitly via a
configuration file.

# CONFIGURATION

darkman requires no configuration. A configuration file and all settings are
optional.

Configuration is read from _~/.config/darkman/config.yaml_, and has the
following format:

```
lat: 52.3
lng: 4.8
dbusserver: true
```

The following settings are available:

- *lat*, *lng*: Latitude and longitude respectively. This value will be used at
  start-up, but will later be superseded by whatever geoclue resolves (if
  enabled). You generally don't need more than one decimal point for your
  location, as describen in https://xkcd.com/2170/.

- *dbusserver* (true/false): whether to expose the current mode via darkman's
  own D-Bus API. The command line tool uses this API to apply changes, so it
  will not work if this setting is disabled.

- *portal* (true/false): whether to expose the current mode via the XDG settings
  portal D-Bus API. Many desktop application will read the current mode via the
  portal and respect what darkman is indicating.

- *usegeoclue* (true/false): whether to use a local geoclue instance to
  determine the current location. On some distributions/setups, this may
  require setting up a geoclue agent to function properly.

# ENVIRONMENT

The following environment variables are also read and will override the
configuration file:

_DARKMAN_LAT_
	Overrides the latitude for the current location.

_DARKMAN_LNG_
	Overrides the longitude for the current location.

_DARKMAN_DBUSSERVER_
	Overrides whether to expose the current mode via D-Bus.

# LICENCE

darkman is licensed under the ISC licence. See LICENCE for details.
