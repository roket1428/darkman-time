from __future__ import annotations

import asyncio
import json
import logging
import os
import subprocess
from datetime import datetime
from datetime import timedelta
from enum import Enum
from pathlib import Path
from typing import Optional
from typing import Tuple

import xdg.BaseDirectory
from astral import Observer
from astral.sun import sun
from dateutil.tz import tzlocal
from xdg import BaseDirectory

from .geoclient import GeoClueClient
from .geoclient import GeoclueResult

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
            self.transition()
        else:
            logger.info("The location is the same as the previous one. Nothing to do.")

    def set_mode(self, mode: Mode) -> None:
        """Change the current mode and activate it."""
        self._mode = mode
        self._mode.activate()

    def geoclue_callback(self, result: GeoclueResult) -> None:
        save_location_into_cache(lat=result.Latitude, lng=result.Longitude)

        location = Observer(latitude=result.Latitude, longitude=result.Longitude)
        self.set_location(location)

    def transition(self) -> None:
        """Transition to the correct mode and queue the next change.

        Calculate the correct mode for right now, and activate it. Also
        determine the time for the next change, and queue a transition.
        """

        next_time, next_mode = self.calculate_next_change()

        # Activate the opposite now. E.g.: If the next change is a
        # transition to dark mode, then we should be in light mode now.
        self.set_mode(next_mode.opposite)

        wait_for = (next_time - datetime.now(tzlocal())).total_seconds()
        logger.info("Will change to %s at %s.", next_mode, next_time)

        loop = asyncio.get_event_loop()
        loop.call_later(wait_for, self.transition)

    def calculate_next_change(self, date=None) -> Tuple[datetime, Mode]:
        """Return the next event."""

        local_sun = sun(self.location, date=date, tzinfo=tzlocal())

        light_time = local_sun["dawn"]
        dark_time = local_sun["sunset"]

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
    # XXX: Maybe connect to system'd login manager and detect when the
    # system suspends and alike.

    location = get_cached_location()
    controller = Controller(location)
    geoclue = GeoClueClient(controller.geoclue_callback)

    loop = asyncio.get_event_loop()
    loop.create_task(geoclue.main(), name="GeoclueMain")

    try:
        loop.run_forever()
    except KeyboardInterrupt:
        loop.stop()


if __name__ == "__main__":
    run()
