from __future__ import annotations

import asyncio
import logging
from typing import Optional
from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect, status
from fastapi.responses import JSONResponse
from app.core.config import get_settings
from app.core.errors import AppError, error_payload
from app.core.middleware import OriginPolicy
from app.core.security import parse_access_token
from app.services.realtime_hub import RealtimeClient, get_realtime_hub

logger = logging.getLogger("pulse-backend")

router = APIRouter(tags=["WebSocket"])


@router.get("/api/ws")
async def ws_precheck(token: Optional[str] = Query(None)):
    """HTTP check endpoint for non-WebSocket clients (e.g. curl/smoke-test validation)."""
    cfg = get_settings()
    if not token:
        raise AppError.unauthorized("invalid or expired access token")
    try:
        parse_access_token(token, cfg.jwt_secret)
    except Exception:
        raise AppError.unauthorized("invalid or expired access token")
    return JSONResponse(
        status_code=status.HTTP_426_UPGRADE_REQUIRED,
        content={"message": "Upgrade to WebSocket required"},
    )


@router.websocket("/api/ws")
async def websocket_endpoint(websocket: WebSocket, token: Optional[str] = None):
    cfg = get_settings()
    origin_policy = OriginPolicy(cfg.cors_origins)

    # Validate Origin
    origin = websocket.headers.get("origin")
    if origin and not origin_policy.is_allowed(origin):
        logger.warning("ws: rejected origin: %s", origin)
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION)
        return

    # Validate Token
    if not token:
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION)
        return

    try:
        user_id = parse_access_token(token, cfg.jwt_secret)
    except Exception:
        await websocket.close(code=status.WS_1008_POLICY_VIOLATION)
        return

    await websocket.accept()
    hub = get_realtime_hub()
    client = RealtimeClient(user_id=user_id, websocket=websocket)
    await hub.register(client)

    async def writer_loop():
        try:
            while not client.closed:
                msg = await client.queue.get()
                await websocket.send_text(msg)
                client.queue.task_done()
        except asyncio.CancelledError:
            pass
        except Exception as e:
            logger.debug("ws: writer error: %s", e)
        finally:
            await hub.unregister(client)

    async def reader_loop():
        try:
            while not client.closed:
                # Discard client incoming messages, process control frames
                await websocket.receive_text()
        except (WebSocketDisconnect, asyncio.CancelledError):
            pass
        except Exception as e:
            logger.debug("ws: reader error: %s", e)
        finally:
            await hub.unregister(client)

    writer_task = asyncio.create_task(writer_loop())
    reader_task = asyncio.create_task(reader_loop())

    done, pending = await asyncio.wait(
        [writer_task, reader_task],
        return_when=asyncio.FIRST_COMPLETED,
    )
    for task in pending:
        task.cancel()
    await hub.unregister(client)
