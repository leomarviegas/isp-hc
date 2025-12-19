"""
API routes for health check runs.
Implements CRUD operations with PostgreSQL storage.
"""
from fastapi import APIRouter, HTTPException, Depends, BackgroundTasks, Query
from typing import List, Optional
import uuid
from datetime import datetime

import models
from db import database, runs, probes
from middleware.auth import get_current_user, AuthenticatedUser, APIKeyAuth
from workers.probe_runner import execute_health_check

router = APIRouter()


@router.post("/", status_code=202)
async def submit_or_start_run(
    run: models.Run,
    background_tasks: BackgroundTasks,
    user: AuthenticatedUser = Depends(APIKeyAuth(required=False)),
):
    """
    Submits a new diagnostic report for storage or initiates a new run.

    If the run contains probe data, it will be stored directly.
    If the run only contains a target, a background health check will be initiated.
    """
    # Generate a new run_id
    run_id = str(uuid.uuid4())

    # Prepare the report JSON (full run data)
    report_data = run.dict()
    report_data["run_id"] = run_id

    # Get user_id if authenticated
    user_id = user.user_id if user else None

    # Check if this is a request to start a new run or to store existing data
    if not run.probes:
        # No probe data - create a pending record and initiate background health check
        pending_report = {
            "run_id": run_id,
            "target": run.target,
            "mode": run.mode,
            "score": 0,
            "summary": "Health check in progress...",
            "probes": [],
            "diagnosis": [],
            "status": "pending",
        }

        # Create the pending run record immediately so UI can fetch it
        query = runs.insert().values(
            run_id=run_id,
            user_id=user_id,
            timestamp=datetime.utcnow(),
            target=run.target,
            mode=run.mode,
            score=0,
            summary="Health check in progress...",
            report=pending_report,
        )
        await database.execute(query)

        # Start background task to run actual health check
        background_tasks.add_task(
            execute_health_check,
            run_id=run_id,
            target=run.target,
            mode=run.mode,
            user_id=user_id,
        )
        return {"run_id": run_id, "status": "pending"}

    # Store the run in the database
    query = runs.insert().values(
        run_id=run_id,
        user_id=user_id,
        timestamp=run.timestamp or datetime.utcnow(),
        target=run.target,
        mode=run.mode,
        score=run.score,
        summary=run.summary,
        report=report_data,
    )

    try:
        db_run_id = await database.execute(query)

        # Store individual probes for queryability
        for probe in run.probes:
            probe_query = probes.insert().values(
                run_db_id=db_run_id,
                name=probe.name.value if hasattr(probe.name, 'value') else probe.name,
                status=probe.status.value if hasattr(probe.status, 'value') else probe.status,
                latency_ms=probe.details.get("latency_ms"),
                details=probe.details,
                error=probe.details.get("error"),
            )
            await database.execute(probe_query)

    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to store run: {str(e)}")

    return {"run_id": run_id, "status": "accepted"}


@router.get("/", response_model=List[models.Run])
async def list_recent_runs(
    limit: int = Query(20, ge=1, le=100),
    offset: int = Query(0, ge=0),
    target: Optional[str] = Query(None, description="Filter by target"),
    user: AuthenticatedUser = Depends(APIKeyAuth(required=False)),
):
    """
    Retrieves a paginated list of the most recent diagnostic runs.
    """
    # Build query with optional filters
    query = runs.select()

    if target:
        query = query.where(runs.c.target == target)

    query = query.order_by(runs.c.timestamp.desc()).limit(limit).offset(offset)

    rows = await database.fetch_all(query)

    result = []
    for row in rows:
        # Extract run data from the stored report JSON
        report = row["report"] or {}
        result.append(models.Run(
            run_id=row["run_id"],
            timestamp=row["timestamp"],
            target=row["target"],
            mode=row["mode"],
            score=row["score"],
            summary=row["summary"],
            probes=report.get("probes", []),
            diagnosis=report.get("diagnosis", []),
            raw=report.get("raw", {}),
        ))

    return result


@router.get("/{run_id}", response_model=models.Run)
async def get_run_details(
    run_id: str,
    user: AuthenticatedUser = Depends(APIKeyAuth(required=False)),
):
    """
    Retrieves the full details for a specific diagnostic run by its ID.
    """
    query = runs.select().where(runs.c.run_id == run_id)
    row = await database.fetch_one(query)

    if not row:
        raise HTTPException(status_code=404, detail="Run not found")

    report = row["report"] or {}

    return models.Run(
        run_id=row["run_id"],
        timestamp=row["timestamp"],
        target=row["target"],
        mode=row["mode"],
        score=row["score"],
        summary=row["summary"],
        probes=report.get("probes", []),
        diagnosis=report.get("diagnosis", []),
        raw=report.get("raw", {}),
    )


@router.get("/{run_id}/raw")
async def get_raw_probe_output(
    run_id: str,
    user: AuthenticatedUser = Depends(APIKeyAuth(required=False)),
):
    """
    Retrieves just the raw, unprocessed output from the probes for a specific run.
    """
    query = runs.select().where(runs.c.run_id == run_id)
    row = await database.fetch_one(query)

    if not row:
        raise HTTPException(status_code=404, detail="Run not found")

    report = row["report"] or {}
    return report.get("raw", {})


@router.delete("/{run_id}", status_code=204)
async def delete_run(
    run_id: str,
    user: AuthenticatedUser = Depends(get_current_user),
):
    """
    Deletes a diagnostic run by its ID.
    Requires authentication.
    """
    # Check if run exists
    query = runs.select().where(runs.c.run_id == run_id)
    row = await database.fetch_one(query)

    if not row:
        raise HTTPException(status_code=404, detail="Run not found")

    # Delete the run (probes will be deleted via CASCADE)
    delete_query = runs.delete().where(runs.c.run_id == run_id)
    await database.execute(delete_query)

    return None


@router.get("/{run_id}/probes")
async def get_run_probes(
    run_id: str,
    user: AuthenticatedUser = Depends(APIKeyAuth(required=False)),
):
    """
    Retrieves the individual probe results for a specific run.
    """
    # First, get the run to ensure it exists and get the db id
    run_query = runs.select().where(runs.c.run_id == run_id)
    run_row = await database.fetch_one(run_query)

    if not run_row:
        raise HTTPException(status_code=404, detail="Run not found")

    # Get all probes for this run
    probe_query = probes.select().where(probes.c.run_db_id == run_row["id"])
    probe_rows = await database.fetch_all(probe_query)

    return [
        {
            "name": row["name"],
            "status": row["status"],
            "latency_ms": row["latency_ms"],
            "details": row["details"],
            "error": row["error"],
        }
        for row in probe_rows
    ]
