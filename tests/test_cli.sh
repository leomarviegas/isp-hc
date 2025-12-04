#!/bin/bash

# This script is a placeholder for CLI integration tests.
# It demonstrates how the compiled Go binary would be tested.

set -e

CLI_BINARY="./build/isp-checker"
SIMULATION_FILE="./simulations/healthy.json"

echo "--- Building CLI (if necessary) ---"
if [ ! -f "$CLI_BINARY" ]; then
    echo "CLI binary not found. Run 'go build -o $CLI_BINARY ./src/cli' first."
    exit 1
fi

echo "--- Test 1: Running in Simulation Mode ---"
OUTPUT_JSON=$(mktemp)
$CLI_BINARY run --mode simulation --target "$SIMULATION_FILE" --output "$OUTPUT_JSON"

# Very basic check: Does the output file contain the run_id from the simulation?
if ! grep -q '"run_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef"' "$OUTPUT_JSON"; then
    echo "Test 1 FAILED: The output JSON does not match the simulation input."
    exit 1
fi
echo "Test 1 PASSED."
rm "$OUTPUT_JSON"

echo "--- Test 2: Checking --serve mode ---"
# Start the server in the background
$CLI_BINARY serve &
SERVER_PID=$!

# Give it a moment to start
sleep 2

# Check if the /metrics endpoint is responding
if ! curl -s http://localhost:8080/metrics | grep -q "isp_checker_runs_total"; then
    echo "Test 2 FAILED: Metrics endpoint is not serving the expected metric."
    kill $SERVER_PID
    exit 1
fi

echo "Test 2 PASSED."
kill $SERVER_PID

echo "--- All CLI tests passed ---"
exit 0
