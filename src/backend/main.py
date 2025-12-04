"""
ISP Health Checker Backend API.
FastAPI application with PostgreSQL storage, authentication, and rate limiting.
"""
import os
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, RedirectResponse
from prometheus_fastapi_instrumentator import Instrumentator
import uvicorn

from .api import runs
from .db import connect_db, disconnect_db
from .middleware.rate_limit import RateLimitMiddleware, RateLimitConfig
from .workers.probe_runner import get_worker_pool

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='{"timestamp": "%(asctime)s", "level": "%(levelname)s", "service": "isp-checker-backend", "message": "%(message)s"}',
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager for startup/shutdown events."""
    # Startup
    logger.info("Starting ISP Health Checker API...")

    try:
        await connect_db()
        logger.info("Database connection established")
    except Exception as e:
        logger.error(f"Failed to connect to database: {e}")
        # Continue anyway - allows running without database for development

    yield

    # Shutdown
    logger.info("Shutting down ISP Health Checker API...")

    # Shutdown worker pool
    worker_pool = get_worker_pool()
    await worker_pool.shutdown()

    try:
        await disconnect_db()
        logger.info("Database connection closed")
    except Exception as e:
        logger.error(f"Error closing database connection: {e}")


# Create FastAPI application
app = FastAPI(
    title="ISP Health Checker API",
    description="API for submitting and retrieving ISP health check diagnostic reports.",
    version="1.0.0",
    lifespan=lifespan,
    docs_url="/docs",
    redoc_url="/redoc",
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=os.getenv("CORS_ORIGINS", "*").split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configure rate limiting
rate_limit_config = RateLimitConfig(
    requests_per_minute=int(os.getenv("RATE_LIMIT_PER_MINUTE", "60")),
    requests_per_hour=int(os.getenv("RATE_LIMIT_PER_HOUR", "1000")),
    burst_size=int(os.getenv("RATE_LIMIT_BURST", "10")),
)
app.add_middleware(RateLimitMiddleware, config=rate_limit_config)

# Setup Prometheus metrics
instrumentator = Instrumentator(
    should_group_status_codes=True,
    should_ignore_untemplated=True,
    should_respect_env_var=True,
    should_instrument_requests_inprogress=True,
    excluded_handlers=["/health", "/metrics"],
    inprogress_name="isp_checker_inprogress",
    inprogress_labels=True,
)
instrumentator.instrument(app).expose(app, include_in_schema=False)

# Include routers
app.include_router(runs.router, prefix="/api/v1/runs", tags=["runs"])


@app.get("/", include_in_schema=False)
async def root():
    """Redirect to API documentation."""
    return RedirectResponse(url="/docs")


@app.get("/health")
async def health_check():
    """Health check endpoint for Kubernetes liveness/readiness probes."""
    return {
        "status": "healthy",
        "service": "isp-checker-backend",
        "version": "1.0.0",
    }


@app.get("/ready")
async def readiness_check():
    """
    Readiness check endpoint.
    Verifies database connectivity.
    """
    from .db import database

    try:
        # Test database connection
        await database.execute("SELECT 1")
        db_status = "connected"
    except Exception as e:
        db_status = f"disconnected: {str(e)}"

    worker_pool = get_worker_pool()

    return {
        "status": "ready" if db_status == "connected" else "not_ready",
        "database": db_status,
        "active_workers": worker_pool.get_active_count(),
    }


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    """Global exception handler for unhandled errors."""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    return JSONResponse(
        status_code=500,
        content={
            "detail": "An internal server error occurred.",
            "type": type(exc).__name__,
        },
    )


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=os.getenv("HOST", "0.0.0.0"),
        port=int(os.getenv("PORT", "8000")),
        reload=os.getenv("DEBUG", "false").lower() == "true",
    )
