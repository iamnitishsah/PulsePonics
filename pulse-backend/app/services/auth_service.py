from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import Optional
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession
from app.core.config import get_settings
from app.core.errors import AppError
from app.core.security import (
    hash_password,
    verify_password,
    verify_dummy_password,
    issue_access_token,
    new_opaque_token,
    hash_token,
)
from app.schemas.auth import (
    User,
    Session,
    SignupInput,
    LoginInput,
    RefreshInput,
    ChangePasswordInput,
    DeleteAccountInput,
)

logger = logging.getLogger("pulse-backend")


class AuthService:
    def __init__(self, db: AsyncSession):
        self.db = db
        self.cfg = get_settings()

    async def get_user_by_email(self, email: str) -> Optional[dict]:
        res = await self.db.execute(
            text("SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE email = :email"),
            {"email": email},
        )
        row = res.mappings().first()
        return dict(row) if row else None

    async def get_user_by_id(self, user_id: int) -> Optional[dict]:
        res = await self.db.execute(
            text("SELECT id, email, name, password_hash, created_at, updated_at FROM users WHERE id = :id"),
            {"id": user_id},
        )
        row = res.mappings().first()
        return dict(row) if row else None

    async def signup(self, in_data: SignupInput) -> Session:
        existing = await self.get_user_by_email(in_data.email)
        if existing:
            raise AppError.conflict("an account with this email already exists")

        name = in_data.name or in_data.email.split("@")[0]
        pw_hash = hash_password(in_data.password)

        res = await self.db.execute(
            text("INSERT INTO users (email, name, password_hash) VALUES (:email, :name, :hash)"),
            {"email": in_data.email, "name": name, "hash": pw_hash},
        )
        user_id = res.lastrowid
        await self.db.commit()

        user_dict = await self.get_user_by_id(user_id)
        if not user_dict:
            raise RuntimeError("failed to read back created user")

        return await self._issue_session(User(**user_dict))

    async def login(self, in_data: LoginInput) -> Session:
        user_dict = await self.get_user_by_email(in_data.email)
        if not user_dict:
            verify_dummy_password(in_data.password)
            raise AppError.unauthorized("invalid email or password")

        if not verify_password(in_data.password, user_dict["password_hash"]):
            raise AppError.unauthorized("invalid email or password")

        return await self._issue_session(User(**user_dict))

    async def refresh(self, in_data: RefreshInput) -> Session:
        invalid_err = AppError.unauthorized("invalid or expired refresh token")
        if not in_data.refresh_token:
            raise invalid_err

        token_h = hash_token(in_data.refresh_token)
        res = await self.db.execute(
            text("SELECT id, user_id, expires_at, revoked_at, rotated FROM refresh_tokens WHERE token_hash = :hash"),
            {"hash": token_h},
        )
        row = res.mappings().first()
        if not row:
            raise invalid_err

        token_record = dict(row)
        now = datetime.now(timezone.utc)

        # Make expires_at timezone-aware for comparison if it is naive
        exp_at = token_record["expires_at"]
        if exp_at.tzinfo is None:
            exp_at = exp_at.replace(tzinfo=timezone.utc)

        if token_record["revoked_at"] is not None:
            if token_record["rotated"]:
                logger.warning(
                    "auth: reuse of rotated refresh token for user %d, revoking all sessions",
                    token_record["user_id"],
                )
                await self.db.execute(
                    text("UPDATE refresh_tokens SET revoked_at = :now WHERE user_id = :uid AND revoked_at IS NULL"),
                    {"now": now, "uid": token_record["user_id"]},
                )
                await self.db.commit()
            raise invalid_err

        if now > exp_at:
            raise invalid_err

        # Atomic rotation
        upd = await self.db.execute(
            text(
                """
                UPDATE refresh_tokens
                SET revoked_at = :now, rotated = TRUE
                WHERE id = :id AND revoked_at IS NULL
                """
            ),
            {"now": now, "id": token_record["id"]},
        )
        if upd.rowcount != 1:
            await self.db.commit()
            raise invalid_err  # Lost race to another concurrent refresh

        user_dict = await self.get_user_by_id(token_record["user_id"])
        if not user_dict:
            await self.db.commit()
            raise invalid_err

        session = await self._issue_session(User(**user_dict))
        return session

    async def logout(self, in_data: RefreshInput) -> None:
        if not in_data.refresh_token:
            return
        token_h = hash_token(in_data.refresh_token)
        now = datetime.now(timezone.utc)
        await self.db.execute(
            text(
                """
                UPDATE refresh_tokens
                SET revoked_at = :now, rotated = FALSE
                WHERE token_hash = :hash AND revoked_at IS NULL
                """
            ),
            {"now": now, "hash": token_h},
        )
        await self.db.commit()

    async def me(self, user_id: int) -> User:
        user_dict = await self.get_user_by_id(user_id)
        if not user_dict:
            raise AppError.unauthorized("account no longer exists")
        return User(**user_dict)

    async def change_password(self, user_id: int, in_data: ChangePasswordInput) -> Session:
        user_dict = await self.get_user_by_id(user_id)
        if not user_dict:
            raise AppError.unauthorized("account no longer exists")

        if not verify_password(in_data.current_password, user_dict["password_hash"]):
            raise AppError.invalid("current_password", "current password is incorrect")

        pw_hash = hash_password(in_data.new_password)
        now = datetime.now(timezone.utc)

        await self.db.execute(
            text("UPDATE users SET password_hash = :hash WHERE id = :id"),
            {"hash": pw_hash, "id": user_id},
        )
        # Revoke all other refresh tokens
        await self.db.execute(
            text("UPDATE refresh_tokens SET revoked_at = :now WHERE user_id = :uid AND revoked_at IS NULL"),
            {"now": now, "uid": user_id},
        )
        await self.db.commit()

        user_dict["password_hash"] = pw_hash
        return await self._issue_session(User(**user_dict))

    async def delete_account(self, user_id: int, in_data: DeleteAccountInput) -> None:
        user_dict = await self.get_user_by_id(user_id)
        if not user_dict:
            raise AppError.unauthorized("account no longer exists")

        if not verify_password(in_data.password, user_dict["password_hash"]):
            raise AppError.invalid("password", "password is incorrect")

        await self.db.execute(text("DELETE FROM users WHERE id = :id"), {"id": user_id})
        await self.db.commit()

    async def _issue_session(self, user: User) -> Session:
        now = datetime.now(timezone.utc)
        access_token, access_exp = issue_access_token(
            user.id, self.cfg.jwt_secret, self.cfg.access_token_ttl, now
        )
        raw_refresh, refresh_hash = new_opaque_token()
        refresh_exp = now + self.cfg.refresh_token_ttl

        await self.db.execute(
            text(
                """
                INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
                VALUES (:user_id, :token_hash, :expires_at)
                """
            ),
            {"user_id": user.id, "token_hash": refresh_hash, "expires_at": refresh_exp},
        )
        await self.db.commit()

        return Session(
            access_token=access_token,
            access_token_expires_at=access_exp,
            refresh_token=raw_refresh,
            refresh_token_expires_at=refresh_exp,
            user=user,
        )

