import asyncio
import logging
from dataclasses import dataclass
from typing import Callable
from typing import Tuple

from jeepney import DBusAddress
from jeepney import new_method_call
from jeepney import Properties
from jeepney.bus_messages import MatchRule
from jeepney.bus_messages import message_bus
from jeepney.io.asyncio import DBusRouter
from jeepney.io.asyncio import open_dbus_connection
from jeepney.io.asyncio import Proxy

logger = logging.getLogger(__name__)


@dataclass
class GeoclueResult:
    Accuracy: float
    Altitude: float
    Description: str
    Heading: float
    Latitude: float
    Longitude: float
    Speed: float
    Timestamp: Tuple[int, int]


class GeoClueClient:
    """An asyncio client for geoclue2."""

    geoclue: DBusAddress
    router: DBusRouter

    def __init__(self, callback: Callable[[GeoclueResult], None]):
        self.callback = callback

    async def stop(self):
        """Tell geoclue to stop polling for geolocation updates."""

        message = new_method_call(self.geoclue, "Stop")
        await self.router.send(message)
        logger.debug("Geoclue client stopped")

    async def _on_location_updated(self, old_path: str, new_path: str):
        """Work with location data to set timers.

        This function is called after GeoClue confirms our location, and sets timers to
        execute sunrise / sundown actions.
        """
        logger.debug("Geoclue indicates that the location has been found.")

        # geoclue will keep on updating the location continuously all day.
        # That's great for other use cases, but once is fine for us.
        await self.stop()

        location_obj = DBusAddress(
            object_path=new_path,
            bus_name="org.freedesktop.GeoClue2",
            interface="org.freedesktop.GeoClue2.Location",
        )
        message = Properties(location_obj).get_all()
        reply = await self.router.send_and_get_reply(message)
        result = GeoclueResult(**{k: v[1] for k, v in reply.body[0].items()})

        logger.info("Got updated location data: %s.", result)
        self.callback(result)

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

        logger.debug("Geoclue manager returned a client path: %s", client_path)

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

        await self.start()

    async def start(self):
        """Start searching for a location."""
        message = new_method_call(self.geoclue, "Start")
        await self.router.send(message)
        logger.info("Geoclue client started")
