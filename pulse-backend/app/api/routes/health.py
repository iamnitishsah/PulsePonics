from __future__ import annotations

from typing import List
from fastapi import APIRouter, Depends, status
from fastapi.responses import JSONResponse
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession
from app.db.session import get_db
from app.schemas.readings import Metric, METRICS

router = APIRouter(tags=["Health"])


@router.get("/api/healthz")
async def healthz(db: AsyncSession = Depends(get_db)):
    try:
        await db.execute(text("SELECT 1"))
        return {"status": "ok"}
    except Exception:
        return JSONResponse(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            content={"status": "database unavailable"},
        )


@router.get("/api/metrics", response_model=List[Metric])
async def get_metrics():
    return METRICS

