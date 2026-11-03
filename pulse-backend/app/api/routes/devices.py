from __future__ import annotations

from typing import Annotated, List
from fastapi import APIRouter, Depends, Response, status
from sqlalchemy.ext.asyncio import AsyncSession
from app.api.deps import get_current_user_id
from app.db.session import get_db
from app.schemas.devices import (
    Device,
    DeviceWithKey,
    DeviceWithSummary,
    CreateDeviceInput,
    UpdateDeviceInput,
)
from app.services.device_service import DeviceService
from app.services.reading_service import ReadingService
from app.services.alert_service import AlertService

router = APIRouter(tags=["Devices"])


async def build_device_summary(
    device: Device,
    reading_service: ReadingService,
    alert_service: AlertService,
) -> DeviceWithSummary:
    latest = await reading_service.latest(device.device_id)
    open_alerts = await alert_service.count_open(device.device_id)
    return DeviceWithSummary(
        **device.model_dump(),
        latest=latest.model_dump(mode="json") if latest else None,
        open_alerts=open_alerts,
    )


@router.get("/api/devices", response_model=List[DeviceWithSummary], status_code=status.HTTP_200_OK)
async def list_devices(
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> List[DeviceWithSummary]:
    dev_service = DeviceService(db)
    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    devices = await dev_service.list(user_id)
    out: List[DeviceWithSummary] = []
    for d in devices:
        item = await build_device_summary(d, reading_service, alert_service)
        out.append(item)
    return out


@router.post("/api/devices", response_model=DeviceWithKey, status_code=status.HTTP_201_CREATED)
async def create_device(
    in_data: CreateDeviceInput,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> DeviceWithKey:
    dev_service = DeviceService(db)
    return await dev_service.create(user_id, in_data)


@router.get("/api/devices/{device_id}", response_model=DeviceWithSummary, status_code=status.HTTP_200_OK)
async def get_device(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> DeviceWithSummary:
    dev_service = DeviceService(db)
    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    device = await dev_service.get(user_id, device_id)
    return await build_device_summary(device, reading_service, alert_service)


@router.patch("/api/devices/{device_id}", response_model=Device, status_code=status.HTTP_200_OK)
async def rename_device(
    device_id: str,
    in_data: UpdateDeviceInput,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Device:
    dev_service = DeviceService(db)
    return await dev_service.rename(user_id, device_id, in_data)


@router.delete("/api/devices/{device_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_device(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Response:
    dev_service = DeviceService(db)
    await dev_service.delete(user_id, device_id)
    return Response(status_code=status.HTTP_204_NO_CONTENT)


@router.post("/api/devices/{device_id}/api-key", response_model=DeviceWithKey, status_code=status.HTTP_200_OK)
async def rotate_device_key(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> DeviceWithKey:
    dev_service = DeviceService(db)
    return await dev_service.rotate_key(user_id, device_id)

