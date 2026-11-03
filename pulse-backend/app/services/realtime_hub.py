from __future__ import annotations

import asyncio
import json
import logging
from typing import Any, Dict, Set
from fastapi import WebSocket, WebSocketDisconnect

logger = logging.getLogger("pulse-backend")

SEND_BUFFER_SIZE = 64
PING_PERIOD_SECONDS = 54.0


class RealtimeClient:
    def __init__(self, user_id: int, websocket: WebSocket):
        self.user_id = user_id
        self.websocket = websocket
        self.queue: asyncio.Queue[str] = asyncio.Queue(maxsize=SEND_BUFFER_SIZE)
        self.closed = False

    def enqueue(self, message: str) -> bool:
        """Non-blocking message enqueue. Returns False if queue is saturated."""
        if self.closed:
            return False
        try:
            self.queue.put_nowait(message)
            return True
        except asyncio.QueueFull:
            return False

    async def close(self) -> None:
        if not self.closed:
            self.closed = True
            try:
                await self.websocket.close()
            except Exception:
                pass


class RealtimeHub:
    """Per-user fan-out WebSocket hub with slow-client eviction."""

    def __init__(self):
        self._clients: Dict[int, Set[RealtimeClient]] = {}
        self._lock = asyncio.Lock()

    async def register(self, client: RealtimeClient) -> None:
        async with self._lock:
            if client.user_id not in self._clients:
                self._clients[client.user_id] = set()
            self._clients[client.user_id].add(client)
        logger.info("ws: client connected for user %d (total: %d)", client.user_id, len(self._clients[client.user_id]))

    async def unregister(self, client: RealtimeClient) -> None:
        async with self._lock:
            if client.user_id in self._clients:
                self._clients[client.user_id].discard(client)
                if not self._clients[client.user_id]:
                    del self._clients[client.user_id]
        await client.close()
        logger.info("ws: client disconnected for user %d", client.user_id)

    async def publish(self, user_id: int, event_type: str, data: Any) -> None:
        """Publish event to all open dashboards belonging to user_id."""
        envelope = json.dumps({"type": event_type, "data": data}, default=str)

        async with self._lock:
            targets = list(self._clients.get(user_id, []))

        stalled: list[RealtimeClient] = []
        for client in targets:
            ok = client.enqueue(envelope)
            if not ok:
                stalled.append(client)

        for client in stalled:
            logger.warning("ws: client of user %d is not keeping up, disconnecting", user_id)
            await self.unregister(client)

    async def close_all(self) -> None:
        """Disconnect every connected client (used during shutdown)."""
        async with self._lock:
            all_clients = [c for client_set in self._clients.values() for c in client_set]
            self._clients.clear()

        for client in all_clients:
            await client.close()


_hub: RealtimeHub | None = None


def get_realtime_hub() -> RealtimeHub:
    global _hub
    if _hub is None:
        _hub = RealtimeHub()
    return _hub

