from __future__ import annotations

from datetime import datetime, timezone, timedelta
import pytest
from app.core.errors import AppError
from app.core.security import (
    issue_access_token,
    parse_access_token,
    hash_password,
    verify_password,
    new_opaque_token,
    hash_token,
)
from app.schemas.auth import SignupInput, validate_password_length

SECRET = "test-secret-that-is-at-least-32-chars!!"


def test_token_round_trip():
    now = datetime.now(timezone.utc)
    token, exp = issue_access_token(42, SECRET, timedelta(minutes=15), now)
    assert exp > now
    user_id = parse_access_token(token, SECRET)
    assert user_id == 42


def test_token_rejects_expired_and_foreign():
    now = datetime.now(timezone.utc)
    # Expired token
    token, _ = issue_access_token(1, SECRET, timedelta(seconds=-10), now)
    with pytest.raises(AppError):
        parse_access_token(token, SECRET)

    # Foreign secret
    other_secret = "another-secret-that-is-also-32-chars!!"
    foreign_token, _ = issue_access_token(1, other_secret, timedelta(minutes=15), now)
    with pytest.raises(AppError):
        parse_access_token(foreign_token, SECRET)

    # Malformed token
    with pytest.raises(AppError):
        parse_access_token("not-a-jwt", SECRET)


def test_password_hashing():
    pw = "hydro-pass-1"
    hashed = hash_password(pw)
    assert verify_password(pw, hashed) is True
    assert verify_password("wrong-pw", hashed) is False


def test_opaque_tokens():
    raw, h = new_opaque_token()
    assert len(raw) > 30
    assert hash_token(raw) == h


def test_signup_validation():
    bad = [
        {"email": "not-an-email", "password": "long-enough"},
        {"email": "Name <a@b.co>", "password": "long-enough"},
        {"email": "a@b.co", "password": "short"},
    ]
    for b in bad:
        with pytest.raises(AppError):
            SignupInput(**b)

