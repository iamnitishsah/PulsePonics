from __future__ import annotations

from typing import Annotated, Optional
from fastapi import Depends, Header, Request
from sqlalchemy.ext.asyncio import AsyncSession
from app.core.config import get_settings
from app.core.errors import AppError
from app.core.security import parse_access_token
from app.db.session import get_db
from app.schemas.devices import Device
from app.services.device_service import DeviceService


async def get_current_user_id(
    authorization: Annotated[Optional[str], Header()] = None,
) -> int:
    """Dependency that parses and verifies the Bearer access token."""
    if not authorization:
        raise AppError.unauthorized("missing bearer token")
    if not authorization.startswith("Bearer "):
        raise AppError.unauthorized("missing bearer token")
    token = authorization[len("Bearer ") :].strip()
    if not token:
        raise AppError.unauthorized("missing bearer token")

    cfg = get_settings()
    return parse_access_token(token, cfg.jwt_secret)


async def get_device_by_key(
    request: Request,
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Device:
    """Dependency that authenticates devices via X-Device-Key header."""
    key = request.headers.get("x-device-key", "").strip()
    if not key:
        raise AppError.unauthorized("missing X-Device-Key header")

    dev_service = DeviceService(db)
    device = await dev_service.by_api_key(key)
    if not device:
        raise AppError.unauthorized("invalid device key")
    return device

