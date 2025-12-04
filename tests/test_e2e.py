"""
End-to-End Tests for ISP Health Checker

Tests cover complete user workflows from submission to retrieval.
"""
import pytest
import time
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../src/backend')))

os.environ['DISABLE_AUTH'] = 'true'

from fastapi.testclient import TestClient
from main import app

client = TestClient(app)


class TestCompleteWorkflow:
    """Test complete user workflows."""

    def test_full_health_check_workflow(self):
        """Test complete workflow: submit -> list -> get details -> get raw."""
        # 1. Submit a new run
        run_data = {
            "target": "8.8.8.8",
            "mode": "full",
            "score": 90.0,
            "summary": "E2E test run",
            "probes": [
                {
                    "name": "ping",
                    "status": "OK",
                    "details": {"latency_ms": 20.5}
                },
                {
                    "name": "dns",
                    "status": "OK",
                    "details": {"resolution_time": "15ms"}
                },
                {
                    "name": "traceroute",
                    "status": "OK",
                    "details": {"hops": 8}
                }
            ],
            "diagnosis": [
                {
                    "component": "LocalNetwork",
                    "confidence": 0.95,
                    "explanation": "All systems operational",
                    "suggested_action": "None required"
                }
            ],
            "raw": {
                "ping_output": "PING 8.8.8.8...",
                "dns_output": "DNS lookup successful"
            }
        }

        # Submit
        submit_response = client.post("/api/v1/runs", json=run_data)
        assert submit_response.status_code == 202
        run_id = submit_response.json()["run_id"]

        # 2. List runs and verify our run appears
        list_response = client.get("/api/v1/runs")
        assert list_response.status_code == 200
        runs = list_response.json()
        run_ids = [r["run_id"] for r in runs]
        assert run_id in run_ids

        # 3. Get run details
        details_response = client.get(f"/api/v1/runs/{run_id}")
        assert details_response.status_code == 200
        details = details_response.json()
        assert details["target"] == run_data["target"]
        assert details["score"] == run_data["score"]
        assert len(details["probes"]) == 3

        # 4. Get raw output
        raw_response = client.get(f"/api/v1/runs/{run_id}/raw")
        assert raw_response.status_code == 200
        raw = raw_response.json()
        assert "ping_output" in raw

        # 5. Get probes
        probes_response = client.get(f"/api/v1/runs/{run_id}/probes")
        assert probes_response.status_code == 200

    def test_multiple_runs_different_targets(self):
        """Test handling multiple runs for different targets."""
        targets = ["8.8.8.8", "1.1.1.1", "9.9.9.9"]
        run_ids = []

        # Submit runs for different targets
        for target in targets:
            response = client.post("/api/v1/runs", json={
                "target": target,
                "mode": "ping",
                "score": 100.0,
                "summary": f"Test for {target}",
                "probes": [{"name": "ping", "status": "OK", "details": {}}],
                "diagnosis": [],
                "raw": {}
            })
            assert response.status_code == 202
            run_ids.append(response.json()["run_id"])

        # Verify all runs can be retrieved
        for run_id in run_ids:
            response = client.get(f"/api/v1/runs/{run_id}")
            assert response.status_code == 200

    def test_pagination_workflow(self):
        """Test pagination works correctly with multiple runs."""
        # Submit several runs
        for i in range(5):
            client.post("/api/v1/runs", json={
                "target": f"10.0.0.{i}",
                "mode": "ping",
                "score": 80.0 + i,
                "summary": f"Pagination test {i}",
                "probes": [],
                "diagnosis": [],
                "raw": {}
            })

        # Get first page
        page1 = client.get("/api/v1/runs?limit=2&offset=0")
        assert page1.status_code == 200
        assert len(page1.json()) <= 2

        # Get second page
        page2 = client.get("/api/v1/runs?limit=2&offset=2")
        assert page2.status_code == 200

    def test_error_recovery_workflow(self):
        """Test handling of error scenarios."""
        # Try to get non-existent run
        response = client.get("/api/v1/runs/non-existent-id")
        assert response.status_code == 404

        # Submit valid run
        submit_response = client.post("/api/v1/runs", json={
            "target": "8.8.8.8",
            "mode": "full",
            "score": 0.0,
            "summary": "Recovery test",
            "probes": [],
            "diagnosis": [],
            "raw": {}
        })
        assert submit_response.status_code == 202

        # Verify we can still list runs after error
        list_response = client.get("/api/v1/runs")
        assert list_response.status_code == 200


