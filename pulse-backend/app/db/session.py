from __future__ import annotations

import logging
import time
from typing import AsyncGenerator
from sqlalchemy import text
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from app.core.config import get_settings

logger = logging.getLogger("pulse-backend")

_engine: AsyncEngine | None = None
_session_maker: async_sessionmaker[AsyncSession] | None = None


def get_engine() -> AsyncEngine:
    global _engine
    if _engine is None:
        cfg = get_settings()
        _engine = create_async_engine(
            cfg.async_database_url,
            pool_size=10,
            max_overflow=5,
            pool_recycle=300,
            pool_pre_ping=True,
            connect_args={"init_command": "SET time_zone = '+00:00'"},
        )
    return _engine


def get_session_maker() -> async_sessionmaker[AsyncSession]:
    global _session_maker
    if _session_maker is None:
        engine = get_engine()
        _session_maker = async_sessionmaker(
            bind=engine,
            class_=AsyncSession,
            expire_on_commit=False,
            autocommit=False,
            autoflush=False,
        )
    return _session_maker


async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """Dependency that yields an async database session."""
    session_maker = get_session_maker()
    async with session_maker() as session:
        try:
            yield session
        except Exception:
            await session.rollback()
            raise


async def wait_and_connect(timeout_seconds: float = 60.0) -> None:
    """Wait for MySQL database to become responsive."""
    engine = get_engine()
    deadline = time.time() + timeout_seconds
    while True:
        try:
            async with engine.connect() as conn:
                await conn.execute(text("SELECT 1"))
                logger.info("connected to MySQL (session time_zone: +00:00)")
                return
        except Exception as e:
            if time.time() > deadline:
                raise RuntimeError(f"Database ping failed after {timeout_seconds}s: {e}") from e
            logger.warning("database not ready (%s), retrying in 2s...", e)
            import asyncio
            await asyncio.sleep(2)


async def close_engine() -> None:
    global _engine, _session_maker
    if _engine:
        await _engine.dispose()
        _engine = None
        _session_maker = None

