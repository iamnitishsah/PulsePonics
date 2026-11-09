from __future__ import annotations

import logging
from datetime import datetime, timezone
from enum import Enum
from typing import Any, Dict, List, Optional
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession
from app.core.errors import AppError
from app.core.time import to_utc, to_ist, utc_now
from app.schemas.alerts import (
    Threshold,
    Alert,
    default_thresholds,
)
from app.schemas.readings import Reading, METRICS
from app.services.realtime_hub import RealtimeHub, get_realtime_hub

logger = logging.getLogger("pulse-backend")

HYSTERESIS_FRACTION = 0.05

ALERT_COLUMNS = """a.id, a.device_id, a.metric, a.kind, a.threshold_value, a.trigger_value,
    a.latest_value, a.opened_at, a.resolved_at, a.acknowledged_at"""


class Action(str, Enum):
    NONE = "none"
    OPEN = "open"
    UPDATE = "update"
    RESOLVE = "resolve"


class Decision:
    def __init__(self, action: Action, kind: str = "", bound: float = 0.0):
        self.action = action
        self.kind = kind
        self.bound = bound


def decide(t: Threshold, value: float, open_alert: Optional[Alert]) -> Decision:
    """Core hysteresis decision logic matching Go decide()."""
    if not t.enabled:
        if open_alert is not None:
            return Decision(Action.RESOLVE)
        return Decision(Action.NONE)

    breached_kind = ""
    breached_bound = 0.0
    if t.min is not None and value < t.min:
        breached_kind = "low"
        breached_bound = t.min
    elif t.max is not None and value > t.max:
        breached_kind = "high"
        breached_bound = t.max

    if breached_kind:
        if open_alert is None:
            return Decision(Action.OPEN, breached_kind, breached_bound)
        return Decision(Action.UPDATE, breached_kind, breached_bound)

    if open_alert is None:
        return Decision(Action.NONE)

    # Check hysteresis margin for resolution
    margin = 0.0
    if t.min is not None and t.max is not None:
        margin = (t.max - t.min) * HYSTERESIS_FRACTION

    cleared = True
    if open_alert.kind == "low" and t.min is not None and value < (t.min + margin):
        cleared = False
    if open_alert.kind == "high" and t.max is not None and value > (t.max - margin):
        cleared = False

    if cleared:
        return Decision(Action.RESOLVE)

    return Decision(Action.UPDATE, open_alert.kind, open_alert.threshold_value)


def merge_thresholds(stored: List[Threshold]) -> List[Threshold]:
    by_metric = {t.metric: t for t in stored}
    out = default_thresholds()
    for idx, def_th in enumerate(out):
        if def_th.metric in by_metric:
            out[idx] = by_metric[def_th.metric]
    return out


