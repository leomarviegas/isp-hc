from fastapi import FastAPI
from starlette.responses import RedirectResponse
import uvicorn

from .api import runs
from .db import database

app = FastAPI(
    title="ISP Health Checker API",
    description="API for submitting and retrieving ISP health check diagnostic reports.",
    version="1.0.0",
)

@app.on_event("startup")
async def startup():
    await database.connect()


@app.on_event("shutdown")
async def shutdown():
    await database.disconnect()

# Include routers
app.include_router(runs.router, prefix="/api/v1/runs", tags=["runs"])


@app.get("/")
async def root():
    return RedirectResponse(url="/docs")


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)