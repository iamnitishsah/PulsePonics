from __future__ import annotations

from datetime import datetime, timezone, timedelta
from typing import Any, Optional

# Indian Standard Time (IST, UTC+05:30)
try:
    from zoneinfo import ZoneInfo
    IST = ZoneInfo("Asia/Kolkata")
except Exception:
    IST = timezone(timedelta(hours=5, minutes=30), name="IST")

UTC = timezone.utc


def utc_now() -> datetime:
    """Current UTC datetime with tzinfo=timezone.utc."""
    return datetime.now(UTC)


def ist_now() -> datetime:
    """Current IST datetime with tzinfo=IST."""
    return datetime.now(IST)


def to_utc(dt: Any) -> Optional[datetime]:
    """
    Ensure datetime is in UTC for database storage / queries.
    If naive (e.g. from MySQL), assumes it represents UTC.
    If string, parses ISO format first.
    """
    if dt is None:
        return None
    if isinstance(dt, str):
        s = dt.replace("Z", "+00:00")
        dt = datetime.fromisoformat(s)
    if not isinstance(dt, datetime):
        return None
    if dt.tzinfo is None:
        return dt.replace(tzinfo=UTC)
    return dt.astimezone(UTC)


def to_ist(dt: Any) -> Optional[datetime]:
    """
    Convert datetime to Indian Standard Time (IST, UTC+05:30).
    If naive (from MySQL DATETIME columns with session +00:00), assumes it represents UTC,
    attaches UTC, then converts to IST (+05:30).
    If string, parses ISO format first.
    """
    if dt is None:
        return None
    if isinstance(dt, str):
        s = dt.replace("Z", "+00:00")
        dt = datetime.fromisoformat(s)
    if not isinstance(dt, datetime):
        return None
    if dt.tzinfo is None:
        # Naive datetime from MySQL (which is stored in UTC)
        dt = dt.replace(tzinfo=UTC)
    return dt.astimezone(IST)
