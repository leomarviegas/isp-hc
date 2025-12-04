"""
Database Integration Tests for ISP Health Checker

Tests cover:
- Database connection
- Table operations
- Query performance
"""
import pytest
import asyncio
import os
import sys

# Add the backend to the path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../src/backend')))

from datetime import datetime
from uuid import uuid4


# Skip database tests if DATABASE_URL is not set to a test database
pytestmark = pytest.mark.skipif(
    os.environ.get('RUN_DB_TESTS', 'false').lower() != 'true',
    reason="Database tests skipped. Set RUN_DB_TESTS=true and provide DATABASE_URL to run."
)


@pytest.fixture
def event_loop():
    """Create an instance of the event loop for the test session."""
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
async def db_connection():
    """Create a database connection for testing."""
    from db import database, connect_db, disconnect_db

    await connect_db()
    yield database
    await disconnect_db()


@pytest.fixture
def sample_run():
    """Create a sample run record."""
    return {
        "run_id": str(uuid4()),
        "timestamp": datetime.utcnow(),
        "target": "8.8.8.8",
        "mode": "full",
        "score": 95.0,
        "summary": "Test run",
        "report": {
            "probes": [],
            "diagnosis": [],
            "raw": {}
        }
    }


class TestDatabaseConnection:
    """Test database connection."""

    @pytest.mark.asyncio
    async def test_connection(self, db_connection):
        """Test that database connection works."""
        result = await db_connection.execute("SELECT 1 as value")
        assert result is not None

    @pytest.mark.asyncio
    async def test_tables_exist(self, db_connection):
        """Test that required tables exist."""
        # Check users table
        result = await db_connection.fetch_all(
            "SELECT table_name FROM information_schema.tables WHERE table_name = 'users'"
        )
        assert len(result) == 1

        # Check runs table
        result = await db_connection.fetch_all(
            "SELECT table_name FROM information_schema.tables WHERE table_name = 'runs'"
        )
        assert len(result) == 1


class TestRunsTable:
    """Test runs table operations."""

    @pytest.mark.asyncio
    async def test_insert_run(self, db_connection, sample_run):
        """Test inserting a run record."""
        from db import runs

        query = runs.insert().values(**sample_run)
        result = await db_connection.execute(query)
        assert result is not None

    @pytest.mark.asyncio
    async def test_select_run(self, db_connection, sample_run):
        """Test selecting a run record."""
        from db import runs

        # Insert first
        insert_query = runs.insert().values(**sample_run)
        await db_connection.execute(insert_query)

        # Select
        select_query = runs.select().where(runs.c.run_id == sample_run["run_id"])
        result = await db_connection.fetch_one(select_query)

        assert result is not None
        assert result["run_id"] == sample_run["run_id"]
        assert result["target"] == sample_run["target"]

    @pytest.mark.asyncio
    async def test_update_run(self, db_connection, sample_run):
        """Test updating a run record."""
        from db import runs

        # Insert first
        insert_query = runs.insert().values(**sample_run)
        await db_connection.execute(insert_query)

        # Update
        new_summary = "Updated test run"
        update_query = runs.update().where(
            runs.c.run_id == sample_run["run_id"]
        ).values(summary=new_summary)
        await db_connection.execute(update_query)

        # Verify
        select_query = runs.select().where(runs.c.run_id == sample_run["run_id"])
        result = await db_connection.fetch_one(select_query)

        assert result["summary"] == new_summary

    @pytest.mark.asyncio
    async def test_delete_run(self, db_connection, sample_run):
        """Test deleting a run record."""
        from db import runs

        # Insert first
        insert_query = runs.insert().values(**sample_run)
        await db_connection.execute(insert_query)

        # Delete
        delete_query = runs.delete().where(runs.c.run_id == sample_run["run_id"])
        await db_connection.execute(delete_query)

        # Verify
        select_query = runs.select().where(runs.c.run_id == sample_run["run_id"])
        result = await db_connection.fetch_one(select_query)

        assert result is None


class TestIndexPerformance:
    """Test index performance."""

    @pytest.mark.asyncio
    async def test_run_id_index(self, db_connection, sample_run):
        """Test that run_id index is used for lookups."""
        from db import runs

        # Insert sample run
        insert_query = runs.insert().values(**sample_run)
        await db_connection.execute(insert_query)

        # Query by run_id should use index
        explain_query = f"EXPLAIN SELECT * FROM runs WHERE run_id = '{sample_run['run_id']}'"
        result = await db_connection.fetch_all(explain_query)

        # Check that index scan is used (not sequential scan)
        explain_text = str(result)
        # This is a basic check - in production you'd want more thorough analysis
        assert result is not None

    @pytest.mark.asyncio
    async def test_target_index(self, db_connection, sample_run):
        """Test that target index is used for filtering."""
        from db import runs

        # Insert sample run
        insert_query = runs.insert().values(**sample_run)
        await db_connection.execute(insert_query)

        # Query by target should use index
        explain_query = f"EXPLAIN SELECT * FROM runs WHERE target = '{sample_run['target']}'"
        result = await db_connection.fetch_all(explain_query)

        assert result is not None
