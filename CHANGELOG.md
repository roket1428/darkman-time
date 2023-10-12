darkman changelog
=================

## v0.1.0

Initial release

## v0.1.1

Include a LICENCE file.

## v0.1.2

Include a systemd service file.

## v0.1.3

Make geoclue stop polling when not needed any more.

## v0.2.0

- Redesign and rewrite.
- Cache the last known location. This makes subsequent startups faster.

## v0.3.0

Rewrite in golang. This makes building and redistributing a lot simpler, but
also allows switching to libraries which simplify exposing a D-Bus service.

## v0.4.0

Provide a D-Bus service for other applications to determine the current mode.
There's both a property (to poll the current mode) and a signal. The signal
will emit when the current mode changes and allows applications to react
immediately.

## v0.4.1

- Logging improvements.
- Don't trigger a transition (or the `ModeChanged` signal) when the mode doesn't
  need to change.
- Run all transition scripts in parallel.

## v0.5.0

- darkman is now configurable via a configuration file, or environment
  variables. See README.md for details.
- `boottimer` has been moved into a separate module. It is considered usable by
  third parties, though its design is likely non-final, and it still needs
  finer testing outside our specific use-case.

## v0.5.1

- A man page is not included.

## v0.5.2

- Add a warning if geoclue is not responding. This should help debug instances
  where darkman isn't working because it can't figure out the current location.
- boottimer: Improve precision of the timer.
- Fix negative latitudes and longitudes not working.

## v0.5.3

- Changing the current mode via the D-Bus API is now possible.
- A `darkmanctl` contrib script is now included to manually transition the
  current mode.

## v0.6.0

- A bug where the D-Bus server sometimes failed to start has been fixed.
- `darkmanctl` is now installed by default, along with shell completions for it.
- A go package is included to query and control the current mode from other go
  applications.

## v0.7.0

- `darkmanctl` has been merged into separate commands of `darkman`. This avoids
  installing a second binary, reduces installation size around half, and the
  new command is shorter to type. If you run darkman via an init script or
  alike, update it to execute `darkmanctl run` instead of just `darkman`.
- How we interact with geoclue has been simplified a bit -- we delegate more to
  geoclue rather than such continuos control over it.
- Calculation of next sundown and sunrise has been simplified, as well as
  reducing unnecessary calculations.

## 0.7.1

- Avoid scripts running concurrently (this could only happen in some very
  unusual wake-up scenarios with very specific timing).

## 1.0.0

- Support integration with the `xdg-desktop-portal`' [dark style preference
  setting][xdp]. With this integration, applications using this portal will
  also be aligned by darkman's current mode and transitions.
- Scripts are now run in sequence rather than in parallel.

[xdp]: https://github.com/flatpak/xdg-desktop-portal/issues/629

## 1.0.1

- Minor bugfixes.
- Example scripts have been moved into `examples/`.

## 1.1.0

- The systemd service now has some extra hardening security rules. If you find
  any regression with your own scripts, please open an issue.
- It is now possible to run darkman without a location and without automated
  transitions (this may be useful when controlling it via a light sensor, or
  manually). See the man page for details.

## 1.2.0

- Fix a signal not being raised when changing the value via the
  xdg-desktop-portal. This resulted in applications not immediately picking up
  the change.

## 1.3.0

- The start-up sequence has changed slightly. Previously, darkman did not listen
  on the D-Bus APIs until a mode had been set. This was problematic in
  scenarios where geoclue does not work and no location ever resolved; because
  no transition ever happened, darkman never listened on the D-Bus APIs, 
  which implied that start-up had not been successful.
  Darkman will now bind to the D-Bus APIs immediately on start-up, but only emit
  a change when an actual transition happens.
  Because of this, querying the mode during start-up _may_ return `NULL`,
  whereas previously it would simply not respond until another mode was set or
  until the query timed out.
- Disabling automatic transitions is now supported.
- Darkman will now cache the last mode to disk. If location-based transitions
  are disabled, this mode will be used at start-up.
- The documentation is now also available at https://darkman.whynothugo.nl/

## 1.3.1

- Fix how the signal for the XDG portal is raised. Previously it was raised
  directly, which would would only be picked up by specific unsandboxed
  applications, but not most of them (mostly since `darkman` raised the signal
  itself rather than via the desktop portal).

## 1.4.0

- The man page now indicates the default value for each config setting.
- When failing to register a D-Bus service (e.g.: because it is already taken),
  darkman will not exit immediately, rather than simply log the error.

## 1.5.0

- When running via systemd.service, darkman no longer depends on
  `graphical-session.target`.
- Implemented `--version`.
- Substantially trimmed down dependency tree by replacing `cobra` and `viper`
  with `flaggy`

## 1.5.1

- Fixed breakage in build scripts.

## 1.5.2

- Fixed build failures on non-64bit architectures.

## 1.5.3

- Avoid conflicting usage of LDFLAGS in build scripts.

## 1.5.4

- Fixed a bug where darkman would stall if geoclue was not running or not
  present.

## 2.0.0

- **BREAKING** Exit with an error if the configuration file has unknown fields.
- **BREAKING** Geoclue integration is now disabled by default. To retain the
  previous behaviour, explicitly enable it in the configuration file.
- **BREAKING** Go version 1.18 is not required to build.
- Don't print usage output if an error occurs. Only show it if the provided
  arguments are invalid. This was a bug, and make reading error output
  extremely unintuitive.
- Various improvements to example scripts.
- Various documentation improvements, including docs for `portals.conf`.
- Droped hardening rules for the systemd service. This doesn't realistically
  add much security in the end but interferes with several configurations.
- Switch back from `flaggy` to `cobra`. The latter has fixed the issues that we
  had in the past, and can generate shell completions.
- Shell completions are now included for `bash`, `fish` and `zsh`.
