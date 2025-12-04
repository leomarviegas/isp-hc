from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
import uvicorn, os, uuid, json
from pydantic import BaseModel
from prometheus_client import Counter, generate_latest, CONTENT_TYPE_LATEST

RUNS_DIR = os.path.join(os.getcwd(), "runs")
os.makedirs(RUNS_DIR, exist_ok=True)

app = FastAPI(title="ISP Health Checker - Backend Prototype")

runs_counter = Counter("isp_checker_runs_total", "Total runs received")

class RunPayload(BaseModel):
    run_id: str
    timestamp: str
    target: str
    mode: str
    score: float
    summary: str
    probes: list
    diagnosis: list
    raw: dict = {}

@app.post("/runs", status_code=201)
async def post_run(payload: RunPayload):
    runs_counter.inc()
    rid = payload.run_id or str(uuid.uuid4())
    path = os.path.join(RUNS_DIR, f"{rid}.json")
    with open(path, "w") as f:
        json.dump(payload.dict(), f, indent=2)
    return {"id": rid, "stored": path}

@app.get("/runs/{run_id}")
async def get_run(run_id: str):
    path = os.path.join(RUNS_DIR, f"{run_id}.json")
    if not os.path.exists(path):
        raise HTTPException(status_code=404, detail="Run not found")
    with open(path) as f:
        return JSONResponse(content=json.load(f))

@app.get("/runs/{run_id}/raw")
async def get_run_raw(run_id: str):
    return await get_run(run_id)

@app.get("/metrics")
async def metrics():
    data = generate_latest()
    return JSONResponse(content=data.decode("utf-8"), media_type=CONTENT_TYPE_LATEST)

if __name__ == '__main__':
    uvicorn.run(app, host="0.0.0.0", port=8000)
