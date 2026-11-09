from __future__ import annotations

import logging
import math
from datetime import datetime, timezone, timedelta
from typing import Any, List, Optional
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession
from app.core.errors import AppError
from app.core.time import to_utc, to_ist, utc_now
from app.schemas.readings import Reading, ReadingInput, Point, History
from app.services.device_service import DeviceService
from app.services.alert_service import AlertService
from app.services.realtime_hub import RealtimeHub, get_realtime_hub

logger = logging.getLogger("pulse-backend")

DEFAULT_HISTORY_RANGE = timedelta(hours=24)
MAX_HISTORY_RANGE = timedelta(days=31)
DEFAULT_MAX_POINTS = 500
MIN_MAX_POINTS = 10
MAX_MAX_POINTS = 2000
DEFAULT_RECENT_LIMIT = 20
MAX_RECENT_LIMIT = 200

NICE_BUCKETS = [10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600, 7200, 10800, 21600, 43200, 86400]

READING_COLUMNS = """id, device_id, ph, ec_ms_cm, water_temp_c, air_temp_c,
    humidity_pct, pressure_hpa, recorded_at, created_at"""


def bucket_for(duration: timedelta, max_points: int) -> int:
    """Pick smallest nice bucket that keeps points <= max_points matching Go bucketFor."""
    need = int(math.ceil(duration.total_seconds() / float(max_points - 1)))
    for b in NICE_BUCKETS:
        if b >= need:
            return b
    return need


