darkman
=======

A framework for dark-mode and light-mode transitions on Linux desktop.

## Introduction

`darkman` runs in the background and turns on night mode at sundown, and turns it off
again at sunrise. `darkman` is not designed to be used interactively: it's designed to
be set up once, and run in the background.

`darkman` will use `geoclue` to determine your location and calculate sundown and
sunrise automatically.

At sundown, it will look for scripts in `$XDG_DATA_DIRS/dark-mode.d/`.
At sunrise, it will look for scripts in `$XDG_DATA_DIRS/light-mode.d/`.

These scripts individually configure different components and applications. Given the
lack of normalised "dark-mode" APIs on Linux desktop, it's likely that scripts for
different applications and toolkits have to be dumped in.

This project seeks to be a source for such scripts too.

Hint: `$XDG_DATA_DIRS` usually matches these, amongst others:

    ~/.local/share/
    /usr/local/share/
    /usr/share/

The variety here allows packages to include their own drop-in scripts. The order of
precedence is also important, so you can mask scripts.

## Installation

### ArchLinux

    paru -S darkman

### Others

    git clone git@gitlab.com:WhyNotHugo/darkman.git
    cd darkman
    make
    sudo make install

## Setup

You can run the service any way you prefer. The recommended technique is using
systemd:

    systemctl --user enable --now darkman.service

Note that the dark-mode and light-mode scripts mentioned above are not included in this
package. You'll need to drop-in scripts you desire.

## How it works

When it starts, darkman tries to determine your current location:

- The config file (no yet implemented).
- The cache file from last time it ran.
- Using the system [`geoclue`](https://directory.fsf.org/wiki/Geoclue).

Based on your location, darkman will determine sunrise/sundown. It will then
switch to dark mode or light mode accordingly.

Finally, it'll set a timer for the next sundown / sunrise (whichever comes
first), to switch to the opposite mode, set another timer, and sleep again.

It's designed to run as a service and require as little intervention
as possible.

## Configuration

`darkman` requires no configuration, but you may, optionally, provide your
geolocation (this is also useful if you don't want to or can't run geoclue).

Configuration is read from `~/.config/darkman/config.yaml`, and takes the
format of:

```yaml
lat: 52.3
lng: 4.8
```

You may also use environment variables to set a location:

```
DARKMAN_LAT=52.3
DARKMAN_LNG=4.8
```

Environment variables take precedence over the configuration file.

You generally don't need more than one decimal point for your location. See
[this article](https://xkcd.com/2170/) for details.

### D-Bus service

A D-Bus endpoint is also exposed. There's a property to determine the current
mode (`Mode`), and a signal to listen to changes (`ModeChanged`). Third-party
applications can use this to determine whether they should render light mode or
dark mode.

See [dbus-api.xml](dbus-api.xml) for an XML description/introspection of the
service.

## Development

`darkman` already works, but is still under development.

For bug and suggestions, see [Issues][issues] on GitLab. Research is also
gathered into these issues.

Feel free to join the IRC channel: #whynothugo on irc.libera.chat.

[issues]: https://gitlab.com/WhyNotHugo/darkman/-/issues

## LICENCE

darkman is licensed under the ISC licence. See LICENCE for details.
