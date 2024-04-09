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

It is also possible to trigger manual transitions and it is also possible to
disable automatic transitions entirely.

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

The open source desktop ecosystem is quite heterogeneous and making different
applications switch between dark/light different mechanism and techniques.

Darkman seeks to implement the more widely adopted standards, while leaving room
for users to hook in custom scripts for other applications.

## Custom executables

For any sort of custom integration, executables (including simple shell scripts)
can be placed in the following two directories:

- _$XDG_DATA_DIRS/dark-mode.d/_: Executed when switching to dark mode.
- _$XDG_DATA_DIRS/light-mode.d/_: Executed when switching to light mode.

These scripts or executables can perform any actions required, like telling
re-writing configuration files for a PDF reader, or controlling a notification
daemon to switch to another theme.

Scripts need to have an executable bit set, or will not be executed.

The variable `$XDG_DATA_DIRS` is defined in the xdg basedir specification, and
usually matches the following, amongst others:

- ~/.local/share/
- /usr/local/share/
- /usr/share/

Example scripts (and discussion on how to integrate different applications) are
available in the project repository:

	https://gitlab.com/WhyNotHugo/darkman

Packages may also drop-in their own scripts into any of these locations,
although application developers are encouraged to use the D-Bus API to
determine the current mode and listen for changes (see below for details).

## XDG Settings portal

Darkman implements the XDG desktop portal's dark mode standard. Applications
using this API should switch to dark/light mode based on darkman's current
preference. This standard was originally pushed by the GNOME and Elementary
teams, and is currently supported by KDE, Firefox and many other projects. You
should expect applications from those environment to support it, amongst others.

For more details on this protocol, see:

	https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.Settings.html

As for *xdg-desktop-portal* version 1.17.0, portals MUST be configured with
per-user configuration *portals.conf(5)*. To force the usage of darkman for
dark/light mode setting, use something like the following:

```
[preferred]
org.freedesktop.impl.portal.Settings=darkman
```

When using a desktop-specific configuration (e.g.: *swaywm-portals.conf*),
please keep in mind that the environment variable *XDG_CURRENT_DESKTOP* must be
set for the *xdg-desktop-portal*.

The *xdg-desktop-portal* should start after *darkman* has started and is ready.
Use *--ready-fd* for readiness notification. This is likely not relevant on
systemd-based setups, where the service manager intermediates in taking the
named bus.

For a more in-depth explanation, see this article:

	https://whynothugo.nl/journal/2024/04/09/darkman-portal-configuration/

## D-Bus API

For custom integrations, darkman exposes a D-Bus API which allows querying and
controlling the current mode. The *get*, *set* and *toggle* commands all use
this API. Usage of this API is also the recommended approach when writing custom
tools (e.g.: switching the current mode based on the input from a light sensor).

## Third party integrations

For Emacs users, a third party package exists to integrate darkman with Emacs:

	https://github.com/grtcdr/darkman.el

There also exists a plugin for neovim users:

	https://github.com/4e554c4c/darkman.nvim

# LOCATION

The current location may be specified in the configuration file. The location is
used to calculate what time sundown and sunrise happen.

It is also possible for darkman to automatically determine the system's location
using *geoclue*. Geoclue's reliability varies depending on distribution and
desktop environment, as an agent often needs to be configured for it to work
properly.

If no location is known, automatic transitions are disabled.

# CONFIGURATION

A configuration file and all settings are optional. Configuration is read from
*~/.config/darkman/config.yaml* (or other paths defined in the XDG basedir
spec), and has the following format:

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

_XDG_CURRENT_DESKTOP_
	Darkman does not use this variable; it should be defined for the
	*xdg-desktop-portal* instead.

# PRIVACY

Darkman will trigger a darkmode/lightmode transition at sundown in the current
location. Any application that is running locally can record or transmit the
time of these transitions and attempt to extrapolate information related the
current location.

When a web browser applies this transition at the same time, open websites can
record this information too.

A potential stalker or tracker can use the above information to infer that you
are likely in a region of the world where sunset happened at a specific time.
This region is usually a wide area spanning tens of thousands of kilometers, but
can be smaller for certain geographical locations.

The author of this tool uses a manually configured location with an integer
latitude and longitude to achieve a sensible balance between privacy and
convenience.

# DEVELOPMENT

For issues and general development inquiries, see the project home currently
hosted at GitLab:

	https://gitlab.com/WhyNotHugo/darkman

# DEBUGGING

To confirm which value is relayed via the *xdg-desktop-portal* use:

```
gdbus call --session \\
    --dest org.freedesktop.portal.Desktop \\
    --object-path /org/freedesktop/portal/desktop \\
    --method org.freedesktop.portal.Settings.ReadOne \\
    org.freedesktop.appearance color-scheme
```

# SEE ALSO

portals.conf(5)
gammastep(1)

# AUTHORS

Developed by Hugo O. Barrera <hugo@whynothugo.nl>, with invaluable contributions
from the community.

darkman is an open source project licensed under the ISC licence and developed
for anyone to use freely. If you would like to sponsor this project, see:

	https://whynothugo.nl/sponsor
