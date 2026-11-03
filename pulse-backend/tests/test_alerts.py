from __future__ import annotations

import pytest
from app.core.errors import AppError
from app.schemas.alerts import Threshold, Alert, default_thresholds
from app.services.alert_service import decide, Action, merge_thresholds


def test_decide():
    ph_band = Threshold(metric="ph", min=5.5, max=6.5, enabled=True)
    open_low = Alert(
        id=1,
        device_id="esp32-hydro-01",
        metric="ph",
        kind="low",
        threshold_value=5.5,
        trigger_value=5.2,
        latest_value=5.2,
        opened_at="2026-10-24T08:00:00Z",
    )
    open_high = Alert(
        id=2,
        device_id="esp32-hydro-01",
        metric="ph",
        kind="high",
        threshold_value=6.5,
        trigger_value=6.9,
        latest_value=6.9,
        opened_at="2026-10-24T08:00:00Z",
    )

    cases = [
        ("in band, nothing open", ph_band, 6.0, None, Action.NONE, ""),
        ("below min opens low", ph_band, 5.2, None, Action.OPEN, "low"),
        ("above max opens high", ph_band, 6.9, None, Action.OPEN, "high"),
        ("still low updates", ph_band, 5.3, open_low, Action.UPDATE, "low"),
        ("just back in band stays open (hysteresis)", ph_band, 5.52, open_low, Action.UPDATE, "low"),
        ("comfortably back resolves low", ph_band, 5.6, open_low, Action.RESOLVE, ""),
        ("comfortably back resolves high", ph_band, 6.4, open_high, Action.RESOLVE, ""),
        ("flip from low to high updates kind", ph_band, 7.0, open_low, Action.UPDATE, "high"),
        ("disabled resolves open alert", Threshold(metric="ph", enabled=False), 9.0, open_low, Action.RESOLVE, ""),
        ("disabled with nothing open", Threshold(metric="ph", enabled=False), 9.0, None, Action.NONE, ""),
        ("max-only threshold", Threshold(metric="water_temp_c", max=24.0, enabled=True), 25.0, None, Action.OPEN, "high"),
        ("max-only resolves exactly at limit", Threshold(metric="water_temp_c", max=24.0, enabled=True), 24.0, open_high, Action.RESOLVE, ""),
    ]

    for name, t, value, open_a, want_action, want_kind in cases:
        d = decide(t, value, open_a)
        assert d.action == want_action, f"{name}: action {d.action} != {want_action}"
        if want_kind:
            assert d.kind == want_kind, f"{name}: kind {d.kind} != {want_kind}"


def test_validate_threshold():
    bad = [
        Threshold(metric="co2", min=1.0, enabled=True),
        Threshold(metric="ph", enabled=True),
        Threshold(metric="ph", min=7.0, max=6.0, enabled=True),
        Threshold(metric="ph", max=15.0, enabled=True),
    ]
    for th in bad:
        with pytest.raises(AppError):
            th.validate_bounds()

    for th in default_thresholds():
        th.validate_bounds()


def test_merge_uses_stored_over_defaults():
    got = merge_thresholds([Threshold(metric="ph", min=6.0, max=7.0, enabled=True)])
    assert len(got) == 6
    ph = next(t for t in got if t.metric == "ph")
    ec = next(t for t in got if t.metric == "ec_ms_cm")
    assert ph.min == 6.0
    assert ph.max == 7.0
    assert ec.min == 1.2

