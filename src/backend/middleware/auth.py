"""
Authentication middleware for ISP Health Checker API.
Implements API key authentication with database validation.
"""
import os
import secrets
from datetime import datetime
from typing import Optional
from fastapi import HTTPException, Security, Depends, status
from fastapi.security import APIKeyHeader
from pydantic import BaseModel

from ..db import database, api_keys, users

# API Key header configuration
API_KEY_HEADER = APIKeyHeader(name="Authorization", auto_error=False)

# Environment variable for bypassing auth in development
DISABLE_AUTH = os.getenv("DISABLE_AUTH", "false").lower() == "true"

# Development API key (only used when DISABLE_AUTH is true)
DEV_API_KEY = "isp-checker-dev-key-12345"


class AuthenticatedUser(BaseModel):
    """Authenticated user information."""
    user_id: int
    username: str
    email: str
    api_key_id: int
    api_key_name: str


async def verify_api_key(api_key_header: Optional[str] = Security(API_KEY_HEADER)) -> Optional[AuthenticatedUser]:
    """
    Verify the API key from the Authorization header.

    Expects format: "Bearer <api_key>" or just "<api_key>"
    """
    if DISABLE_AUTH:
        # Return a mock user for development
        return AuthenticatedUser(
            user_id=1,
            username="dev",
            email="dev@localhost",
            api_key_id=1,
            api_key_name="Development Key"
        )

    if not api_key_header:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing API key. Provide 'Authorization: Bearer <api_key>' header.",
            headers={"WWW-Authenticate": "Bearer"},
        )

    # Extract the key from "Bearer <key>" format
    if api_key_header.startswith("Bearer "):
        api_key = api_key_header[7:]
    else:
        api_key = api_key_header

    if not api_key:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid API key format.",
            headers={"WWW-Authenticate": "Bearer"},
        )

    # Query the database for the API key
    query = api_keys.select().where(
        api_keys.c.key == api_key,
        api_keys.c.is_active == True
    )

    key_record = await database.fetch_one(query)

    if not key_record:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or inactive API key.",
            headers={"WWW-Authenticate": "Bearer"},
        )

    # Check if the key has expired
    if key_record["expires_at"] and key_record["expires_at"] < datetime.utcnow():
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="API key has expired.",
            headers={"WWW-Authenticate": "Bearer"},
        )

    # Get the associated user
    user_query = users.select().where(
        users.c.id == key_record["user_id"],
        users.c.is_active == True
    )

    user_record = await database.fetch_one(user_query)

    if not user_record:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User account is inactive or deleted.",
            headers={"WWW-Authenticate": "Bearer"},
        )

    return AuthenticatedUser(
        user_id=user_record["id"],
        username=user_record["username"],
        email=user_record["email"],
        api_key_id=key_record["id"],
        api_key_name=key_record["name"]
    )


async def get_current_user(
    user: AuthenticatedUser = Depends(verify_api_key)
) -> AuthenticatedUser:
    """Dependency to get the current authenticated user."""
    return user


def generate_api_key() -> str:
    """Generate a secure random API key."""
    return secrets.token_urlsafe(32)


class APIKeyAuth:
    """
    Class-based authentication for use as a dependency.
    Allows optional authentication where some endpoints may work without auth.
    """

    def __init__(self, required: bool = True):
        self.required = required

    async def __call__(
        self,
        api_key_header: Optional[str] = Security(API_KEY_HEADER)
    ) -> Optional[AuthenticatedUser]:
        """Validate API key and return user."""
        if not api_key_header and not self.required:
            return None

        return await verify_api_key(api_key_header)
