from __future__ import annotations

import re
from datetime import datetime
from typing import Any, Optional
from pydantic import BaseModel, ConfigDict, Field, field_validator
from app.core.errors import AppError
from app.core.time import to_ist

MIN_PASSWORD_LEN = 8
MAX_PASSWORD_LEN = 72
MAX_NAME_LEN = 100
MAX_EMAIL_LEN = 254

EMAIL_REGEX = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")


def validate_password_length(password: str, field_name: str = "password") -> str:
    pw_bytes = len(password.encode("utf-8"))
    if len(password) < MIN_PASSWORD_LEN:
        raise AppError.invalid(field_name, f"password must be at least {MIN_PASSWORD_LEN} characters")
    if pw_bytes > MAX_PASSWORD_LEN:
        raise AppError.invalid(field_name, f"password must be at most {MAX_PASSWORD_LEN} bytes")
    return password


class User(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    email: str
    name: str
    created_at: datetime

    @field_validator("created_at", mode="before")
    @classmethod
    def convert_to_ist(cls, v: Any) -> Any:
        return to_ist(v) if v is not None else v


class Session(BaseModel):
    access_token: str
    access_token_expires_at: datetime
    refresh_token: str
    refresh_token_expires_at: datetime
    user: User

    @field_validator("access_token_expires_at", "refresh_token_expires_at", mode="before")
    @classmethod
    def convert_to_ist(cls, v: Any) -> Any:
        return to_ist(v) if v is not None else v


class SignupInput(BaseModel):
    email: str
    password: str
    name: Optional[str] = ""

    @field_validator("email")
    @classmethod
    def validate_email(cls, v: str) -> str:
        s = v.strip().lower()
        if not s:
            raise AppError.invalid("email", "email is required")
        if not EMAIL_REGEX.match(s) or len(s) > MAX_EMAIL_LEN:
            raise AppError.invalid("email", "email address is not valid")
        return s

    @field_validator("password")
    @classmethod
    def validate_pw(cls, v: str) -> str:
        return validate_password_length(v, "password")

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: Optional[str]) -> str:
        if v is None:
            return ""
        s = v.strip()
        if len(s) > MAX_NAME_LEN:
            raise AppError.invalid("name", f"name must be at most {MAX_NAME_LEN} characters")
        return s


class LoginInput(BaseModel):
    email: str
    password: str

    @field_validator("email")
    @classmethod
    def clean_email(cls, v: str) -> str:
        s = v.strip().lower()
        if not s:
            raise AppError.invalid("", "email and password are required")
        return s

    @field_validator("password")
    @classmethod
    def clean_password(cls, v: str) -> str:
        if not v:
            raise AppError.invalid("", "email and password are required")
        return v


class RefreshInput(BaseModel):
    refresh_token: str


class ChangePasswordInput(BaseModel):
    current_password: str
    new_password: str

    @field_validator("new_password")
    @classmethod
    def validate_new_pw(cls, v: str) -> str:
        return validate_password_length(v, "new_password")


class DeleteAccountInput(BaseModel):
    password: str

