from __future__ import annotations

import re
from datetime import datetime
from typing import Any, Optional
from pydantic import BaseModel, ConfigDict, field_validator
from app.core.errors import AppError

DEVICE_ID_REGEX = re.compile(r"^[A-Za-z0-9_-]{3,64}$")
MAX_NAME_LEN = 100


def validate_device_id(device_id: str) -> str:
    s = device_id.strip()
    if not DEVICE_ID_REGEX.match(s):
        raise AppError.invalid("device_id", "device_id must be 3-64 characters: letters, digits, '-' or '_'")
    return s


def clean_name(raw: Optional[str], fallback: str = "") -> str:
    s = (raw or "").strip()
    if not s:
        s = fallback
    if not s:
        raise AppError.invalid("name", "name is required")
    if len(s) > MAX_NAME_LEN:
        raise AppError.invalid("name", f"name must be at most {MAX_NAME_LEN} characters")
    return s


class Device(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    device_id: str
    name: str
    api_key_prefix: str
    created_at: datetime


class DeviceWithKey(Device):
    api_key: str


class Summary(BaseModel):
    latest: Optional[Any] = None
    open_alerts: int = 0


class DeviceWithSummary(Device):
    latest: Optional[Any] = None
    open_alerts: int = 0


class CreateDeviceInput(BaseModel):
    device_id: str
    name: Optional[str] = ""

    @field_validator("device_id")
    @classmethod
    def check_device_id(cls, v: str) -> str:
        return validate_device_id(v)


class UpdateDeviceInput(BaseModel):
    name: str

    @field_validator("name")
    @classmethod
    def check_name(cls, v: str) -> str:
        return clean_name(v)

