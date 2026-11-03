from __future__ import annotations

import os
import re
import secrets
from datetime import timedelta
from typing import List
from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


def parse_duration(v: str | int | float | timedelta) -> timedelta:
    """Parse a Go-style duration string (e.g. '15m', '168h', '10s') or return timedelta."""
    if isinstance(v, timedelta):
        return v
    if isinstance(v, (int, float)):
        return timedelta(seconds=float(v))
    s = str(v).strip()
    match = re.match(r"^(\d+)\s*(s|m|h|d)$", s)
    if not match:
        # Try raw integer seconds
        try:
            return timedelta(seconds=float(s))
        except ValueError:
            raise ValueError(f"Invalid duration format: '{s}'. Expected formats like '15m', '168h', '10s'")
    val, unit = int(match.group(1)), match.group(2)
    if unit == "s":
        return timedelta(seconds=val)
    if unit == "m":
        return timedelta(minutes=val)
    if unit == "h":
        return timedelta(hours=val)
    if unit == "d":
        return timedelta(days=val)
    raise ValueError(f"Unknown duration unit in: {s}")


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    # Database
    db_user: str = Field(alias="DB_USER")
    db_password: str = Field(alias="DB_PASSWORD")
    db_host: str = Field(default="127.0.0.1", alias="DB_HOST")
    db_port: int = Field(default=3306, alias="DB_PORT")
    db_name: str = Field(alias="DB_NAME")

    # Server
    server_port: int = Field(default=8080, alias="SERVER_PORT")
    cors_allowed_origins: str = Field(default="http://localhost:5173", alias="CORS_ALLOWED_ORIGINS")
    trust_proxy_headers: bool = Field(default=False, alias="TRUST_PROXY_HEADERS")

    # MQTT
    mqtt_broker_url: str = Field(default="tcp://localhost:1883", alias="MQTT_BROKER_URL")
    mqtt_username: str = Field(alias="MQTT_USERNAME")
    mqtt_password: str = Field(alias="MQTT_PASSWORD")
    mqtt_client_id: str = Field(default="", alias="MQTT_CLIENT_ID")

    # Auth
    jwt_secret: str = Field(alias="JWT_SECRET")
    access_token_ttl_raw: str = Field(default="15m", alias="ACCESS_TOKEN_TTL")
    refresh_token_ttl_raw: str = Field(default="168h", alias="REFRESH_TOKEN_TTL")

    @field_validator("jwt_secret")
    @classmethod
    def validate_jwt_secret(cls, v: str) -> str:
        if len(v) < 32:
            raise ValueError("JWT_SECRET must be at least 32 characters (try: openssl rand -hex 32)")
        return v

    @property
    def cors_origins(self) -> List[str]:
        out = []
        for part in self.cors_allowed_origins.split(","):
            cleaned = part.strip().rstrip("/")
            if cleaned:
                out.append(cleaned)
        return out

    @property
    def access_token_ttl(self) -> timedelta:
        return parse_duration(self.access_token_ttl_raw)

    @property
    def refresh_token_ttl(self) -> timedelta:
        return parse_duration(self.refresh_token_ttl_raw)

    @property
    def client_id(self) -> str:
        if self.mqtt_client_id:
            return self.mqtt_client_id
        return f"pulse-backend-{secrets.token_hex(4)}"

    @property
    def async_database_url(self) -> str:
        # aiomysql URL format with UTC timezone session
        return (
            f"mysql+aiomysql://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/"
            f"{self.db_name}?charset=utf8mb4"
        )


settings: Settings | None = None


def get_settings() -> Settings:
    global settings
    if settings is None:
        settings = Settings()  # pyright: ignore
    return settings

