from __future__ import annotations

import asyncio
import json
import logging
import os
import subprocess
from dataclasses import dataclass
from datetime import datetime
from datetime import timedelta
from enum import Enum
from pathlib import Path
from typing import Callable
from typing import Optional
from typing import Tuple

import xdg.BaseDirectory
from astral import Observer
from astral.sun import sun
from dateutil.tz import tzlocal
from jeepney import DBusAddress
from jeepney import new_method_call
from jeepney import Properties
from jeepney.bus_messages import MatchRule
from jeepney.bus_messages import message_bus
from jeepney.io.asyncio import DBusRouter
from jeepney.io.asyncio import open_dbus_connection
from jeepney.io.asyncio import Proxy
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
        convention by looking in ``$XDG_DATA_DIRS/$MODE.d/``.

        Where $MODE is either ``dark-mode`` or ``light-mode``.
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


class Controller:
    """Main controller that understands the current state.

    The controller understands the currently set mode, and handles scheduling
    of future changes.
    """

    _location: Optional[Observer] = None
    _mode = Optional[Mode]
    _next = Optional[asyncio.Handle]

    def __init__(self, location: Optional[Observer]):
        self.set_location(location)

    @property
    def location(self) -> Optional[Observer]:
        return self._location

    @property
    def mode(self) -> Optional[Mode]:
        return self._mode  # type: ignore

    def set_location(self, location: Observer) -> None:
        """Set the current location and set timers to update the mode."""
        if location != self._location:
            self._location = location
            self._set_timer()

    def activate_mode(self, mode: Mode) -> None:
        """Activates a specific mode, and sets tasks for the next change."""

        self._model = mode
        mode.activate()
        self._set_timer()

    def _set_timer(self) -> None:
        """Schedule the next mode transition."""

        next_time, next_mode = self.calculate_next_change()

        if not self.mode:  # If this is the first run.
            # Activate the opposite now. E.g.: If the next change is a
            # transition to dark mode, then we should be in light mode now.
            self._mode = next_mode.opposite
            self._mode.activate()

        wait_for = (next_time - datetime.now(tzlocal())).total_seconds()
        logger.info("Will change to %s at %s.", next_mode, next_time)

        loop = asyncio.get_event_loop()
        self._next = loop.call_later(wait_for, self.activate_mode, next_mode)

    def calculate_next_change(self, date=None) -> Tuple[datetime, Mode]:
        """Return the next event."""

        local_sun = sun(self.location, date=date, tzinfo=tzlocal())

        light_time = local_sun["dawn"]
        dark_time = local_sun["dusk"] + (local_sun["dusk"] - local_sun["sunset"])

        now = datetime.now(tzlocal())

        # XXX: There's an assumption made in this code that sunrise always comes before
        # sunset. I _think_ this is true anywhere in the world any time of the
        # year, though have a feeling that this might be one of these silly non-truths
        # we programmers assume somehow. 🤔

        if dark_time < now:
            # Already dark today, next change is tomorrow:
            return self.calculate_next_change(now + timedelta(days=1))
        elif light_time < now < dark_time:
            return dark_time, Mode.Dark
        elif now < light_time:
            return light_time, Mode.Light
        else:
            raise Exception("Something went wrong. Please report this!")

    # gsettings set io.elementary.terminal.settings prefer-dark-style true


