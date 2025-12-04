#!/usr/bin/env python3
"""
Database migration runner for ISP Health Checker.
Applies SQL migrations in order.
"""
import os
import sys
import glob
import psycopg2
from psycopg2.extensions import ISOLATION_LEVEL_AUTOCOMMIT

DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgresql://user:password@localhost/isp_health_checker"
)


def parse_database_url(url: str) -> dict:
    """Parse PostgreSQL database URL into connection parameters."""
    # postgresql://user:password@host:port/database
    url = url.replace("postgresql://", "")

    # Split user:password from host:port/database
    if "@" in url:
        auth, rest = url.split("@", 1)
        user, password = auth.split(":", 1) if ":" in auth else (auth, "")
    else:
        user, password = "", ""
        rest = url

    # Split host:port from database
    if "/" in rest:
        host_port, database = rest.rsplit("/", 1)
    else:
        host_port, database = rest, "postgres"

    # Split host from port
    if ":" in host_port:
        host, port = host_port.split(":", 1)
        port = int(port)
    else:
        host = host_port
        port = 5432

    return {
        "user": user,
        "password": password,
        "host": host,
        "port": port,
        "database": database,
    }


def get_connection(database_override: str = None):
    """Get database connection."""
    params = parse_database_url(DATABASE_URL)
    if database_override:
        params["database"] = database_override
    return psycopg2.connect(**params)


def ensure_database_exists():
    """Create database if it doesn't exist."""
    params = parse_database_url(DATABASE_URL)
    db_name = params["database"]

    try:
        # Connect to postgres database to create the target database
        conn = get_connection("postgres")
        conn.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)
        cur = conn.cursor()

        # Check if database exists
        cur.execute(
            "SELECT 1 FROM pg_catalog.pg_database WHERE datname = %s",
            (db_name,)
        )
        exists = cur.fetchone()

        if not exists:
            cur.execute(f'CREATE DATABASE "{db_name}"')
            print(f"Created database: {db_name}")
        else:
            print(f"Database already exists: {db_name}")

        cur.close()
        conn.close()
    except Exception as e:
        print(f"Error creating database: {e}")
        raise


def create_migrations_table():
    """Create migrations tracking table if it doesn't exist."""
    conn = get_connection()
    cur = conn.cursor()

    cur.execute("""
        CREATE TABLE IF NOT EXISTS _migrations (
            id SERIAL PRIMARY KEY,
            filename VARCHAR(255) NOT NULL UNIQUE,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.commit()
    cur.close()
    conn.close()


def get_applied_migrations() -> set:
    """Get set of already applied migration filenames."""
    conn = get_connection()
    cur = conn.cursor()

    cur.execute("SELECT filename FROM _migrations")
    applied = {row[0] for row in cur.fetchall()}

    cur.close()
    conn.close()
    return applied


def apply_migration(filepath: str):
    """Apply a single migration file."""
    filename = os.path.basename(filepath)

    with open(filepath, "r") as f:
        sql = f.read()

    conn = get_connection()
    cur = conn.cursor()

    try:
        # Execute the migration SQL
        cur.execute(sql)

        # Record the migration
        cur.execute(
            "INSERT INTO _migrations (filename) VALUES (%s)",
            (filename,)
        )

        conn.commit()
        print(f"Applied migration: {filename}")
    except Exception as e:
        conn.rollback()
        print(f"Failed to apply migration {filename}: {e}")
        raise
    finally:
        cur.close()
        conn.close()


def run_migrations():
    """Run all pending migrations."""
    migrations_dir = os.path.dirname(os.path.abspath(__file__))

    # Find all SQL migration files
    migration_files = sorted(glob.glob(os.path.join(migrations_dir, "*.sql")))

    if not migration_files:
        print("No migration files found.")
        return

    # Ensure database exists
    ensure_database_exists()

    # Create migrations tracking table
    create_migrations_table()

    # Get already applied migrations
    applied = get_applied_migrations()

    # Apply pending migrations
    pending = [f for f in migration_files if os.path.basename(f) not in applied]

    if not pending:
        print("All migrations are up to date.")
        return

    print(f"Found {len(pending)} pending migration(s).")

    for filepath in pending:
        apply_migration(filepath)

    print("All migrations applied successfully.")


def rollback_last():
    """Rollback the last applied migration (if rollback file exists)."""
    conn = get_connection()
    cur = conn.cursor()

    # Get the last applied migration
    cur.execute(
        "SELECT filename FROM _migrations ORDER BY applied_at DESC LIMIT 1"
    )
    row = cur.fetchone()

    if not row:
        print("No migrations to rollback.")
        cur.close()
        conn.close()
        return

    filename = row[0]
    rollback_file = filename.replace(".sql", "_rollback.sql")
    migrations_dir = os.path.dirname(os.path.abspath(__file__))
    rollback_path = os.path.join(migrations_dir, rollback_file)

    if not os.path.exists(rollback_path):
        print(f"No rollback file found for {filename}")
        cur.close()
        conn.close()
        return

    with open(rollback_path, "r") as f:
        sql = f.read()

    try:
        cur.execute(sql)
        cur.execute("DELETE FROM _migrations WHERE filename = %s", (filename,))
        conn.commit()
        print(f"Rolled back migration: {filename}")
    except Exception as e:
        conn.rollback()
        print(f"Failed to rollback {filename}: {e}")
        raise
    finally:
        cur.close()
        conn.close()


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "rollback":
        rollback_last()
    else:
        run_migrations()
