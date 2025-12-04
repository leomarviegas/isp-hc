"""
Database configuration and connection management for ISP Health Checker.
Uses SQLAlchemy for ORM and async database operations.
"""
from sqlalchemy import (
    create_engine,
    MetaData,
    Table,
    Column,
    Integer,
    String,
    Float,
    DateTime,
    Boolean,
    Text,
    ForeignKey,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker, declarative_base
from databases import Database
import os

# Database URL configuration
DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgresql://user:password@localhost/isp_health_checker"
)

# Convert to async URL for SQLAlchemy async engine
ASYNC_DATABASE_URL = DATABASE_URL.replace("postgresql://", "postgresql+asyncpg://")

# Sync engine for migrations
engine = create_engine(DATABASE_URL)

# Async engine for application
async_engine = create_async_engine(ASYNC_DATABASE_URL, echo=False)

# Session factory
AsyncSessionLocal = sessionmaker(
    bind=async_engine,
    class_=AsyncSession,
    expire_on_commit=False,
)

# Base for ORM models
Base = declarative_base()

# Metadata for table definitions
metadata = MetaData()

# ============================================================================
# Table Definitions
# ============================================================================

users = Table(
    "users",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("username", String(50), unique=True, nullable=False),
    Column("email", String(255), unique=True, nullable=False),
    Column("password_hash", String(255), nullable=False),
    Column("created_at", DateTime, server_default="now()"),
    Column("updated_at", DateTime, server_default="now()"),
    Column("is_active", Boolean, default=True),
)

api_keys = Table(
    "api_keys",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("key", String(255), unique=True, nullable=False, index=True),
    Column("user_id", Integer, ForeignKey("users.id", ondelete="CASCADE")),
    Column("name", String(100), nullable=False),
    Column("created_at", DateTime, server_default="now()"),
    Column("expires_at", DateTime, nullable=True),
    Column("is_active", Boolean, default=True),
)

runs = Table(
    "runs",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("run_id", String(255), unique=True, nullable=False, index=True),
    Column("user_id", Integer, ForeignKey("users.id", ondelete="CASCADE"), nullable=True),
    Column("timestamp", DateTime, nullable=False),
    Column("target", String(255), nullable=False, index=True),
    Column("mode", String(50), nullable=False),
    Column("score", Float, nullable=False),
    Column("summary", Text),
    Column("report", JSONB, nullable=False),
    Column("created_at", DateTime, server_default="now()"),
)

probes = Table(
    "probes",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("run_db_id", Integer, ForeignKey("runs.id", ondelete="CASCADE")),
    Column("name", String(50), nullable=False),
    Column("status", String(20), nullable=False),
    Column("latency_ms", Float, nullable=True),
    Column("details", JSONB),
    Column("error", Text, nullable=True),
)

# ============================================================================
# Database connection using 'databases' library for async operations
# ============================================================================

database = Database(DATABASE_URL)


async def get_db_session() -> AsyncSession:
    """Dependency for getting async database sessions."""
    async with AsyncSessionLocal() as session:
        yield session


async def connect_db():
    """Connect to the database."""
    await database.connect()


async def disconnect_db():
    """Disconnect from the database."""
    await database.disconnect()


def create_tables():
    """Create all tables in the database (for development/testing)."""
    metadata.create_all(engine)


def drop_tables():
    """Drop all tables in the database (for development/testing)."""
    metadata.drop_all(engine)