class TestHealthCheckModes:
    """Test different health check modes."""

    def test_ping_only_mode(self):
        """Test ping-only mode submission."""
        response = client.post("/api/v1/runs", json={
            "target": "8.8.8.8",
            "mode": "ping",
            "score": 100.0,
            "summary": "Ping successful",
            "probes": [
                {"name": "ping", "status": "OK", "details": {"latency_ms": 15.0}}
            ],
            "diagnosis": [],
            "raw": {}
        })
        assert response.status_code == 202

    def test_dns_only_mode(self):
        """Test DNS-only mode submission."""
        response = client.post("/api/v1/runs", json={
            "target": "google.com",
            "mode": "dns",
            "score": 100.0,
            "summary": "DNS resolution successful",
            "probes": [
                {"name": "dns", "status": "OK", "details": {"resolved_ip": "142.250.80.46"}}
            ],
            "diagnosis": [],
            "raw": {}
        })
        assert response.status_code == 202

    def test_traceroute_only_mode(self):
        """Test traceroute-only mode submission."""
        response = client.post("/api/v1/runs", json={
            "target": "8.8.8.8",
            "mode": "traceroute",
            "score": 100.0,
            "summary": "Traceroute complete",
            "probes": [
                {"name": "traceroute", "status": "OK", "details": {"hops": 12}}
            ],
            "diagnosis": [],
            "raw": {}
        })
        assert response.status_code == 202


class TestDiagnosisScenarios:
    """Test various diagnosis scenarios."""

    def test_healthy_diagnosis(self):
        """Test healthy network diagnosis."""
        response = client.post("/api/v1/runs", json={
            "target": "8.8.8.8",
            "mode": "full",
            "score": 95.0,
            "summary": "Connection healthy",
            "probes": [
                {"name": "ping", "status": "OK", "details": {"packet_loss": 0}}
            ],
            "diagnosis": [
                {
                    "component": "LocalNetwork",
                    "confidence": 0.95,
                    "explanation": "No issues detected",
                    "suggested_action": "None required"
                }
            ],
            "raw": {}
        })
        assert response.status_code == 202

        run_id = response.json()["run_id"]
        details = client.get(f"/api/v1/runs/{run_id}").json()
        assert len(details["diagnosis"]) == 1
        assert details["diagnosis"][0]["component"] == "LocalNetwork"

    def test_network_issue_diagnosis(self):
        """Test network issue diagnosis."""
        response = client.post("/api/v1/runs", json={
            "target": "10.0.0.1",
            "mode": "full",
            "score": 35.0,
            "summary": "Network issues detected",
            "probes": [
                {"name": "ping", "status": "CRIT", "details": {"packet_loss": 75}}
            ],
            "diagnosis": [
                {
                    "component": "Transit",
                    "confidence": 0.85,
                    "explanation": "High packet loss on transit path",
                    "suggested_action": "Contact ISP support"
                },
                {
                    "component": "Upstream",
                    "confidence": 0.60,
                    "explanation": "Possible upstream congestion",
                    "suggested_action": "Monitor during off-peak hours"
                }
            ],
            "raw": {}
        })
        assert response.status_code == 202

        run_id = response.json()["run_id"]
        details = client.get(f"/api/v1/runs/{run_id}").json()
        assert len(details["diagnosis"]) == 2
