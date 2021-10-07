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
