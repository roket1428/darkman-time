darkman(1)

# NAME

darkman - daemon for dark-mode and light-mode transitions on Unix-like desktops

# SYNOPSIS

*darkman* _run_++
*darkman* _set_ [_light_|_dark_]++
*darkman* _get_++
*darkman* _toggle_

# DESCRIPTION

*darkman* runs in the background and turns on dark mode at sundown, and turns it off
again at sunrise. darkman is not designed to be used interactively: it's designed to
be set up once, and run in the background.

It is, however, possible to trigger manual transitions and even disable
automatic transitions entirely.

# COMMANDS

*run*
	Runs the darkman service. This command is intended to be executed by a
	service manager, init script or alike.

*set* <light|dark>
	Sets the current mode.

*get*
	Prints the current mode.

*toggle*
	Toggle the current mode.

# INTEGRATIONS

The open source desktop ecosystem is quite heterogeneous and switching between
dark/light mode on different applications requires different mechanism and
techniques.

Darkman seeks to implement the more widely supported standards, while leaving
room for users to hook in custom scripts for other applications.

Darkman implements the FreeDesktop dark mode standard. Applications using this
API should switch to dark/light mode based on darkman's current preference. This
standard was originally pushed by the GNOME and Elementary teams, and is
currently supported by KDE, Firefox and many other projects. You should expect
applications from those environment to support it, amongst others.

For additional application support, custom executables (including simple
shell scripts) can be placed in the following two directories:

- _$XDG_DATA_DIRS/dark-mode.d/_: Scripts to run when switching to dark mode.
- _$XDG_DATA_DIRS/light-mode.d/_: Scripts to run when switching to light mode.

; This is the recommended approach for simpler tools like mako(1), zathura(1), etc.

Scripts need to have an executable bit set, or will not be executed.

For additional sample scripts (and discussion on how to integrate different
applications) see the project website:

	https://gitlab.com/WhyNotHugo/darkman

For custom integrations, darkman exposes a D-Bus API which allows querying and
controlling the current mode. The *get*, *set* and *toggle* commands all use
this API. Usage of this API is also the recommended approach when writing custom
tools (e.g.: switching the current mode based on the input from a light sensor).

For Emacs users, a third party package exists to integrate darkman with Emacs:

	https://github.com/grtcdr/darkman.el

# LOCATION

darkman will automatically determine the system's location using *geoclue*.
Geoclue's reliability varies depending on distribution and desktop environment,
and it will often not work without a geoclue agent running.

If using geoclue is not an option, the location may be specified explicitly via
a configuration file.

# CONFIGURATION

A configuration file and all settings are optional. Configuration is read from
*~/.config/darkman/config.yaml*, and has the following format:

```
lat: 52.3
lng: 4.8
dbusserver: true
```

The following settings are available:

- *lat*, *lng*: Latitude and longitude respectively. This value will be used at
  start-up, but will later be superseded by whatever geoclue resolves (if
  enabled). More than one decimal point is generally not needed, as described
  in https://xkcd.com/2170/.

- *usegeoclue* (true/*false*): Whether to use a local geoclue instance to
  determine the current location. On some distributions/setups, this may
  require setting up a geoclue agent to function properly. Setting this to
  false without explicitly setting lat and lng disables automatic transitions
  entirely.

- *dbusserver* (*true*/false): Whether to expose the current mode via darkman's
  own D-Bus API. The command line tool uses this API to apply changes, so it
  will not work if this setting is disabled.

- *portal* (*true*/false): Whether to expose the current mode via the XDG settings
  portal D-Bus API. Many desktop application will read the current mode via the
  portal and respect what darkman is indicating.

# ENVIRONMENT

The following environment variables are also read and will override the
configuration file:

_DARKMAN_LAT_
	Overrides the latitude for the current location.

_DARKMAN_LNG_
	Overrides the longitude for the current location.

_DARKMAN_DBUSSERVER_
	Overrides whether to expose the current mode via D-Bus.

# PORTAL

As for *xdg-desktop-portal* version 1.17.0, portals can be configured with
per-user configuration *portals.conf(5)*. To force the usage of darkman for
dark/light mode setting, use something like the following:

```
[preferred]
org.freedesktop.impl.portal.Settings=darkman
```

# PRIVACY

Darkman will trigger a darkmode/lightmode transition at sundown in the current
location. When a web browser applies this transition at the same time, open
websites can use this information to extrapolate some information about the
current location.

# DEVELOPMENT

For issues and general development inquiries, see the project home currently
hosted at GitLab:

	https://gitlab.com/WhyNotHugo/darkman

# SEE ALSO

portals.conf(5)
gammastep(1)

# AUTHORS

Developed by Hugo O. Barrera <hugo@whynothugo.nl>, with invaluable contributions
from the community.

darkman is an open source project licensed under the ISC licence and developed
for anyone to use freely. If you would like to sponsor this project, see:

	https://whynothugo.nl/sponsor
