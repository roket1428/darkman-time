darkman
=======

A framework for dark-mode and light-mode transitions on Linux desktop.

## Introduction

`darkman` runs in the background and turns on dark mode at sundown, and turns it off
again at sunrise. `darkman` is not designed to be used interactively: it's designed to
be set up once, and run in the background.

At sundown, it will look for scripts in `$XDG_DATA_DIRS/dark-mode.d/`.
At sunrise, it will look for scripts in `$XDG_DATA_DIRS/light-mode.d/`.

These scripts individually configure different components and applications.

Sample and reference scripts are included in the `examples` directory, and
further contributions for specific applications or environments are welcome.

Hint: `$XDG_DATA_DIRS` usually matches these, amongst others:

    ~/.local/share/
    /usr/local/share/
    /usr/share/

Packages may also drop-in their own scripts into any of these locations,
although application developers are encouraged to use the D-Bus API to
determine the current mode and listen for changes (see below for details).

## Installation

### ArchLinux

    paru -S darkman

### Fedora

    dnf install darkman

### Others

You'll need `scdoc` and `go` installed to build from source. There are
typically both available in distribution repositories (e.g.: `apt-get
install...`).

    git clone git@gitlab.com:WhyNotHugo/darkman.git
    cd darkman
    make
    sudo make install

## Setup

You can run the service any way you prefer. If you use systemd, a service file
is included:

    systemctl --user enable --now darkman.service

`darkman` **DOES NOT** depend on systemd, and is usable on non-systemd
distributions.

Note that the dark-mode and light-mode scripts mentioned above (and available
in the source repository) are not included in this package. You'll need to
drop-in the scripts you desire.

## How it works

When it starts, darkman tries to determine your current location:

- The config file.
- The cache file from last time it ran.
- Using the system [`geoclue`](https://directory.fsf.org/wiki/Geoclue).

Based on your location, darkman will determine sunrise/sundown. It will then
switch to dark mode or light mode accordingly.

Finally, it'll set a timer for the next sundown / sunrise (whichever comes
first), to switch to the opposite mode, set another timer, and sleep again.

It's designed to run as a service and require as little intervention as
possible.

It is possible to manually query or change the current mode using the
`darkman`. See `darkman --help` for details.

## Configuration

See [the man page](https://darkman.whynothugo.nl) (`man darkman`) for
configuration details.

## D-Bus service

A D-Bus endpoint is also exposed. There's a property to determine the current
mode (`Mode`), and a signal to listen to changes (`ModeChanged`). Third-party
applications can use this to determine whether they should render light mode or
dark mode.

See [dbus-api.xml](dbus-api.xml) for an XML description/introspection of the
service.

A `libdarkman` go package is available to query the same D-Bus API from other
client applications written in go. See [its documentation][libdarkman] for
details.

[libdarkman]: https://godocs.io/gitlab.com/WhyNotHugo/darkman/libdarkman

## Development

`darkman` works well and is actively maintained.

For bug and suggestions, see [Issues][issues] on GitLab. Ongoing research is
also gathered into these issues.

If you find the tool useful, please, considering [sponsoring its
development][ko-fi].

Feel free to join the IRC channel: #whynothugo on irc.libera.chat.

[issues]: https://gitlab.com/WhyNotHugo/darkman/-/issues
[ko-fi]: https://ko-fi.com/whynothugo

## LICENCE

darkman is licensed under the ISC licence. See LICENCE for details.
