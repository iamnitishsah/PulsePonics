from __future__ import annotations

from datetime import datetime, timezone, timedelta
import pytest
from app.core.errors import AppError
from app.schemas.readings import ReadingInput
from app.services.reading_service import bucket_for
from app.mqtt.subscriber import device_from_topic

TEST_NOW = datetime(2026, 10, 24, 8, 20, 0, tzinfo=timezone.utc)


def firmware_payload() -> ReadingInput:
    return ReadingInput(
        device_id="esp32-hydro-01",
        water_temp_c=28.44,
        recorded_at=TEST_NOW,
    )


def test_validate_accepts_firmware_payload():
    payload = firmware_payload()
    payload.validate_payload(TEST_NOW)


def test_validate_rejects_bad_payloads():
    cases = [
        ("missing device", {"device_id": "", "water_temp_c": 20.0}),
        ("topic chars", {"device_id": "esp32/01", "water_temp_c": 20.0}),
        ("no sensor values", {"device_id": "esp32-01"}),
        ("ph too high", {"device_id": "esp32-01", "ph": 14.5}),
        ("ec negative", {"device_id": "esp32-01", "ec_ms_cm": -0.1}),
        ("water too cold", {"device_id": "esp32-01", "water_temp_c": -11.0}),
        ("air too hot", {"device_id": "esp32-01", "air_temp_c": 61.0}),
        ("humidity > 100", {"device_id": "esp32-01", "humidity_pct": 101.0}),
        ("pressure too low", {"device_id": "esp32-01", "pressure_hpa": 200.0}),
        ("future timestamp", {"device_id": "esp32-01", "water_temp_c": 20.0, "recorded_at": TEST_NOW + timedelta(hours=1)}),
    ]

    for name, data in cases:
        payload = ReadingInput(**data)
        with pytest.raises(AppError):
            payload.validate_payload(TEST_NOW)


def test_validate_normalises_time():
    payload = ReadingInput(device_id="esp32-hydro-01", water_temp_c=20.0)
    payload.validate_payload(TEST_NOW)
    assert payload.recorded_at == TEST_NOW


def test_bucket_for_never_exceeds_max_points():
    cases = [
        (timedelta(hours=1), 12),
        (timedelta(days=1), 500),
        (timedelta(days=7), 500),
        (timedelta(days=31), 10),
        (timedelta(seconds=90), 10),
    ]
    for dur, max_pts in cases:
        b = bucket_for(dur, max_pts)
        buckets = int(dur.total_seconds()) // b + 1
        assert buckets <= max_pts, f"duration {dur} max {max_pts}: bucket {b}s gives {buckets} points"


def test_device_from_topic():
    assert device_from_topic("hydroponics/esp32-hydro-01/telemetry") == "esp32-hydro-01"
    assert device_from_topic("hydroponics/telemetry") == ""

