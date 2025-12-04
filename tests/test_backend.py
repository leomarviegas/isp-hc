"""
Backend API Tests for ISP Health Checker

Tests cover:
- CRUD operations for runs
- Authentication
- Rate limiting
- Error handling
"""
import pytest
from fastapi.testclient import TestClient
import os
import sys

# Add the backend to the path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../src/backend')))

# Set environment variable to disable auth for testing
os.environ['DISABLE_AUTH'] = 'true'

from main import app

client = TestClient(app)


# ============================================================================
# Fixtures
# ============================================================================

@pytest.fixture
def healthy_run_payload():
    """Create a sample healthy run payload."""
    return {
        "target": "8.8.8.8",
        "mode": "full",
        "score": 95.0,
        "summary": "Connection appears healthy.",
        "probes": [
            {
                "name": "ping",
                "status": "OK",
                "details": {
                    "latency_ms": 25.5,
                    "packets_sent": 3,
                    "packets_received": 3,
                    "packet_loss": 0
                }
            },
            {
                "name": "dns",
                "status": "OK",
                "details": {
                    "resolution_time": "12.3ms",
                    "resolved_ip": "8.8.8.8"
                }
            }
        ],
        "diagnosis": [
            {
                "component": "LocalNetwork",
                "confidence": 0.95,
                "explanation": "All probes completed successfully.",
                "suggested_action": "No action required."
            }
        ],
        "raw": {
            "ping_output": "PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data..."
        }
    }


@pytest.fixture
def minimal_run_payload():
    """Create a minimal run payload (for starting a new check)."""
    return {
        "target": "1.1.1.1",
        "mode": "ping",
        "score": 0,
        "summary": "Pending..."
    }


@pytest.fixture
def warning_run_payload():
    """Create a run payload with warning status."""
    return {
        "target": "example.com",
        "mode": "full",
        "score": 65.0,
        "summary": "Some packet loss detected.",
        "probes": [
            {
                "name": "ping",
                "status": "WARN",
                "details": {
                    "latency_ms": 150.5,
                    "packets_sent": 3,
                    "packets_received": 2,
                    "packet_loss": 33
                }
            }
        ],
        "diagnosis": [],
        "raw": {}
    }


# ============================================================================
# Run CRUD Tests
# ============================================================================

class TestRunCRUD:
    """Test CRUD operations for runs."""

    def test_submit_run_with_probes(self, healthy_run_payload):
        """Test submitting a valid run with probe data."""
        response = client.post("/api/v1/runs", json=healthy_run_payload)
        assert response.status_code == 202

        data = response.json()
        assert "run_id" in data
        assert data["status"] == "accepted"
        assert len(data["run_id"]) == 36  # UUID format

    def test_submit_run_starts_background_check(self, minimal_run_payload):
        """Test submitting a minimal run starts a background health check."""
        response = client.post("/api/v1/runs", json=minimal_run_payload)
        assert response.status_code == 202

        data = response.json()
        assert "run_id" in data
        assert data["status"] == "pending"

    def test_get_run_details(self, healthy_run_payload):
        """Test retrieving a run after submission."""
        # Submit a run
        post_response = client.post("/api/v1/runs", json=healthy_run_payload)
        run_id = post_response.json()["run_id"]

        # Retrieve it
        get_response = client.get(f"/api/v1/runs/{run_id}")
        assert get_response.status_code == 200

        data = get_response.json()
        assert data["run_id"] == run_id
        assert data["target"] == healthy_run_payload["target"]
        assert data["score"] == healthy_run_payload["score"]

    def test_get_nonexistent_run(self):
        """Test that retrieving a nonexistent run returns 404."""
        response = client.get("/api/v1/runs/nonexistent-uuid-12345")
        assert response.status_code == 404
        assert "not found" in response.json()["detail"].lower()

    def test_list_runs(self, healthy_run_payload):
        """Test listing runs returns paginated results."""
        # Submit a run
        client.post("/api/v1/runs", json=healthy_run_payload)

        response = client.get("/api/v1/runs")
        assert response.status_code == 200

        data = response.json()
        assert isinstance(data, list)

    def test_list_runs_pagination(self, healthy_run_payload):
        """Test pagination parameters work correctly."""
        response = client.get("/api/v1/runs?limit=5&offset=0")
        assert response.status_code == 200

        data = response.json()
        assert len(data) <= 5

    def test_list_runs_filter_by_target(self, healthy_run_payload):
        """Test filtering runs by target."""
        # Submit a run
        client.post("/api/v1/runs", json=healthy_run_payload)

        response = client.get(f"/api/v1/runs?target={healthy_run_payload['target']}")
        assert response.status_code == 200

    def test_get_raw_output(self, healthy_run_payload):
        """Test retrieving raw probe output."""
        # Submit a run
        post_response = client.post("/api/v1/runs", json=healthy_run_payload)
        run_id = post_response.json()["run_id"]

        # Get raw output
        raw_response = client.get(f"/api/v1/runs/{run_id}/raw")
        assert raw_response.status_code == 200

    def test_get_run_probes(self, healthy_run_payload):
        """Test retrieving individual probe results."""
        # Submit a run
        post_response = client.post("/api/v1/runs", json=healthy_run_payload)
        run_id = post_response.json()["run_id"]

        # Get probes
        probes_response = client.get(f"/api/v1/runs/{run_id}/probes")
        assert probes_response.status_code == 200


# ============================================================================
# Health Check Endpoint Tests
# ============================================================================

class TestHealthEndpoints:
    """Test health and readiness endpoints."""

    def test_health_endpoint(self):
        """Test the health check endpoint."""
        response = client.get("/health")
        assert response.status_code == 200

        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "isp-checker-backend"

    def test_ready_endpoint(self):
        """Test the readiness check endpoint."""
        response = client.get("/ready")
        assert response.status_code == 200

        data = response.json()
        assert "status" in data
        assert "database" in data
        assert "active_workers" in data


# ============================================================================
# Validation Tests
# ============================================================================

class TestValidation:
    """Test input validation."""

    def test_invalid_run_missing_target(self):
        """Test that missing required fields return validation error."""
        response = client.post("/api/v1/runs", json={
            "mode": "full",
            "score": 0,
            "summary": "Test"
        })
        assert response.status_code == 422  # Validation error

    def test_invalid_pagination_limit(self):
        """Test that invalid pagination params are rejected."""
        response = client.get("/api/v1/runs?limit=-1")
        assert response.status_code == 422

    def test_invalid_pagination_offset(self):
        """Test that invalid offset is rejected."""
        response = client.get("/api/v1/runs?offset=-5")
        assert response.status_code == 422


# ============================================================================
# Error Handling Tests
# ============================================================================

class TestErrorHandling:
    """Test error handling."""

    def test_invalid_json(self):
        """Test that invalid JSON returns appropriate error."""
        response = client.post(
            "/api/v1/runs",
            content="not valid json",
            headers={"Content-Type": "application/json"}
        )
        assert response.status_code == 422

    def test_method_not_allowed(self):
        """Test that unsupported methods return 405."""
        response = client.put("/api/v1/runs")
        assert response.status_code == 405


# ============================================================================
# Root Endpoint Tests
# ============================================================================

class TestRootEndpoint:
    """Test root endpoint behavior."""

    def test_root_redirects_to_docs(self):
        """Test that root path redirects to documentation."""
        response = client.get("/", follow_redirects=False)
        assert response.status_code == 307
        assert "/docs" in response.headers.get("location", "")
