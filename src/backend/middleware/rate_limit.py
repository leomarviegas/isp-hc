"""
Rate limiting middleware for ISP Health Checker API.
Implements token bucket algorithm with Redis-compatible in-memory fallback.
"""
import asyncio
import time
import os
from collections import defaultdict
from dataclasses import dataclass, field
from typing import Optional, Callable, Dict
from functools import wraps

from fastapi import HTTPException, Request, status
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import Response


@dataclass
class RateLimitConfig:
    """Configuration for rate limiting."""
    requests_per_minute: int = 60
    requests_per_hour: int = 1000
    burst_size: int = 10


@dataclass
class TokenBucket:
    """Token bucket for rate limiting."""
    tokens: float
    last_update: float
    rate: float  # tokens per second
    capacity: int

    def consume(self, tokens: int = 1) -> bool:
        """Try to consume tokens. Returns True if successful."""
        now = time.time()
        elapsed = now - self.last_update

        # Refill tokens based on elapsed time
        self.tokens = min(self.capacity, self.tokens + elapsed * self.rate)
        self.last_update = now

        if self.tokens >= tokens:
            self.tokens -= tokens
            return True
        return False

    @property
    def retry_after(self) -> int:
        """Calculate seconds until a token is available."""
        if self.tokens >= 1:
            return 0
        tokens_needed = 1 - self.tokens
        return int(tokens_needed / self.rate) + 1


class InMemoryRateLimiter:
    """
    In-memory rate limiter using token bucket algorithm.
    For production, consider using Redis for distributed rate limiting.
    """

    def __init__(self, config: RateLimitConfig = None):
        self.config = config or RateLimitConfig()
        self._buckets: Dict[str, TokenBucket] = {}
        self._lock = asyncio.Lock()

        # Calculate rate (tokens per second) from requests per minute
        self._rate = self.config.requests_per_minute / 60.0
        self._capacity = self.config.burst_size

    def _get_key(self, identifier: str, endpoint: str = "") -> str:
        """Generate a unique key for the rate limit bucket."""
        return f"{identifier}:{endpoint}" if endpoint else identifier

    async def is_allowed(self, identifier: str, endpoint: str = "") -> tuple[bool, int]:
        """
        Check if request is allowed under rate limit.

        Returns:
            Tuple of (is_allowed, retry_after_seconds)
        """
        key = self._get_key(identifier, endpoint)

        async with self._lock:
            if key not in self._buckets:
                self._buckets[key] = TokenBucket(
                    tokens=self._capacity,
                    last_update=time.time(),
                    rate=self._rate,
                    capacity=self._capacity
                )

            bucket = self._buckets[key]
            allowed = bucket.consume(1)
            retry_after = 0 if allowed else bucket.retry_after

            return allowed, retry_after

    async def cleanup_old_buckets(self, max_age_seconds: int = 3600):
        """Remove stale buckets to prevent memory leaks."""
        now = time.time()
        async with self._lock:
            stale_keys = [
                key for key, bucket in self._buckets.items()
                if now - bucket.last_update > max_age_seconds
            ]
            for key in stale_keys:
                del self._buckets[key]


# Global rate limiter instance
_rate_limiter: Optional[InMemoryRateLimiter] = None


def get_rate_limiter() -> InMemoryRateLimiter:
    """Get or create the global rate limiter."""
    global _rate_limiter
    if _rate_limiter is None:
        config = RateLimitConfig(
            requests_per_minute=int(os.getenv("RATE_LIMIT_PER_MINUTE", "60")),
            requests_per_hour=int(os.getenv("RATE_LIMIT_PER_HOUR", "1000")),
            burst_size=int(os.getenv("RATE_LIMIT_BURST", "10")),
        )
        _rate_limiter = InMemoryRateLimiter(config)
    return _rate_limiter


def get_client_identifier(request: Request) -> str:
    """
    Extract a unique identifier for the client.
    Uses API key if available, otherwise falls back to IP address.
    """
    # Try to get API key from header
    auth_header = request.headers.get("Authorization", "")
    if auth_header.startswith("Bearer "):
        return f"key:{auth_header[7:]}"

    # Fall back to IP address
    # Check for proxy headers
    forwarded_for = request.headers.get("X-Forwarded-For")
    if forwarded_for:
        # Take the first IP in the chain
        return f"ip:{forwarded_for.split(',')[0].strip()}"

    real_ip = request.headers.get("X-Real-IP")
    if real_ip:
        return f"ip:{real_ip}"

    # Use direct client IP
    client = request.client
    if client:
        return f"ip:{client.host}"

    return "ip:unknown"


class RateLimitMiddleware(BaseHTTPMiddleware):
    """
    ASGI middleware for global rate limiting.
    """

    def __init__(self, app, config: RateLimitConfig = None):
        super().__init__(app)
        self.limiter = InMemoryRateLimiter(config) if config else get_rate_limiter()

    async def dispatch(self, request: Request, call_next) -> Response:
        """Process the request and apply rate limiting."""
        # Skip rate limiting for certain paths
        if request.url.path in ["/", "/docs", "/openapi.json", "/health", "/metrics"]:
            return await call_next(request)

        identifier = get_client_identifier(request)
        endpoint = request.url.path

        allowed, retry_after = await self.limiter.is_allowed(identifier, endpoint)

        if not allowed:
            return Response(
                content='{"detail": "Rate limit exceeded. Please retry later."}',
                status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                headers={
                    "Content-Type": "application/json",
                    "Retry-After": str(retry_after),
                    "X-RateLimit-Reset": str(int(time.time()) + retry_after),
                },
            )

        response = await call_next(request)

        # Add rate limit headers to response
        response.headers["X-RateLimit-Limit"] = str(self.limiter.config.requests_per_minute)

        return response


def rate_limit(requests_per_minute: int = 60):
    """
    Decorator for endpoint-specific rate limiting.

    Usage:
        @router.get("/resource")
        @rate_limit(requests_per_minute=10)
        async def get_resource():
            ...
    """
    def decorator(func: Callable):
        @wraps(func)
        async def wrapper(*args, request: Request = None, **kwargs):
            if request is None:
                # Try to find request in kwargs or args
                for arg in args:
                    if isinstance(arg, Request):
                        request = arg
                        break

            if request:
                limiter = get_rate_limiter()
                identifier = get_client_identifier(request)
                endpoint = f"{func.__module__}.{func.__name__}"

                allowed, retry_after = await limiter.is_allowed(identifier, endpoint)

                if not allowed:
                    raise HTTPException(
                        status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                        detail="Rate limit exceeded for this endpoint.",
                        headers={"Retry-After": str(retry_after)},
                    )

            return await func(*args, **kwargs)

        return wrapper
    return decorator