class ReadingService:
    def __init__(
        self,
        db: AsyncSession,
        device_service: DeviceService,
        alert_service: AlertService,
        hub: RealtimeHub | None = None,
    ):
        self.db = db
        self.device_service = device_service
        self.alert_service = alert_service
        self.hub = hub or get_realtime_hub()

    async def ingest(self, in_data: ReadingInput) -> Reading:
        in_data.validate_payload()
        assert in_data.device_id is not None
        assert in_data.recorded_at is not None

        owner_id, ok = await self.device_service.owner_of(in_data.device_id)
        if not ok:
            raise AppError.not_found("device is not registered")

        # Stored in UTC
        recorded_at_utc = to_utc(in_data.recorded_at)

        # Check for duplicate reading (device_id, recorded_at)
        res = await self.db.execute(
            text("SELECT 1 FROM readings WHERE device_id = :device_id AND recorded_at = :recorded_at"),
            {"device_id": in_data.device_id, "recorded_at": recorded_at_utc},
        )
        if res.scalar():
            raise AppError.conflict("a reading with this recorded_at is already stored")

        insert_res = await self.db.execute(
            text(
                """
                INSERT INTO readings (device_id, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa, recorded_at)
                VALUES (:device_id, :ph, :ec_ms_cm, :water_temp_c, :air_temp_c, :humidity_pct, :pressure_hpa, :recorded_at)
                """
            ),
            {
                "device_id": in_data.device_id,
                "ph": in_data.ph,
                "ec_ms_cm": in_data.ec_ms_cm,
                "water_temp_c": in_data.water_temp_c,
                "air_temp_c": in_data.air_temp_c,
                "humidity_pct": in_data.humidity_pct,
                "pressure_hpa": in_data.pressure_hpa,
                "recorded_at": recorded_at_utc,
            },
        )
        await self.db.commit()
        reading_id = insert_res.lastrowid

        # Read back row so response and ws event carry exact stored decimals
        reading_dict = await self._get_reading_by_id(reading_id)
        if not reading_dict:
            raise RuntimeError("failed to read back inserted reading")

        reading = Reading(**reading_dict)

        # Notify websocket clients
        await self.hub.publish(owner_id, "reading", reading.model_dump(mode="json"))

        # Evaluate alerts
        await self.alert_service.evaluate(owner_id, reading)

        return reading

    async def _get_reading_by_id(self, reading_id: int) -> Optional[dict]:
        res = await self.db.execute(
            text(f"SELECT {READING_COLUMNS} FROM readings WHERE id = :id"),
            {"id": reading_id},
        )
        row = res.mappings().first()
        return dict(row) if row else None

    async def latest(self, device_id: str) -> Optional[Reading]:
        res = await self.db.execute(
            text(f"SELECT {READING_COLUMNS} FROM readings WHERE device_id = :did ORDER BY recorded_at DESC LIMIT 1"),
            {"did": device_id},
        )
        row = res.mappings().first()
        return Reading(**row) if row else None

    async def recent(self, device_id: str, limit: int = DEFAULT_RECENT_LIMIT) -> List[Reading]:
        if limit <= 0:
            limit = DEFAULT_RECENT_LIMIT
        if limit > MAX_RECENT_LIMIT:
            raise AppError.invalid("limit", f"limit must be at most {MAX_RECENT_LIMIT}")

        res = await self.db.execute(
            text(f"SELECT {READING_COLUMNS} FROM readings WHERE device_id = :did ORDER BY recorded_at DESC LIMIT :limit"),
            {"did": device_id, "limit": limit},
        )
        rows = res.mappings().all()
        return [Reading(**r) for r in rows]

    async def history(
        self,
        device_id: str,
        from_time: Optional[datetime] = None,
        to_time: Optional[datetime] = None,
        max_points: int = DEFAULT_MAX_POINTS,
    ) -> History:
        now = utc_now()
        to_t = to_utc(to_time) if to_time else now
        from_t = to_utc(from_time) if from_time else (to_t - DEFAULT_HISTORY_RANGE)

        if not from_t < to_t:
            raise AppError.invalid("from", "from must be before to")
        if (to_t - from_t) > MAX_HISTORY_RANGE:
            raise AppError.invalid("from", "range must be at most 31 days")

        if max_points == 0:
            max_points = DEFAULT_MAX_POINTS
        if max_points < MIN_MAX_POINTS or max_points > MAX_MAX_POINTS:
            raise AppError.invalid(
                "max_points",
                f"max_points must be between {MIN_MAX_POINTS} and {MAX_MAX_POINTS}",
            )

        # Count records
        res = await self.db.execute(
            text("SELECT COUNT(*) FROM readings WHERE device_id = :did AND recorded_at >= :from AND recorded_at <= :to"),
            {"did": device_id, "from": from_t, "to": to_t},
        )
        count = res.scalar() or 0

        if count <= max_points:
            # Return raw readings
            range_res = await self.db.execute(
                text(
                    """
                    SELECT recorded_at, ph, ec_ms_cm, water_temp_c, air_temp_c, humidity_pct, pressure_hpa
                    FROM readings
                    WHERE device_id = :did AND recorded_at >= :from AND recorded_at <= :to
                    ORDER BY recorded_at
                    """
                ),
                {"did": device_id, "from": from_t, "to": to_t},
            )
            raw_points = [Point(**row) for row in range_res.mappings().all()]
            return History(
                device_id=device_id,
                from_=from_t,
                to=to_t,
                bucket_seconds=0,
                points=raw_points,
            )

        # Aggregate downsampled points
        bucket_seconds = bucket_for(to_t - from_t, max_points)
        agg_res = await self.db.execute(
            text(
                """
                SELECT
                    FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(recorded_at) / :bucket) * :bucket) AS recorded_at,
                    AVG(ph) AS ph, AVG(ec_ms_cm) AS ec_ms_cm, AVG(water_temp_c) AS water_temp_c,
                    AVG(air_temp_c) AS air_temp_c, AVG(humidity_pct) AS humidity_pct, AVG(pressure_hpa) AS pressure_hpa
                FROM readings
                WHERE device_id = :did AND recorded_at >= :from AND recorded_at <= :to
                GROUP BY 1
                ORDER BY 1
                """
            ),
            {"bucket": bucket_seconds, "did": device_id, "from": from_t, "to": to_t},
        )
        agg_points = [Point(**row) for row in agg_res.mappings().all()]
        return History(
            device_id=device_id,
            from_=from_t,
            to=to_t,
            bucket_seconds=bucket_seconds,
            points=agg_points,
        )

