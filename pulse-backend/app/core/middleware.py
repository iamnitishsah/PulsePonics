from __future__ import annotations

import asyncio
import fnmatch
import logging
import time
from typing import Callable, Dict, List, Optional
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
from app.core.errors import AppError, error_payload

logger = logging.getLogger("pulse-backend")


def get_client_ip(request: Request, trust_proxy: bool) -> str:
    """Extract client IP, taking right-most X-Forwarded-For entry when trust_proxy is true."""
    if trust_proxy:
        xff = request.headers.get("x-forwarded-for")
        if xff:
            parts = [p.strip() for p in xff.split(",") if p.strip()]
            if parts:
                return parts[-1]
    if request.client and request.client.host:
        return request.client.host
    return "127.0.0.1"


class OriginPolicy:
    def __init__(self, patterns: List[str]):
        self.patterns = patterns

    def is_allowed(self, origin: str) -> bool:
        if not origin:
            return False
        for pattern in self.patterns:
            if pattern == "*" or pattern == origin:
                return True
            if "*" in pattern and fnmatch.fnmatch(origin, pattern):
                return True
        return False


class CORSAndLoggingMiddleware(BaseHTTPMiddleware):
    def __init__(self, app, origin_policy: OriginPolicy):
        super().__init__(app)
        self.origin_policy = origin_policy

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        origin = request.headers.get("origin")
        is_preflight = (
            request.method == "OPTIONS"
            and "access-control-request-method" in request.headers
        )

        start = time.perf_counter()

        if is_preflight:
            response = Response(status_code=204)
        else:
            response = await call_next(request)

        # CORS Headers
        if origin:
            response.headers.append("Vary", "Origin")
            if self.origin_policy.is_allowed(origin):
                response.headers["Access-Control-Allow-Origin"] = origin
                if is_preflight:
                    response.headers["Access-Control-Allow-Methods"] = (
                        "GET, POST, PUT, PATCH, DELETE, OPTIONS"
                    )
                    response.headers["Access-Control-Allow-Headers"] = (
                        "Authorization, Content-Type, X-Device-Key"
                    )
                    response.headers["Access-Control-Max-Age"] = "600"

        duration_ms = (time.perf_counter() - start) * 1000
        logger.info(
            "%s %s -> %d (%.2fms)",
            request.method,
            request.url.path,
            response.status_code,
            duration_ms,
        )
        return response


class TokenBucket:
    def __init__(self, rate: float, burst: int):
        self.rate = rate  # tokens added per second
        self.burst = burst
        self.tokens = float(burst)
        self.last_update = time.time()

    def allow(self) -> bool:
        now = time.time()
        elapsed = now - self.last_update
        self.last_update = now
        self.tokens = min(float(self.burst), self.tokens + elapsed * self.rate)
        if self.tokens >= 1.0:
            self.tokens -= 1.0
            return True
        return False


class RateLimiter:
    """Per-IP token bucket rate limiter for auth endpoints."""

    def __init__(self, refill_period: float = 6.0, burst: int = 10, trust_proxy: bool = False):
        self.rate = 1.0 / refill_period  # tokens per second
        self.burst = burst
        self.trust_proxy = trust_proxy
        self.visitors: Dict[str, tuple[TokenBucket, float]] = {}
        self._lock = asyncio.Lock()
        self._cleanup_task: Optional[asyncio.Task] = None

    async def allow(self, ip: str) -> bool:
        now = time.time()
        async with self._lock:
            if ip not in self.visitors:
                self.visitors[ip] = (TokenBucket(self.rate, self.burst), now)
            bucket, _ = self.visitors[ip]
            self.visitors[ip] = (bucket, now)
            return bucket.allow()

    async def check(self, request: Request) -> None:
        ip = get_client_ip(request, self.trust_proxy)
        allowed = await self.allow(ip)
        if not allowed:
            raise AppError.too_many_requests("too many requests, try again in a minute")

    async def cleanup_loop(self) -> None:
        try:
            while True:
                await asyncio.sleep(60)
                now = time.time()
                async with self._lock:
                    stale = [ip for ip, (_, last) in self.visitors.items() if now - last > 600]
                    for ip in stale:
                        del self.visitors[ip]
        except asyncio.CancelledError:
            pass

    def start_cleanup(self) -> None:
        if self._cleanup_task is None:
            self._cleanup_task = asyncio.create_task(self.cleanup_loop())

    def stop_cleanup(self) -> None:
        if self._cleanup_task:
            self._cleanup_task.cancel()
            self._cleanup_task = None