class AlertService:
    def __init__(self, db: AsyncSession, hub: RealtimeHub | None = None):
        self.db = db
        self.hub = hub or get_realtime_hub()

    async def get_thresholds(self, device_id: str) -> List[Threshold]:
        res = await self.db.execute(
            text("SELECT metric, min_value, max_value, enabled FROM thresholds WHERE device_id = :device_id"),
            {"device_id": device_id},
        )
        stored = [Threshold(**row) for row in res.mappings().all()]
        return merge_thresholds(stored)

    async def update_thresholds(
        self,
        user_id: int,
        device_id: str,
        in_thresholds: List[Threshold],
        latest_reading: Optional[Reading] = None,
    ) -> List[Threshold]:
        if not in_thresholds:
            raise AppError.invalid("thresholds", "at least one threshold is required")

        seen: set[str] = set()
        for t in in_thresholds:
            t.validate_bounds()
            if t.metric in seen:
                raise AppError.invalid(t.metric, f"duplicate metric '{t.metric}'")
            seen.add(t.metric)

        for t in in_thresholds:
            await self.db.execute(
                text(
                    """
                    INSERT INTO thresholds (device_id, metric, min_value, max_value, enabled)
                    VALUES (:device_id, :metric, :min_val, :max_val, :enabled)
                    ON DUPLICATE KEY UPDATE
                        min_value = VALUES(min_value),
                        max_value = VALUES(max_value),
                        enabled = VALUES(enabled)
                    """
                ),
                {
                    "device_id": device_id,
                    "metric": t.metric,
                    "min_val": t.min,
                    "max_val": t.max,
                    "enabled": t.enabled,
                },
            )
        await self.db.commit()

        # Resolve alerts for metrics that were just disabled
        open_alerts = await self.get_open_alerts(device_id)
        now = utc_now()
        for t in in_thresholds:
            if not t.enabled and t.metric in open_alerts:
                await self._resolve(user_id, open_alerts[t.metric], now)

        # Re-evaluate latest reading immediately if available
        if latest_reading:
            await self.evaluate(user_id, latest_reading)

        return await self.get_thresholds(device_id)

    async def get_open_alerts(self, device_id: str) -> Dict[str, Alert]:
        res = await self.db.execute(
            text(f"SELECT {ALERT_COLUMNS} FROM alerts a WHERE a.device_id = :did AND a.resolved_at IS NULL"),
            {"did": device_id},
        )
        out: Dict[str, Alert] = {}
        for row in res.mappings().all():
            a = Alert(**row)
            a.compute_status()
            out[a.metric] = a
        return out

    async def evaluate(self, user_id: int, r: Reading) -> None:
        try:
            thresholds = await self.get_thresholds(r.device_id)
            open_alerts = await self.get_open_alerts(r.device_id)

            for t in thresholds:
                v = r.get_metric_value(t.metric)
                if v is None:
                    continue  # sensor sent null: keep alert state unchanged

                current = open_alerts.get(t.metric)
                d = decide(t, v, current)

                if d.action == Action.OPEN:
                    try:
                        res = await self.db.execute(
                            text(
                                """
                                INSERT INTO alerts (device_id, metric, kind, threshold_value, trigger_value, latest_value, opened_at)
                                VALUES (:device_id, :metric, :kind, :threshold_value, :trigger_value, :latest_value, :opened_at)
                                """
                            ),
                            {
                                "device_id": r.device_id,
                                "metric": t.metric,
                                "kind": d.kind,
                                "threshold_value": d.bound,
                                "trigger_value": v,
                                "latest_value": v,
                                "opened_at": to_utc(r.recorded_at),
                            },
                        )
                        await self.db.commit()
                        alert_id = res.lastrowid
                        alert_dict = await self._get_alert_by_id(alert_id)
                        if alert_dict:
                            alert_obj = Alert(**alert_dict)
                            alert_obj.compute_status()
                            await self.hub.publish(user_id, "alert", alert_obj.model_dump(mode="json"))
                    except Exception as e:
                        # Duplicate key on uq_alerts_open is expected if concurrent insert happened
                        await self.db.rollback()
                        logger.debug("alerts: insert open alert skipped: %s", e)

                elif d.action == Action.UPDATE and current is not None:
                    await self.db.execute(
                        text(
                            """
                            UPDATE alerts
                            SET kind = :kind, threshold_value = :bound, latest_value = :last
                            WHERE id = :id AND resolved_at IS NULL
                            """
                        ),
                        {"kind": d.kind, "bound": d.bound, "last": v, "id": current.id},
                    )
                    await self.db.commit()

                elif d.action == Action.RESOLVE and current is not None:
                    await self._resolve(user_id, current, r.recorded_at)

        except Exception as e:
            logger.error("alerts: evaluate reading %d (%s) failed: %s", r.id, r.device_id, e)

    async def _resolve(self, user_id: int, a: Alert, at: datetime) -> None:
        await self.db.execute(
            text("UPDATE alerts SET resolved_at = :at WHERE id = :id AND resolved_at IS NULL"),
            {"at": to_utc(at), "id": a.id},
        )
        await self.db.commit()
        alert_dict = await self._get_alert_by_id(a.id)
        if alert_dict:
            resolved = Alert(**alert_dict)
            resolved.compute_status()
            await self.hub.publish(user_id, "alert", resolved.model_dump(mode="json"))

    async def _get_alert_by_id(self, alert_id: int) -> Optional[dict]:
        res = await self.db.execute(
            text(f"SELECT {ALERT_COLUMNS} FROM alerts a WHERE a.id = :id"),
            {"id": alert_id},
        )
        row = res.mappings().first()
        return dict(row) if row else None

    async def list(
        self,
        user_id: int,
        device_id: Optional[str] = None,
        status_filter: Optional[str] = None,
        limit: int = 50,
    ) -> List[Alert]:
        where = ["d.user_id = :uid"]
        params: Dict[str, Any] = {"uid": user_id, "limit": limit}

        if device_id:
            where.append("a.device_id = :device_id")
            params["device_id"] = device_id

        if status_filter == "open":
            where.append("a.resolved_at IS NULL")
        elif status_filter == "resolved":
            where.append("a.resolved_at IS NOT NULL")
        elif status_filter not in (None, "", "all"):
            raise AppError.invalid("status", "status must be open, resolved or all")

        if limit <= 0 or limit > 200:
            raise AppError.invalid("limit", "limit must be between 1 and 200")

        sql_query = f"""
            SELECT {ALERT_COLUMNS}
            FROM alerts a JOIN devices d ON d.device_id = a.device_id
            WHERE {" AND ".join(where)}
            ORDER BY (a.resolved_at IS NULL) DESC, a.opened_at DESC
            LIMIT :limit
        """
        res = await self.db.execute(text(sql_query), params)
        alerts: List[Alert] = []
        for row in res.mappings().all():
            a = Alert(**row)
            a.compute_status()
            alerts.append(a)
        return alerts

    async def count_open(self, device_id: str) -> int:
        res = await self.db.execute(
            text("SELECT COUNT(*) FROM alerts WHERE device_id = :device_id AND resolved_at IS NULL"),
            {"device_id": device_id},
        )
        return res.scalar() or 0

    async def acknowledge(self, user_id: int, alert_id: int) -> Alert:
        # Check ownership
        res = await self.db.execute(
            text("SELECT 1 FROM alerts a JOIN devices d ON d.device_id = a.device_id WHERE a.id = :aid AND d.user_id = :uid"),
            {"aid": alert_id, "uid": user_id},
        )
        if not res.scalar():
            raise AppError.not_found("alert not found")

        now = utc_now()
        await self.db.execute(
            text("UPDATE alerts SET acknowledged_at = COALESCE(acknowledged_at, :now) WHERE id = :aid"),
            {"now": now, "aid": alert_id},
        )
        await self.db.commit()

        alert_dict = await self._get_alert_by_id(alert_id)
        if not alert_dict:
            raise AppError.not_found("alert not found")
        a = Alert(**alert_dict)
        a.compute_status()
        return a

