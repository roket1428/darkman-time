from __future__ import annotations

import asyncio
import logging

from .controller import Controller
from .controller import get_cached_location
from .geoclient import GeoClueClient

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)


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
