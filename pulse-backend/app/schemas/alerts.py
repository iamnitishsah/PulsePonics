from __future__ import annotations

from datetime import datetime
from typing import List, Optional
from pydantic import BaseModel, ConfigDict, Field
from app.core.errors import AppError
from app.schemas.readings import METRICS, METRICS_BY_KEY


class Threshold(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    metric: str
    min: Optional[float] = Field(default=None, alias="min_value")
    max: Optional[float] = Field(default=None, alias="max_value")
    enabled: bool = True

    model_config = ConfigDict(populate_by_name=True)

    def validate_bounds(self) -> None:
        m = METRICS_BY_KEY.get(self.metric)
        if not m:
            raise AppError.invalid("metric", f"unknown metric '{self.metric}'")

        if self.enabled and self.min is None and self.max is None:
            raise AppError.invalid(self.metric, f"{m.label}: set a minimum, a maximum, or both to enable it")

        for bound in (self.min, self.max):
            if bound is not None and (bound < m.min or bound > m.max):
                min_str = f"{m.min:g}"
                max_str = f"{m.max:g}"
                raise AppError.invalid(self.metric, f"{m.label} limits must be between {min_str} and {max_str}")

        if self.min is not None and self.max is not None and self.min >= self.max:
            raise AppError.invalid(self.metric, f"{m.label}: minimum must be lower than maximum")


def default_thresholds() -> List[Threshold]:
    return [
        Threshold(metric="ph", min=5.5, max=6.5, enabled=True),
        Threshold(metric="ec_ms_cm", min=1.2, max=2.4, enabled=True),
        Threshold(metric="water_temp_c", min=18.0, max=24.0, enabled=True),
        Threshold(metric="air_temp_c", min=18.0, max=28.0, enabled=True),
        Threshold(metric="humidity_pct", min=45.0, max=75.0, enabled=True),
        Threshold(metric="pressure_hpa", min=None, max=None, enabled=False),
    ]


class ThresholdsBody(BaseModel):
    thresholds: List[Threshold]


class Alert(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    device_id: str
    metric: str
    kind: str  # "low" or "high"
    threshold_value: float
    trigger_value: float
    latest_value: float
    opened_at: datetime
    resolved_at: Optional[datetime] = None
    acknowledged_at: Optional[datetime] = None
    status: str = "open"

    def compute_status(self) -> None:
        self.status = "resolved" if self.resolved_at is not None else "open"

