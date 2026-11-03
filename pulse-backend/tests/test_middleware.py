from __future__ import annotations

import pytest
from starlette.requests import Request
from app.core.middleware import OriginPolicy, get_client_ip


def test_origin_policy():
    p = OriginPolicy(["http://localhost:5173", "https://pulseponics*.vercel.app"])
    cases = {
        "http://localhost:5173": True,
        "https://pulseponics.vercel.app": True,
        "https://pulseponics-git-main.vercel.app": True,
        "https://evil.vercel.app": False,
        "https://pulseponics.vercel.app.evil.com": False,
        "http://localhost:3000": False,
        "": False,
    }
    for origin, want in cases.items():
        assert p.is_allowed(origin) == want, f"Allowed({origin}) was not {want}"


def test_client_ip():
    scope_direct = {
        "type": "http",
        "client": ("172.18.0.5", 41234),
        "headers": [(b"x-forwarded-for", b"6.6.6.6, 203.0.113.9")],
    }
    req_direct = Request(scope_direct)
    assert get_client_ip(req_direct, trust_proxy=False) == "172.18.0.5"
    assert get_client_ip(req_direct, trust_proxy=True) == "203.0.113.9"

