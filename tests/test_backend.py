import pytest
from fastapi.testclient import TestClient
import json
import os

# This is a bit of a hack to make the main app importable
# In a real project, you'd have a proper package structure
import sys
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../src/backend')))
from main import app


client = TestClient(app)

@pytest.fixture
def healthy_run_payload():
    # In a real test suite, you'd load this from the actual simulations file
    return {
      "run_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
      "timestamp": "2025-12-03T10:00:00Z",
      "target": "8.8.8.8",
      "mode": "simulation",
      "score": 8.0,
      "summary": "Connection appears healthy.",
      "probes": [],
      "diagnosis": [],
      "raw": {}
    }

def test_submit_run(healthy_run_payload):
    """
    Test submitting a valid run to the /runs endpoint.
    """
    response = client.post("/api/v1/runs", json=healthy_run_payload)
    assert response.status_code == 202
    data = response.json()
    assert "run_id" in data
    assert data["run_id"] != healthy_run_payload["run_id"] # The backend should assign a new ID

def test_get_run(healthy_run_payload):
    """
    Test retrieving a run after it has been submitted.
    """
    # First, submit a run
    post_response = client.post("/api/v1/runs", json=healthy_run_payload)
    run_id = post_response.json()["run_id"]

    # Now, retrieve it
    get_response = client.get(f"/api/v1/runs/{run_id}")
    assert get_response.status_code == 200
    data = get_response.json()
    assert data["run_id"] == run_id
    assert data["target"] == healthy_run_payload["target"]

def test_get_nonexistent_run():
    """
    Test that retrieving a nonexistent run returns a 404 error.
    """
    response = client.get("/api/v1/runs/nonexistent-id")
    assert response.status_code == 404

def test_list_runs(healthy_run_payload):
    """
    Test that listing runs returns a list of runs.
    """
    # Submit a run to ensure the list is not empty
    client.post("/api/v1/runs", json=healthy_run_payload)
    
    response = client.get("/api/v1/runs")
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, list)
    assert len(data) > 0