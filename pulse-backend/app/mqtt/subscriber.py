from __future__ import annotations

import asyncio
import json
import logging
from urllib.parse import urlparse
from paho.mqtt import client as mqtt
from app.core.config import Settings
from app.core.errors import AppError, ErrorKind
from app.db.session import get_session_maker
from app.schemas.readings import ReadingInput
from app.services.alert_service import AlertService
from app.services.device_service import DeviceService
from app.services.reading_service import ReadingService
from app.services.realtime_hub import get_realtime_hub

logger = logging.getLogger("pulse-backend")

TELEMETRY_TOPIC = "hydroponics/+/telemetry"


def parse_broker_url(url: str) -> tuple[str, int]:
    """Parse tcp://host:port, mqtt://host:port or host:port."""
    if "://" not in url:
        url = "tcp://" + url
    parsed = urlparse(url)
    host = parsed.hostname or "localhost"
    port = parsed.port or 1883
    return host, port


def device_from_topic(topic: str) -> str:
    parts = topic.split("/")
    if len(parts) == 3:
        return parts[1]
    return ""


class MQTTSubscriber:
    def __init__(self, cfg: Settings):
        self.cfg = cfg
        self.host, self.port = parse_broker_url(cfg.mqtt_broker_url)
        self.client_id = cfg.client_id
        self.username = cfg.mqtt_username
        self.password = cfg.mqtt_password

        self.client: mqtt.Client | None = None
        self._queue: asyncio.Queue[tuple[str, bytes]] = asyncio.Queue(maxsize=1000)
        self._consumer_task: asyncio.Task | None = None
        self._loop: asyncio.AbstractEventLoop | None = None

    def start(self, loop: asyncio.AbstractEventLoop) -> None:
        self._loop = loop
        self._consumer_task = asyncio.create_task(self._process_queue())

        try:
            # Paho MQTT Client CallbackAPIVersion.VERSION2
            self.client = mqtt.Client(
                callback_api_version=mqtt.CallbackAPIVersion.VERSION2,
                client_id=self.client_id,
            )
        except AttributeError:
            self.client = mqtt.Client(client_id=self.client_id)

        if self.username:
            self.client.username_pw_set(self.username, self.password)

        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.client.on_message = self._on_message

        self.client.reconnect_delay_set(min_delay=1, max_delay=30)

        try:
            self.client.connect_async(self.host, self.port, keepalive=60)
            self.client.loop_start()
            logger.info("mqtt: connecting to %s:%d as client %s", self.host, self.port, self.client_id)
        except Exception as e:
            logger.error("mqtt: connect error: %s", e)

    def stop(self) -> None:
        if self.client:
            self.client.loop_stop()
            self.client.disconnect()
            self.client = None
        if self._consumer_task:
            self._consumer_task.cancel()
            self._consumer_task = None
        logger.info("mqtt: stopped subscriber")

    def _on_connect(self, client: mqtt.Client, userdata: any, flags: any, reason_code: any, properties: any = None) -> None:
        rc = getattr(reason_code, "value", reason_code)
        if rc == 0:
            logger.info("mqtt: connected to %s:%d, subscribing to %s", self.host, self.port, TELEMETRY_TOPIC)
            client.subscribe(TELEMETRY_TOPIC, qos=1)
        else:
            logger.warning("mqtt: connection failed with reason %s", reason_code)

    def _on_disconnect(self, client: mqtt.Client, userdata: any, flags: any, reason_code: any, properties: any = None) -> None:
        logger.warning("mqtt: connection lost: %s (reconnecting)", reason_code)

    def _on_message(self, client: mqtt.Client, userdata: any, msg: mqtt.MQTTMessage) -> None:
        if self._loop and not self._loop.is_closed():
            try:
                self._loop.call_soon_threadsafe(self._queue.put_nowait, (msg.topic, msg.payload))
            except asyncio.QueueFull:
                logger.warning("mqtt: incoming message queue full, dropping message from %s", msg.topic)

    async def _process_queue(self) -> None:
        session_maker = get_session_maker()
        hub = get_realtime_hub()

        while True:
            try:
                topic, payload = await self._queue.get()
                topic_device = device_from_topic(topic)
                if not topic_device:
                    self._queue.task_done()
                    continue

                try:
                    data = json.loads(payload.decode("utf-8"))
                except Exception as e:
                    logger.warning("mqtt: %s: invalid JSON: %s", topic, e)
                    self._queue.task_done()
                    continue

                if not data.get("device_id"):
                    data["device_id"] = topic_device

                if data.get("device_id") != topic_device:
                    logger.warning(
                        'mqtt: %s: payload device_id "%s" does not match topic, dropped',
                        topic,
                        data.get("device_id"),
                    )
                    self._queue.task_done()
                    continue

                async with session_maker() as session:
                    dev_service = DeviceService(session)
                    alert_service = AlertService(session, hub)
                    reading_service = ReadingService(session, dev_service, alert_service, hub)

                    try:
                        in_payload = ReadingInput(**data)
                        reading = await reading_service.ingest(in_payload)
                        logger.info("mqtt: stored reading id=%d device=%s", reading.id, reading.device_id)
                    except AppError as e:
                        if e.kind == ErrorKind.NOT_FOUND:
                            logger.info('mqtt: device "%s" is not registered, reading dropped', topic_device)
                        elif e.kind == ErrorKind.CONFLICT:
                            logger.info(
                                "mqtt: duplicate reading device=%s recorded_at=%s ignored (firmware retry)",
                                topic_device,
                                data.get("recorded_at"),
                            )
                        elif e.kind == ErrorKind.INVALID:
                            logger.warning("mqtt: rejected reading from %s: %s", topic_device, e)
                        else:
                            logger.error("mqtt: failed to store reading from %s: %s", topic_device, e)
                    except Exception as e:
                        logger.error("mqtt: unexpected error storing reading from %s: %s", topic_device, e)

                self._queue.task_done()
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("mqtt: queue worker error: %s", e)

