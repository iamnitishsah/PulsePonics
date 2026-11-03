from __future__ import annotations

import logging
from enum import Enum
from typing import Any, Optional
from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from pydantic import BaseModel

logger = logging.getLogger("pulse-backend")


class ErrorKind(str, Enum):
    INVALID = "invalid"
    UNAUTHORIZED = "unauthorized"
    FORBIDDEN = "forbidden"
    NOT_FOUND = "not_found"
    CONFLICT = "conflict"
    TOO_MANY_REQUESTS = "too_many_requests"
    INTERNAL = "internal"


class AppError(Exception):
    def __init__(self, kind: ErrorKind, message: str, field: Optional[str] = None):
        super().__init__(message)
        self.kind = kind
        self.message = message
        self.field = field

    @classmethod
    def invalid(cls, field: str, message: str) -> "AppError":
        return cls(ErrorKind.INVALID, message, field if field else None)

    @classmethod
    def unauthorized(cls, message: str) -> "AppError":
        return cls(ErrorKind.UNAUTHORIZED, message)

    @classmethod
    def forbidden(cls, message: str) -> "AppError":
        return cls(ErrorKind.FORBIDDEN, message)

    @classmethod
    def not_found(cls, message: str) -> "AppError":
        return cls(ErrorKind.NOT_FOUND, message)

    @classmethod
    def conflict(cls, message: str) -> "AppError":
        return cls(ErrorKind.CONFLICT, message)

    @classmethod
    def too_many_requests(cls, message: str) -> "AppError":
        return cls(ErrorKind.TOO_MANY_REQUESTS, message)


STATUS_MAP = {
    ErrorKind.INVALID: status.HTTP_400_BAD_REQUEST,
    ErrorKind.UNAUTHORIZED: status.HTTP_401_UNAUTHORIZED,
    ErrorKind.FORBIDDEN: status.HTTP_403_FORBIDDEN,
    ErrorKind.NOT_FOUND: status.HTTP_404_NOT_FOUND,
    ErrorKind.CONFLICT: status.HTTP_409_CONFLICT,
    ErrorKind.TOO_MANY_REQUESTS: status.HTTP_429_TOO_MANY_REQUESTS,
    ErrorKind.INTERNAL: status.HTTP_500_INTERNAL_SERVER_ERROR,
}


def error_payload(message: str, field: Optional[str] = None) -> dict[str, Any]:
    res: dict[str, Any] = {"error": message}
    if field:
        res["field"] = field
    return res


def register_error_handlers(app: FastAPI) -> None:
    @app.exception_handler(AppError)
    async def app_error_handler(request: Request, exc: AppError) -> JSONResponse:
        status_code = STATUS_MAP.get(exc.kind, status.HTTP_500_INTERNAL_SERVER_ERROR)
        return JSONResponse(
            status_code=status_code,
            content=error_payload(exc.message, exc.field),
        )

    @app.exception_handler(RequestValidationError)
    async def validation_error_handler(request: Request, exc: RequestValidationError) -> JSONResponse:
        errors = exc.errors()
        if not errors:
            return JSONResponse(
                status_code=status.HTTP_400_BAD_REQUEST,
                content=error_payload("invalid request body"),
            )
        first_error = errors[0]
        # Locate the offending field name
        loc = first_error.get("loc", [])
        field_name = str(loc[-1]) if loc and loc[-1] != "body" else None
        msg = first_error.get("msg", "invalid input")
        # Clean up default Pydantic message prefixes if present
        if msg.startswith("Value error, "):
            msg = msg[len("Value error, "):]
        return JSONResponse(
            status_code=status.HTTP_400_BAD_REQUEST,
            content=error_payload(msg, field_name),
        )

    @app.exception_handler(Exception)
    async def unhandled_exception_handler(request: Request, exc: Exception) -> JSONResponse:
        logger.exception("unhandled error in %s %s: %s", request.method, request.url.path, exc)
        return JSONResponse(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            content=error_payload("internal server error"),
        )

