from __future__ import annotations

import logging
from typing import List, Optional, Tuple
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession
from app.core.errors import AppError
from app.core.security import new_api_key, hash_api_key, API_KEY_PREFIX
from app.schemas.devices import (
    Device,
    DeviceWithKey,
    CreateDeviceInput,
    UpdateDeviceInput,
    clean_name,
)

logger = logging.getLogger("pulse-backend")

DEVICE_COLUMNS = "id, device_id, user_id, name, api_key_hash, api_key_prefix, created_at"


class DeviceService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def list(self, user_id: int) -> List[Device]:
        res = await self.db.execute(
            text(f"SELECT {DEVICE_COLUMNS} FROM devices WHERE user_id = :uid ORDER BY created_at"),
            {"uid": user_id},
        )
        rows = res.mappings().all()
        return [Device(**row) for row in rows]

    async def get(self, user_id: int, device_id: str) -> Device:
        res = await self.db.execute(
            text(f"SELECT {DEVICE_COLUMNS} FROM devices WHERE device_id = :device_id"),
            {"device_id": device_id},
        )
        row = res.mappings().first()
        if not row or row["user_id"] != user_id:
            raise AppError.not_found("device not found")
        return Device(**row)

    async def authorize(self, user_id: int, device_id: str) -> None:
        await self.get(user_id, device_id)

    async def create(self, user_id: int, in_data: CreateDeviceInput) -> DeviceWithKey:
        device_id = in_data.device_id.strip()
        name = clean_name(in_data.name, device_id)

        # Check uniqueness
        res = await self.db.execute(
            text("SELECT 1 FROM devices WHERE device_id = :device_id"),
            {"device_id": device_id},
        )
        if res.scalar():
            raise AppError.conflict("this device ID is already registered")

        raw_key, key_hash, prefix = new_api_key()

        await self.db.execute(
            text(
                """
                INSERT INTO devices (device_id, user_id, name, api_key_hash, api_key_prefix)
                VALUES (:device_id, :user_id, :name, :hash, :prefix)
                """
            ),
            {
                "device_id": device_id,
                "user_id": user_id,
                "name": name,
                "hash": key_hash,
                "prefix": prefix,
            },
        )
        await self.db.commit()

        device = await self.get(user_id, device_id)
        return DeviceWithKey(**device.model_dump(), api_key=raw_key)

    async def rename(self, user_id: int, device_id: str, in_data: UpdateDeviceInput) -> Device:
        device = await self.get(user_id, device_id)
        name = clean_name(in_data.name)

        await self.db.execute(
            text("UPDATE devices SET name = :name WHERE device_id = :device_id"),
            {"name": name, "device_id": device.device_id},
        )
        await self.db.commit()

        device.name = name
        return device

    async def delete(self, user_id: int, device_id: str) -> None:
        device = await self.get(user_id, device_id)
        await self.db.execute(
            text("DELETE FROM devices WHERE device_id = :device_id"),
            {"device_id": device.device_id},
        )
        await self.db.commit()

    async def rotate_key(self, user_id: int, device_id: str) -> DeviceWithKey:
        device = await self.get(user_id, device_id)
        raw_key, key_hash, prefix = new_api_key()

        await self.db.execute(
            text(
                """
                UPDATE devices
                SET api_key_hash = :hash, api_key_prefix = :prefix
                WHERE device_id = :device_id
                """
            ),
            {"hash": key_hash, "prefix": prefix, "device_id": device.device_id},
        )
        await self.db.commit()

        device.api_key_prefix = prefix
        return DeviceWithKey(**device.model_dump(), api_key=raw_key)

    async def owner_of(self, device_id: str) -> Tuple[int, bool]:
        """Check if device is registered; return (user_id, True) or (0, False)."""
        res = await self.db.execute(
            text("SELECT user_id FROM devices WHERE device_id = :device_id"),
            {"device_id": device_id},
        )
        row = res.first()
        if not row:
            return 0, False
        return row[0], True

    async def by_api_key(self, key: str) -> Optional[Device]:
        if not key.startswith(API_KEY_PREFIX):
            return None
        kh = hash_api_key(key)
        res = await self.db.execute(
            text(f"SELECT {DEVICE_COLUMNS} FROM devices WHERE api_key_hash = :hash"),
            {"hash": kh},
        )
        row = res.mappings().first()
        return Device(**row) if row else None

