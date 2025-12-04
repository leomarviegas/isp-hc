from fastapi import APIRouter, HTTPException
from typing import List
import uuid

from .. import models

router = APIRouter()

# In-memory storage for demonstration purposes.
# A real implementation would use the database.
db = {}


@router.post("/", status_code=202)
async def submit_or_start_run(run: models.Run):
    """
    Submits a new diagnostic report for storage or initiates a new run.
    """
    run.run_id = str(uuid.uuid4())
    db[run.run_id] = run
    return {"run_id": run.run_id}


@router.get("/", response_model=List[models.Run])
async def list_recent_runs(limit: int = 20, offset: int = 0):
    """
    Retrieves a paginated list of the most recent diagnostic runs.
    """
    return list(db.values())[offset : offset + limit]


@router.get("/{run_id}", response_model=models.Run)
async def get_run_details(run_id: str):
    """
    Retrieves the full details for a specific diagnostic run by its ID.
    """
    if run_id not in db:
        raise HTTPException(status_code=404, detail="Run not found")
    return db[run_id]

@router.get("/{run_id}/raw")
async def get_raw_probe_output(run_id: str):
    """
    Retrieves just the raw, unprocessed output from the probes for a specific run.
    """
    if run_id not in db:
        raise HTTPException(status_code=404, detail="Run not found")
    return db[run_id].raw
