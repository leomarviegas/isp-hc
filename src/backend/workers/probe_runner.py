"""
Background worker for executing health check probes.
Runs the Go CLI and stores results in the database.
"""
import asyncio
import json
import subprocess
import os
import logging
from datetime import datetime
from typing import Optional, Dict, Any

from ..db import database, runs

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Path to the Go CLI binary
CLI_BINARY = os.getenv("ISP_CHECKER_CLI", "/usr/local/bin/isp-checker")

# Timeout for CLI execution (seconds)
CLI_TIMEOUT = int(os.getenv("CLI_TIMEOUT", "60"))


async def execute_health_check(
    run_id: str,
    target: str,
    mode: str = "full",
    user_id: Optional[int] = None,
) -> Dict[str, Any]:
    """
    Execute a health check using the Go CLI and store results.

    Args:
        run_id: Unique identifier for this run
        target: Target host/IP to check
        mode: Check mode (full, ping, dns, traceroute)
        user_id: Optional user ID for ownership

    Returns:
        Dict containing the run result
    """
    logger.info(f"Starting health check for run_id={run_id}, target={target}, mode={mode}")

    start_time = datetime.utcnow()
    result = None
    error_message = None

    try:
        # Execute the Go CLI
        result = await run_cli_probe(target, mode)

    except asyncio.TimeoutError:
        logger.error(f"Health check timed out for run_id={run_id}")
        error_message = "Health check timed out"
        result = create_error_result(target, mode, error_message)

    except subprocess.CalledProcessError as e:
        logger.error(f"CLI execution failed for run_id={run_id}: {e}")
        error_message = f"CLI execution failed: {e.stderr}"
        result = create_error_result(target, mode, error_message)

    except FileNotFoundError:
        logger.error(f"CLI binary not found: {CLI_BINARY}")
        error_message = "CLI binary not found"
        result = create_error_result(target, mode, error_message)

    except Exception as e:
        logger.error(f"Unexpected error for run_id={run_id}: {e}")
        error_message = f"Unexpected error: {str(e)}"
        result = create_error_result(target, mode, error_message)

    # Update the result with run metadata
    result["run_id"] = run_id
    result["timestamp"] = start_time.isoformat()

    # Store the result in the database
    await store_run_result(run_id, result, user_id)

    logger.info(f"Completed health check for run_id={run_id}, score={result.get('score', 0)}")

    return result


async def run_cli_probe(target: str, mode: str) -> Dict[str, Any]:
    """
    Run the Go CLI probe and return the parsed result.
    """
    # Check if CLI binary exists
    if not os.path.exists(CLI_BINARY):
        # Fall back to running via go run or using simulation
        return await run_simulation_probe(target, mode)

    # Build the CLI command
    cmd = [
        CLI_BINARY,
        "run",
        "--target", target,
        "--type", mode,
    ]

    # Run the CLI asynchronously
    process = await asyncio.create_subprocess_exec(
        *cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )

    try:
        stdout, stderr = await asyncio.wait_for(
            process.communicate(),
            timeout=CLI_TIMEOUT
        )
    except asyncio.TimeoutError:
        process.kill()
        raise

    if process.returncode != 0:
        raise subprocess.CalledProcessError(
            process.returncode,
            cmd,
            output=stdout,
            stderr=stderr.decode() if stderr else ""
        )

    # Parse the JSON output
    output = stdout.decode().strip()

    # Find the JSON in the output (CLI may have other output before it)
    json_start = output.find("{")
    if json_start >= 0:
        output = output[json_start:]

    return json.loads(output)


async def run_simulation_probe(target: str, mode: str) -> Dict[str, Any]:
    """
    Run a simulated probe when CLI is not available.
    Useful for development and testing.
    """
    logger.info(f"Running simulation probe for target={target}, mode={mode}")

    # Simulate network latency
    await asyncio.sleep(0.5)

    # Generate simulated results
    probes_result = []

    if mode in ["full", "ping"]:
        probes_result.append({
            "name": "ping",
            "status": "ok",
            "latency_ms": 25.5,
            "details": {
                "packets_sent": 3,
                "packets_received": 3,
                "packet_loss": 0,
                "min_rtt": 24.1,
                "avg_rtt": 25.5,
                "max_rtt": 27.2,
            }
        })

    if mode in ["full", "dns"]:
        probes_result.append({
            "name": "dns",
            "status": "ok",
            "latency_ms": 12.3,
            "details": {
                "resolution_time": "12.3ms",
                "resolved_ip": target if target.replace(".", "").isdigit() else "93.184.216.34",
            }
        })

    if mode in ["full", "traceroute"]:
        probes_result.append({
            "name": "traceroute",
            "status": "ok",
            "latency_ms": 150.0,
            "details": {
                "hops": 8,
                "complete": True,
            }
        })

    # Calculate score based on probe results
    ok_count = sum(1 for p in probes_result if p["status"] == "ok")
    total_count = len(probes_result)
    score = (ok_count / total_count * 100) if total_count > 0 else 0

    return {
        "target": target,
        "mode": mode,
        "score": score,
        "summary": "Connection appears healthy." if score >= 80 else "Some issues detected.",
        "probes": probes_result,
        "diagnosis": [
            {
                "component": "LocalNetwork",
                "confidence": 0.95,
                "explanation": "All probes completed successfully.",
                "suggested_action": "No action required."
            }
        ] if score >= 80 else [
            {
                "component": "Transit",
                "confidence": 0.7,
                "explanation": "Some network issues detected.",
                "suggested_action": "Check network connectivity."
            }
        ],
        "raw": {
            "simulation": True,
            "note": "This is simulated data. CLI binary not available."
        }
    }


