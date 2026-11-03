from __future__ import annotations

from typing import Annotated
from fastapi import APIRouter, Depends, Request, Response, status
from sqlalchemy.ext.asyncio import AsyncSession
from app.api.deps import get_current_user_id
from app.db.session import get_db
from app.schemas.auth import (
    SignupInput,
    LoginInput,
    RefreshInput,
    ChangePasswordInput,
    DeleteAccountInput,
    Session,
    User,
)
from app.services.auth_service import AuthService

router = APIRouter(tags=["Auth"])


@router.post("/api/auth/signup", response_model=Session, status_code=status.HTTP_201_CREATED)
async def signup(
    in_data: SignupInput,
    request: Request,
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Session:
    limiter = getattr(request.app.state, "rate_limiter", None)
    if limiter:
        await limiter.check(request)
    svc = AuthService(db)
    return await svc.signup(in_data)


@router.post("/api/auth/login", response_model=Session, status_code=status.HTTP_200_OK)
async def login(
    in_data: LoginInput,
    request: Request,
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Session:
    limiter = getattr(request.app.state, "rate_limiter", None)
    if limiter:
        await limiter.check(request)
    svc = AuthService(db)
    return await svc.login(in_data)


@router.post("/api/auth/refresh", response_model=Session, status_code=status.HTTP_200_OK)
async def refresh(
    in_data: RefreshInput,
    request: Request,
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Session:
    limiter = getattr(request.app.state, "rate_limiter", None)
    if limiter:
        await limiter.check(request)
    svc = AuthService(db)
    return await svc.refresh(in_data)


@router.post("/api/auth/logout", status_code=status.HTTP_204_NO_CONTENT)
async def logout(
    in_data: RefreshInput,
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Response:
    svc = AuthService(db)
    await svc.logout(in_data)
    return Response(status_code=status.HTTP_204_NO_CONTENT)


@router.get("/api/me", response_model=User, status_code=status.HTTP_200_OK)
async def get_me(
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> User:
    svc = AuthService(db)
    return await svc.me(user_id)


@router.post("/api/me/password", response_model=Session, status_code=status.HTTP_200_OK)
async def change_password(
    in_data: ChangePasswordInput,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Session:
    svc = AuthService(db)
    return await svc.change_password(user_id, in_data)


@router.delete("/api/me", status_code=status.HTTP_204_NO_CONTENT)
async def delete_account(
    in_data: DeleteAccountInput,
    user_id: Annotated[int, Depends(get_current_user_id)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Response:
    svc = AuthService(db)
    await svc.delete_account(user_id, in_data)
    return Response(status_code=status.HTTP_204_NO_CONTENT)

