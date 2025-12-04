from sqlalchemy import (
    create_engine,
    MetaData,
    Table,
    Column,
    Integer,
    String,
    Float,
    DateTime,
    JSON,
)
from databases import Database
import os

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://user:password@localhost/isp_health_checker")

engine = create_engine(DATABASE_URL)
metadata = MetaData()

runs = Table(
    "runs",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("run_id", String, unique=True, index=True),
    Column("timestamp", DateTime),
    Column("target", String, index=True),
    Column("score", Float),
    Column("summary", String),
    Column("report", JSON), # Storing the full JSON report
)

database = Database(DATABASE_URL)