def create_error_result(target: str, mode: str, error: str) -> Dict[str, Any]:
    """
    Create an error result when probe execution fails.
    """
    return {
        "target": target,
        "mode": mode,
        "score": 0,
        "summary": f"Health check failed: {error}",
        "probes": [],
        "diagnosis": [
            {
                "component": "LocalNetwork",
                "confidence": 0.5,
                "explanation": error,
                "suggested_action": "Check the health checker configuration and try again."
            }
        ],
        "raw": {
            "error": error
        }
    }


async def store_run_result(
    run_id: str,
    result: Dict[str, Any],
    user_id: Optional[int] = None
):
    """
    Store the run result in the database.
    """
    timestamp = datetime.fromisoformat(result.get("timestamp", datetime.utcnow().isoformat()))

    query = runs.insert().values(
        run_id=run_id,
        user_id=user_id,
        timestamp=timestamp,
        target=result.get("target", "unknown"),
        mode=result.get("mode", "unknown"),
        score=result.get("score", 0),
        summary=result.get("summary", ""),
        report=result,
    )

    try:
        await database.execute(query)
        logger.info(f"Stored run result for run_id={run_id}")
    except Exception as e:
        logger.error(f"Failed to store run result for run_id={run_id}: {e}")
        raise


class WorkerPool:
    """
    Worker pool for managing concurrent health check executions.
    """

    def __init__(self, max_workers: int = 10):
        self.max_workers = max_workers
        self._semaphore = asyncio.Semaphore(max_workers)
        self._active_tasks: Dict[str, asyncio.Task] = {}

    async def submit(
        self,
        run_id: str,
        target: str,
        mode: str = "full",
        user_id: Optional[int] = None,
    ) -> str:
        """
        Submit a health check job to the worker pool.

        Returns the run_id immediately; execution happens in background.
        """
        async def run_with_semaphore():
            async with self._semaphore:
                try:
                    return await execute_health_check(run_id, target, mode, user_id)
                finally:
                    self._active_tasks.pop(run_id, None)

        task = asyncio.create_task(run_with_semaphore())
        self._active_tasks[run_id] = task

        return run_id

    def get_active_count(self) -> int:
        """Get the number of currently active tasks."""
        return len(self._active_tasks)

    def is_run_active(self, run_id: str) -> bool:
        """Check if a specific run is currently active."""
        return run_id in self._active_tasks

    async def cancel(self, run_id: str) -> bool:
        """
        Cancel a running health check.

        Returns True if the task was found and cancelled.
        """
        task = self._active_tasks.get(run_id)
        if task and not task.done():
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass
            return True
        return False

    async def shutdown(self, timeout: float = 30.0):
        """
        Gracefully shut down the worker pool.

        Waits for all active tasks to complete or timeout.
        """
        if not self._active_tasks:
            return

        logger.info(f"Shutting down worker pool with {len(self._active_tasks)} active tasks")

        # Wait for tasks to complete with timeout
        tasks = list(self._active_tasks.values())
        done, pending = await asyncio.wait(tasks, timeout=timeout)

        # Cancel any remaining tasks
        for task in pending:
            task.cancel()

        logger.info("Worker pool shutdown complete")


# Global worker pool instance
_worker_pool: Optional[WorkerPool] = None


def get_worker_pool() -> WorkerPool:
    """Get or create the global worker pool."""
    global _worker_pool
    if _worker_pool is None:
        max_workers = int(os.getenv("MAX_WORKERS", "10"))
        _worker_pool = WorkerPool(max_workers=max_workers)
    return _worker_pool