class GeoClueClient:
    """A client for geoclue2 that waits for the location and runs the callback."""

    geoclue: DBusAddress
    router: DBusRouter

    def __init__(self, callback: Callable[[Observer], None]):
        self.callback = callback

    async def _stop_geolocation(self):
        """Tell geoclue to stop polling for geolocation updates."""

        message = new_method_call(self.geoclue, "Stop")
        await self.router.send(message)
        logger.info("Geoclue client stopped")

    async def _on_location_updated(self, old_path: str, new_path: str):
        """Work with location data to set timers.

        This function is called after GeoClue confirms our location, and sets timers to
        execute sunrise / sundown actions.
        """
        logger.info("Received location update signal from geoclue")

        # geoclue will keep on updating the location continuously all day.
        # Don't want that.
        await self._stop_geolocation()

        location_obj = DBusAddress(
            object_path=new_path,
            bus_name="org.freedesktop.GeoClue2",
            interface="org.freedesktop.GeoClue2.Location",
        )
        message = Properties(location_obj).get_all()
        reply = await self.router.send_and_get_reply(message)
        props = {k: v[1] for k, v in reply.body[0].items()}
        lat = props["Latitude"]
        lon = props["Longitude"]

        logger.info("Got updated location data: %f, %f", lat, lon)
        save_location_into_cache(lat=lat, lng=lon)
        location = Observer(latitude=lat, longitude=lon)

        self.callback(location)

    async def _create_geoclue_object(self) -> None:
        """Creates a geoclue client, and returns it.

        Clients are private and per-connection. So we need to keep the connection around
        to further communicate with the client using it.
        """

        # Ask the manager API to create a client
        manager = DBusAddress(
            object_path="/org/freedesktop/GeoClue2/Manager",
            bus_name="org.freedesktop.GeoClue2",
            interface="org.freedesktop.GeoClue2.Manager",
        )
        message = new_method_call(manager, "GetClient")
        reply = await self.router.send_and_get_reply(message)
        self.client_path = client_path = reply.body[0]

        logger.info("Geoclue manager returned a client path: %s", client_path)

        self.geoclue = DBusAddress(
            object_path=client_path,
            bus_name="org.freedesktop.GeoClue2",
            interface="org.freedesktop.GeoClue2.Client",
        )

        # This value needs to be set for some form of authorisation.
        # I've no idea what the _right_ value is, but this works fine.
        # Asked upstream at https://gitlab.freedesktop.org/geoclue/geoclue/-/issues/138
        message = Properties(self.geoclue).set("DesktopId", "s", "9")
        await self.router.send(message)

    async def listen(self, ready: asyncio.Event):
        # Set a callback for location updates.
        match_rule = MatchRule(
            type="signal",
            interface="org.freedesktop.GeoClue2.Client",
            path=self.client_path,
        )
        await Proxy(message_bus, self.router).AddMatch(match_rule)

        with self.router.filter(match_rule) as q:
            ready.set()
            msg = await q.get()

            old_path, new_path = msg.body
            await self._on_location_updated(old_path, new_path)

    async def main(self):
        """Listens to location changes."""

        # Geoclue expects all calls to be made from the same connection:
        conn = await open_dbus_connection(bus="SYSTEM")
        self.router = DBusRouter(conn)

        await self._create_geoclue_object()
        logger.info("Got geoclue client: %s.", self.geoclue)

        listener_ready = asyncio.Event()
        asyncio.create_task(self.listen(listener_ready), name="GeoclueSignalListener")
        await listener_ready.wait()

        message = new_method_call(self.geoclue, "Start")
        await self.router.send(message)
        logger.info("Geoclue client started")


def save_location_into_cache(lat: float, lng: float) -> None:
    os.makedirs(
        os.path.join(xdg.BaseDirectory.xdg_cache_home, "darkman"),
        exist_ok=True,
    )
    cache = os.path.join(xdg.BaseDirectory.xdg_cache_home, "darkman", "location.json")
    with open(cache, "w") as f:
        json.dump({"lat": lat, "lng": lng}, f)


def get_cached_location() -> Optional[Observer]:
    cache = os.path.join(xdg.BaseDirectory.xdg_cache_home, "darkman", "location.json")
    if not os.path.isfile(cache):
        logger.info("No cached location found.")
        return None

    with open(cache) as f:
        cached_data = json.load(f)

    logger.info(
        "Found cached location data: %f, %f",
        cached_data["lat"],
        cached_data["lng"],
    )

    return Observer(latitude=cached_data["lat"], longitude=cached_data["lng"])


def run():
    # XXX: Maybe connect to system'd login manager and detect when the system suspends
    # and shit.
    #
    # async tasks can also be cancelled.

    location = get_cached_location()
    controller = Controller(location)
    geoclient = GeoClueClient(controller.set_location)

    loop = asyncio.get_event_loop()
    loop.create_task(geoclient.main(), name="Main")

    try:
        loop.run_forever()
    except KeyboardInterrupt:
        loop.stop()


if __name__ == "__main__":
    run()
