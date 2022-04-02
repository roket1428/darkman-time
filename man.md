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
configuration file or a environment variables.

# CONFIGURATION

darkman requires no configuration, but you may, optionally, provide your
geolocation.

Configuration is read from _~/.config/darkman/config.yaml_, and takes the
format of:

```
lat: 52.3
lng: 4.8
dbusserver: true
```

You generally don't need more than one decimal point for your location. See
https://xkcd.com/2170/ for details.

The `dbusserver` setting defines whether darkman should expose the current
mode via its D-Bus API or not.

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
