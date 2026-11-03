from __future__ import annotations

from typing import Annotated, List, Optional
from fastapi import APIRouter, Depends, Query, status
from sqlalchemy.ext.asyncio import AsyncSession
from app.api.deps import get_current_user_id
from app.db.session import get_db
from app.schemas.alerts import Alert, ThresholdsBody
from app.services.device_service import DeviceService
from app.services.alert_service import AlertService
from app.services.reading_service import ReadingService

router = APIRouter(tags=["Alerts"])


@router.get("/api/devices/{device_id}/thresholds", response_model=ThresholdsBody, status_code=status.HTTP_200_OK)
async def get_thresholds(
    device_id: str,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> ThresholdsBody:
    dev_service = DeviceService(db)
    await dev_service.authorize(user_id, device_id)

    alert_service = AlertService(db)
    ts = await alert_service.get_thresholds(device_id)
    return ThresholdsBody(thresholds=ts)


@router.put("/api/devices/{device_id}/thresholds", response_model=ThresholdsBody, status_code=status.HTTP_200_OK)
async def put_thresholds(
    device_id: str,
    body: ThresholdsBody,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> ThresholdsBody:
    dev_service = DeviceService(db)
    await dev_service.authorize(user_id, device_id)

    alert_service = AlertService(db)
    reading_service = ReadingService(db, dev_service, alert_service)

    latest = await reading_service.latest(device_id)
    updated = await alert_service.update_thresholds(user_id, device_id, body.thresholds, latest)
    return ThresholdsBody(thresholds=updated)


@router.get("/api/alerts", response_model=List[Alert], status_code=status.HTTP_200_OK)
async def list_alerts(
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
    device_id: Optional[str] = Query(None, alias="device_id"),
    status_filter: Optional[str] = Query(None, alias="status"),
    limit: int = Query(50, alias="limit"),
) -> List[Alert]:
    alert_service = AlertService(db)
    return await alert_service.list(user_id, device_id, status_filter, limit)


@router.post("/api/alerts/{alert_id}/acknowledge", response_model=Alert, status_code=status.HTTP_200_OK)
async def acknowledge_alert(
    alert_id: int,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Alert:
    alert_service = AlertService(db)
    return await alert_service.acknowledge(user_id, alert_id)

