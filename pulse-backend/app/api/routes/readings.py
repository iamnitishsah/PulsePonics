from __future__ import annotations

from datetime import datetime
from typing import Annotated, List, Optional
from fastapi import APIRouter, Depends, Query, status
from sqlalchemy.ext.asyncio import AsyncSession
from app.api.deps import get_current_user_id, get_device_by_key
from app.core.errors import AppError
from app.db.session import get_db
from app.schemas.devices import Device
from app.schemas.readings import Reading, ReadingInput, History
from app.services.device_service import DeviceService
from app.services.reading_service import ReadingService
from app.services.alert_service import AlertService

router = APIRouter(tags=["Readings"])


def parse_rfc3339(s: Optional[str], field_name: str) -> Optional[datetime]:
    if not s:
        return None
    try:
        # Replace 'Z' with '+00:00' for Python fromisoformat
        cleaned = s.replace("Z", "+00:00")
        return datetime.fromisoformat(cleaned)
    except Exception:
        raise AppError.invalid(field_name, f"{field_name} must be an RFC3339 time, e.g. 2026-10-24T08:20:00Z")


@router.post("/api/readings", response_model=Reading, status_code=status.HTTP_201_CREATED)
async def ingest_reading(
    in_data: ReadingInput,
    device: Annotated[Device, Depends(get_device_by_key)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Reading:
    if not in_data.device_id:
        in_data.device_id = device.device_id
    if in_data.device_id != device.device_id:
        raise AppError.forbidden("device_id does not match the device key")

    dev_service = DeviceService(db)
    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    return await reading_service.ingest(in_data)


@router.get("/api/devices/{device_id}/readings", response_model=History, status_code=status.HTTP_200_OK)
async def get_history(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
    from_: Optional[str] = Query(None, alias="from"),
    to_: Optional[str] = Query(None, alias="to"),
    max_points: int = Query(500, alias="max_points"),
) -> History:
    dev_service = DeviceService(db)
    await dev_service.authorize(user_id, device_id)

    from_dt = parse_rfc3339(from_, "from")
    to_dt = parse_rfc3339(to_, "to")

    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    return await reading_service.history(device_id, from_dt, to_dt, max_points)


@router.get("/api/devices/{device_id}/readings/latest", response_model=Reading, status_code=status.HTTP_200_OK)
async def get_latest_reading(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Reading:
    dev_service = DeviceService(db)
    await dev_service.authorize(user_id, device_id)

    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    reading = await reading_service.latest(device_id)
    if not reading:
        raise AppError.not_found("no readings yet for this device")
    return reading


@router.get("/api/devices/{device_id}/readings/recent", response_model=List[Reading], status_code=status.HTTP_200_OK)
async def get_recent_readings(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
    limit: int = Query(20, alias="limit"),
) -> List[Reading]:
    dev_service = DeviceService(db)
    await dev_service.authorize(user_id, device_id)

    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    return await reading_service.recent(device_id, limit)

