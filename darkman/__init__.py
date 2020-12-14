from __future__ import annotations

import logging
import subprocess
from dataclasses import dataclass
from datetime import datetime
from datetime import timedelta
from enum import Enum
from pathlib import Path
from typing import Callable
from typing import Optional

from astral import Observer
from astral.sun import sun
from dateutil.tz import tzlocal
from twisted.internet import defer
from twisted.internet import reactor
from txdbus import client
from txdbus import error
from txdbus.objects import RemoteDBusObject
from xdg import BaseDirectory

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)


class Mode(Enum):
    Light = "light-mode"
    Dark = "dark-mode"

    @property
    def opposite(mode: Mode) -> Mode:
        if mode == Mode.Light:
            return Mode.Dark
        elif mode == Mode.Dark:
            return Mode.Light
        raise ValueError("Expected a Mode.")

    def activate(self):
        """Activate this mode.

        This is done by running one of more scripts in directories defined by
        convention:

        """
        logger.info("Activating %s.", self)

        scripts = {}
        for d in BaseDirectory.xdg_data_dirs:
            path = Path(d) / f"{self.value}.d"

            if path.is_dir():
                for file in (path).iterdir():
                    if file.name not in scripts:
                        scripts[file.name] = file
                        logger.info("Collected `%s.`", file)
                    else:
                        logger.info("Ignoring `%s`; it's been masked.", file)

        for name, script in scripts.items():
            proc = subprocess.run([script], shell=True)
            logger.info("Running %s returned %d.", name, proc.returncode)

        logger.info("Done switching to %s.", self)


@dataclass
class Event:
    """An event in the future in which we'll do a transition."""

    when: datetime
    mode: Mode
    scheduler: Scheduler

    def schedule(self, scheduler: Scheduler):
        """Schedules this event."""
        now = datetime.now(tzlocal())

        # max here set this to "0". This is to avoid breakage if we're scheduling during
        # the exact moment that a transition must take place.
        wait_for = max((self.when - now).total_seconds(), 0)

        logger.info("Will change to %s in %s seconds.", self.mode, wait_for)
        reactor.callLater(wait_for, self.execute, scheduler=scheduler)

    def execute(self, scheduler: Scheduler):
        """Execute this event, and schedule the next one."""
        self.mode.activate()
        scheduler.mode = self.mode

        # XXX: Should I wait here or have some offset? If the clock is skewed a few
        # seconds, this might re-schedule the same event....?
        next_event = self.gen_next(scheduler)
        next_event.schedule(scheduler=scheduler)

    @classmethod
    def gen_next(cls, scheduler: Scheduler, date=None) -> Event:
        """Return the next event."""

        local_sun = sun(scheduler.location, date=date, tzinfo=tzlocal())

        light_time = local_sun["dawn"]
        dark_time = local_sun["sunset"] + (local_sun["dusk"] - local_sun["sunset"])

        now = datetime.now(tzlocal())

        # XXX: There's an assumption made in this code that sunrise always comes before
        # sunset. I _think_ this is true anywhere in the world any time of the
        # year, though have a feeling that this might be one of these silly non-truths
        # we programmers assume somehow. 🤔

        if dark_time < now:
            # Already dark today, next change is tomorrow:
            return cls.gen_next(scheduler, now + timedelta(days=1))
        elif light_time < now < dark_time:
            return Event(dark_time, Mode.Dark, scheduler=scheduler)
        elif now < light_time:
            return Event(light_time, Mode.Light, scheduler=scheduler)
        else:
            raise Exception("Something went wrong. Please report this!")


class Scheduler:
    _location: Optional[Observer] = None
    mode = Optional[Mode]

    @property
    def location(self):
        return self._location

    def set_location(self, location):
        if location != self._location:
            self._location = location
            self._set_timer()

    def _set_timer(self) -> None:
        """Set timers for the next color scheme transition."""

        for call in reactor.getDelayedCalls():
            # Cancel previously scheduled events.
            # We've moved, so those no longer apply.
            call.cancel()

        event = Event.gen_next(self)

        # Activate the opposite now. E.g.: If the next change is a transition to
        # dark mode, then we should be in light mode now.
        event.mode.opposite.activate()

        event.schedule(scheduler=self)

    # gsettings set io.elementary.terminal.settings prefer-dark-style true


class GeoClueClient:
    """A client for geoclue2 that waits for the location and runs the callback."""

    geoclue: RemoteDBusObject

    @defer.inlineCallbacks
    def _stop_geolocation(self):
        """Tell geoclue to stop polling for geolocation updates."""
        yield self.geoclue.callRemote("Stop")
        logger.info("Geoclue client stopped")

    @defer.inlineCallbacks
    def _on_location_updated(self, old_path: str, new_path: str):
        """Work with location data to set timers.

        This function is called after GeoClue confirms our location, and sets timers to
        execute sunrise / sundown actions.
        """
        logger.info("Received location update signal from geoclue")

        # geoclue will keep on updating the location continuously all day.
        # Don't want that.
        yield self._stop_geolocation()

        location_obj = yield self.connection.getRemoteObject(
            "org.freedesktop.GeoClue2",
            new_path,
        )
        lat = yield location_obj.callRemote(
            "Get",
            "org.freedesktop.GeoClue2.Location",
            "Latitude",
        )
        lon = yield location_obj.callRemote(
            "Get",
            "org.freedesktop.GeoClue2.Location",
            "Longitude",
        )
        logger.info("Got updated location data: %f, %f", lat, lon)
        location = Observer(latitude=lat, longitude=lon)

        self.callback(location)

    @defer.inlineCallbacks
    def _create_geoclue_object(self) -> RemoteDBusObject:
        """Creates a geoclue client, and returns it.

        Clients are private and per-connection. So we need to keep the connection around
        to further communicate with the client using it.
        """

        # Ask the manager API to create a client
        manager = yield self.connection.getRemoteObject(
            "org.freedesktop.GeoClue2",
            "/org/freedesktop/GeoClue2/Manager",
        )
        client_path = yield manager.callRemote("GetClient")

        # Get the client object:
        self.geoclue = yield self.connection.getRemoteObject(
            "org.freedesktop.GeoClue2",
            client_path,
        )
        # This value needs to be set for some form of authorisation.
        # I've no idea what the _right_ value is, but this works fine.
        # Asked upstream at https://gitlab.freedesktop.org/geoclue/geoclue/-/issues/138
        yield self.geoclue.callRemote(
            "Set",
            "org.freedesktop.GeoClue2.Client",
            "DesktopId",
            "9",
        )

    @defer.inlineCallbacks
    def main(self, callback: Callable[[Observer], None]):
        """Listens to location changes."""
        self.callback = callback

        try:
            # Geoclue expects all calls to be made from the same connection:
            self.connection = yield client.connect(reactor, "system")

            yield self._create_geoclue_object()
            logger.info("Got geoclue client: %s.", self.geoclue)

            # Set a callback for location updates.
            self.geoclue.notifyOnSignal("LocationUpdated", self._on_location_updated)

            # Find our location using geoclue.
            yield self.geoclue.callRemote("Start")
            logger.info("Geoclue client started")

        except error.DBusException:
            logger.exception("DBus Error!")
        except Exception:
            logger.exception("Internal error!")


def run():
    scheduler = Scheduler()
    geoclient = GeoClueClient()
    reactor.callWhenRunning(geoclient.main, scheduler.set_location)
    reactor.run()


if __name__ == "__main__":
    run()
