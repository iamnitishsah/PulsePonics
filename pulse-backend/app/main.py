from __future__ import annotations

import asyncio
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI
from app.core.config import get_settings
from app.core.errors import register_error_handlers
from app.core.middleware import (
    CORSAndLoggingMiddleware,
    OriginPolicy,
    RateLimiter,
)
from app.db.session import wait_and_connect, close_engine, get_engine
from app.db.migrations import run_migrations
from app.services.realtime_hub import get_realtime_hub
from app.mqtt.subscriber import MQTTSubscriber

from app.api.routes import (
    auth,
    devices,
    readings,
    alerts,
    health,
    ws,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("pulse-backend")


@asynccontextmanager
async def lifespan(app: FastAPI):
    cfg = get_settings()
    logger.info("starting PulsePonics backend (port %d)", cfg.server_port)

    # 1. Connect and migrate DB
    await wait_and_connect(timeout_seconds=60.0)
    engine = get_engine()
    await run_migrations(engine)

    # 2. Start RateLimiter cleanup loop
    app.state.rate_limiter.start_cleanup()

    # 3. Start MQTT subscriber
    loop = asyncio.get_running_loop()
    mqtt_sub = MQTTSubscriber(cfg)
    mqtt_sub.start(loop)
    app.state.mqtt_subscriber = mqtt_sub

    logger.info("startup complete, ready for traffic")
    yield

    # Shutdown sequence
    logger.info("shutting down backend...")
    mqtt_sub.stop()
    app.state.rate_limiter.stop_cleanup()

    hub = get_realtime_hub()
    await hub.close_all()

    await close_engine()
    logger.info("shutdown complete")


def create_app() -> FastAPI:
    cfg = get_settings()

    app = FastAPI(
        title="PulsePonics API",
        version="1.0.0",
        lifespan=lifespan,
    )

    # Attach shared rate limiter
    app.state.rate_limiter = RateLimiter(
        refill_period=6.0,
        burst=10,
        trust_proxy=cfg.trust_proxy_headers,
    )

    # Register error handlers
    register_error_handlers(app)

    # Add custom CORS & Request logging middleware
    origin_policy = OriginPolicy(cfg.cors_origins)
    app.add_middleware(CORSAndLoggingMiddleware, origin_policy=origin_policy)

    # Include API Routers
    app.include_router(health.router)
    app.include_router(auth.router)
    app.include_router(devices.router)
    app.include_router(readings.router)
    app.include_router(alerts.router)
    app.include_router(ws.router)

    return app


app = create_app()


if __name__ == "__main__":
    import uvicorn

    cfg = get_settings()
    uvicorn.run("app.main:app", host="0.0.0.0", port=cfg.server_port, reload=False)

