from __future__ import annotations

from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List, Optional
from pydantic import BaseModel, ConfigDict, Field, field_validator
from app.core.errors import AppError
from app.core.time import to_ist, to_utc, utc_now
from app.schemas.devices import validate_device_id

MAX_CLOCK_SKEW = timedelta(minutes=5)


class Metric(BaseModel):
    key: str
    label: str
    unit: str
    min: float
    max: float


METRICS: List[Metric] = [
    Metric(key="ph", label="pH", unit="", min=0.0, max=14.0),
    Metric(key="ec_ms_cm", label="EC", unit="mS/cm", min=0.0, max=20.0),
    Metric(key="water_temp_c", label="Water temperature", unit="°C", min=-10.0, max=60.0),
    Metric(key="air_temp_c", label="Air temperature", unit="°C", min=-20.0, max=60.0),
    Metric(key="humidity_pct", label="Humidity", unit="%", min=0.0, max=100.0),
    Metric(key="pressure_hpa", label="Pressure", unit="hPa", min=300.0, max=1100.0),
]

METRICS_BY_KEY: Dict[str, Metric] = {m.key: m for m in METRICS}


class Reading(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    device_id: str
    ph: Optional[float] = None
    ec_ms_cm: Optional[float] = None
    water_temp_c: Optional[float] = None
    air_temp_c: Optional[float] = None
    humidity_pct: Optional[float] = None
    pressure_hpa: Optional[float] = None
    recorded_at: datetime
    created_at: datetime

    @field_validator("recorded_at", "created_at", mode="before")
    @classmethod
    def convert_to_ist(cls, v: Any) -> Any:
        return to_ist(v) if v is not None else v

    def get_metric_value(self, metric: str) -> Optional[float]:
        return getattr(self, metric, None)


class ReadingInput(BaseModel):
    device_id: Optional[str] = ""
    ph: Optional[float] = None
    ec_ms_cm: Optional[float] = None
    water_temp_c: Optional[float] = None
    air_temp_c: Optional[float] = None
    humidity_pct: Optional[float] = None
    pressure_hpa: Optional[float] = None
    recorded_at: Optional[datetime] = None

    def validate_payload(self, now: datetime | None = None) -> None:
        if now is None:
            now = datetime.now(timezone.utc)
        else:
            now = now.astimezone(timezone.utc)

        if not self.device_id:
            raise AppError.invalid("device_id", "device_id is required")
        validate_device_id(self.device_id)

        has_sensor = False
        for m in METRICS:
            v = getattr(self, m.key)
            if v is not None:
                has_sensor = True
                if v < m.min or v > m.max:
                    # Format without trailing zeros if whole
                    min_str = f"{m.min:g}"
                    max_str = f"{m.max:g}"
                    raise AppError.invalid(m.key, f"{m.key} must be between {min_str} and {max_str}")

        if not has_sensor:
            raise AppError.invalid("", "at least one sensor value must be provided")

        if self.recorded_at is None:
            self.recorded_at = now
        else:
            rec = self.recorded_at.astimezone(timezone.utc)
            if rec > now + MAX_CLOCK_SKEW:
                raise AppError.invalid("recorded_at", "recorded_at is in the future; check the device clock")
            self.recorded_at = rec


class Point(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    recorded_at: datetime
    ph: Optional[float] = None
    ec_ms_cm: Optional[float] = None
    water_temp_c: Optional[float] = None
    air_temp_c: Optional[float] = None
    humidity_pct: Optional[float] = None
    pressure_hpa: Optional[float] = None

    @field_validator("recorded_at", mode="before")
    @classmethod
    def convert_to_ist(cls, v: Any) -> Any:
        return to_ist(v) if v is not None else v


class History(BaseModel):
    device_id: str
    from_: datetime = Field(alias="from")
    to: datetime
    bucket_seconds: int
    points: List[Point]

    model_config = ConfigDict(populate_by_name=True)

    @field_validator("from_", "to", mode="before")
    @classmethod
    def convert_to_ist(cls, v: Any) -> Any:
        return to_ist(v) if v is not None else v

