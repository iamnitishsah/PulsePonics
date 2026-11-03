from __future__ import annotations

import base64
import hashlib
import secrets
from datetime import datetime, timezone, timedelta
from typing import Tuple
import bcrypt
import jwt
from app.core.errors import AppError

JWT_ISSUER = "pulseponics"
JWT_ALGORITHM = "HS256"
API_KEY_PREFIX = "ppk_"
SHOWN_KEY_PREFIX_LEN = 10

# Pre-computed dummy hash so failed logins take equal time (timing attack prevention)
DUMMY_HASH: bytes = bcrypt.hashpw(b"pulseponics-timing-equaliser", bcrypt.gensalt(10))


def hash_password(password: str) -> str:
    """Hash password using bcrypt."""
    pw_bytes = password.encode("utf-8")
    salt = bcrypt.gensalt(10)
    return bcrypt.hashpw(pw_bytes, salt).decode("utf-8")


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """Verify password against bcrypt hash."""
    try:
        return bcrypt.checkpw(plain_password.encode("utf-8"), hashed_password.encode("utf-8"))
    except Exception:
        return False


def verify_dummy_password(plain_password: str) -> None:
    """Run bcrypt against dummy hash to equalize response timing."""
    try:
        bcrypt.checkpw(plain_password.encode("utf-8"), DUMMY_HASH)
    except Exception:
        pass


def issue_access_token(user_id: int, secret: str, ttl: timedelta, now: datetime | None = None) -> Tuple[str, datetime]:
    """Issue short-lived HS256 JWT access token."""
    if now is None:
        now = datetime.now(timezone.utc)
    else:
        now = now.astimezone(timezone.utc)
    expires = now + ttl

    payload = {
        "iss": JWT_ISSUER,
        "sub": str(user_id),
        "iat": int(now.timestamp()),
        "exp": int(expires.timestamp()),
    }
    token = jwt.encode(payload, secret, algorithm=JWT_ALGORITHM)
    return token, expires


def parse_access_token(token: str, secret: str) -> int:
    """Parse and validate JWT access token, returning subject user ID."""
    try:
        payload = jwt.decode(
            token,
            secret,
            algorithms=[JWT_ALGORITHM],
            issuer=JWT_ISSUER,
            options={"require": ["exp", "iss", "sub"]},
        )
        user_id_str = payload.get("sub")
        if not user_id_str:
            raise AppError.unauthorized("invalid token subject")
        user_id = int(user_id_str)
        if user_id <= 0:
            raise AppError.unauthorized("invalid token subject")
        return user_id
    except jwt.ExpiredSignatureError:
        raise AppError.unauthorized("invalid or expired access token")
    except (jwt.PyJWTError, ValueError):
        raise AppError.unauthorized("invalid or expired access token")


def hash_token(token: str) -> str:
    """Compute SHA-256 hex digest of a token."""
    return hashlib.sha256(token.encode("utf-8")).hexdigest()


def new_opaque_token(prefix: str = "") -> Tuple[str, str]:
    """Generate random URL-safe 32-byte token and its SHA-256 hash."""
    random_bytes = secrets.token_bytes(32)
    # URL-safe base64 without padding
    b64 = base64.urlsafe_b64encode(random_bytes).decode("ascii").rstrip("=")
    token = f"{prefix}{b64}"
    return token, hash_token(token)


def new_api_key() -> Tuple[str, str, str]:
    """Generate device API key (ppk_...), its SHA-256 hash, and 10-char prefix."""
    key, h = new_opaque_token(API_KEY_PREFIX)
    prefix = key[:SHOWN_KEY_PREFIX_LEN]
    return key, h, prefix


def hash_api_key(key: str) -> str:
    return hash_token(key)

