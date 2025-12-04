# Feature Backlog & Test Scenarios

## 1. Prioritized Feature Backlog

### MVP (Minimum Viable Product)

*   **Feat-1: CLI Probe Runner:**
    *   **Description:** The core CLI tool can run a basic set of probes (ping, traceroute, dns) against a public target.
    *   **AC:**
        *   `isp-checker run --target <ip>` executes without errors.
        *   Output is valid JSON conforming to the standard schema.
        *   `--mode=simulation` reads a local JSON and outputs it.
*   **Feat-2: Basic Scoring Engine:**
    *   **Description:** The tool calculates a health score based on packet loss and high latency.
    *   **AC:**
        *   A run with 0% loss and low latency scores below 20.
        *   A run with >10% loss scores above 70.
*   **Feat-3: Simulation JSONs:**
    *   **Description:** Three representative simulation files (`healthy`, `intermittent_loss`, `upstream_outage`) are created.
    *   **AC:**
        *   Files are present in `simulations/`.
        *   Each file successfully validates against the JSON schema.

### v1 (First Public Release)

*   **Feat-4: Backend API:**
    *   **Description:** A FastAPI backend with endpoints to submit and retrieve run results.
    *   **AC:**
        *   `POST /runs` accepts a valid JSON report and returns a `202` with a run ID.
        *   `GET /runs/{id}` retrieves the stored report.
*   **Feat-5: Additional Probes:**
    *   **Description:** Add `http` and `mtr` probes to the CLI runner.
    *   **AC:**
        *   `isp-checker run` includes `http` and `mtr` sections in the output.
*   **Feat-6: Prometheus Exporter:**
    *   **Description:** The backend exposes a `/metrics` endpoint in Prometheus format.
    *   **AC:**
        *   Metrics `isp_checker_run_total` and `isp_checker_score` are present.
*   **Feat-7: Dockerization:**
    *   **Description:** `Dockerfile` and `docker-compose.yml` for simplified deployment.
    *   **AC:**
        *   `docker-compose up` starts the backend API successfully.

### v2 (Future Enhancements)

*   **Feat-8: Web UI Dashboard:**
    *   **Description:** A React-based UI to view and search for runs.
    *   **AC:**
        *   Main dashboard displays a table of the 10 most recent runs.
        *   Clicking a run navigates to the detailed drilldown view.
*   **Feat-9: Advanced Probes:**
    *   **Description:** Add `speedtest` and `bgp` probes.
    *   **AC:**
        *   The tool can measure bandwidth and retrieve basic BGP ASN information.
*   **Feat-10: Authentication:**
    *   **Description:** Add API key authentication to the backend.
    *   **AC:**
        *   Requests to `/runs` without a valid `X-API-Key` header are rejected.
*   **Feat-11: Kubernetes Deployment:**
    *   **Description:** A Helm chart for deploying the application to Kubernetes.
    *   **AC:**
        *   `helm install isp-checker ./charts/isp-checker` deploys the backend and a web UI.

---

## 2. End-to-End Test Scenarios

### Happy Path Scenarios

**Scenario 1: Healthy Connection**
*   **Preconditions:** Network is stable. Target `8.8.8.8` is reachable with <20ms latency and 0% packet loss. DNS is responsive.
*   **Command:** `isp-checker run --target 8.8.8.8`
*   **Expected JSON:** See `simulations/healthy.json`. Score should be < 20. `diagnosis` array is empty or confirms health.

**Scenario 2: Successful API Submission**
*   **Preconditions:** Backend is running. A valid `healthy.json` file exists.
*   **Command:** `curl -X POST http://localhost:8000/runs -d @simulations/healthy.json`
*   **Expected Result:** `202 Accepted` response. A subsequent `GET /runs/{run_id}` returns the full JSON.

**Scenario 3: CLI Simulation Mode**
*   **Preconditions:** `simulations/healthy.json` exists.
*   **Command:** `isp-checker run --mode simulation --target simulations/healthy.json`
*   **Expected JSON:** The output JSON is identical to `simulations/healthy.json`.

### Failure Mode Scenarios

**Scenario 4: Intermittent Packet Loss**
*   **Preconditions:** Simulated 5% packet loss to a mid-path hop (e.g., hop 6).
*   **Command:** `isp-checker run --target 1.1.1.1`
*   **Expected JSON:** See `simulations/intermittent_loss.json`. Score is elevated (e.g., 40-60). `diagnosis` points to "Peering/Transit" with an explanation mentioning packet loss at a specific hop.

**Scenario 5: Full Upstream Outage**
*   **Preconditions:** Default route is down. Target `8.8.8.8` is unreachable. DNS resolution fails.
*   **Command:** `isp-checker run --target 8.8.8.8`
*   **Expected JSON:** See `simulations/upstream_outage.json`. Score is very high (e.g., 90-100). `ping` and `traceroute` probes are `CRIT` with 100% loss. `dns` probe is `CRIT`. `diagnosis` identifies an "Upstream Outage".

**Scenario 6: Local DNS Failure**
*   **Preconditions:** Local DNS resolver (e.g., `192.168.1.1`) is unresponsive, but direct IP connectivity works.
*   **Command:** `isp-checker run --target google.com`
*   **Expected JSON:** Score is high (~75). `dns` probe is `CRIT`. `ping` and other IP-based probes might be `NA` as the name can't be resolved. `diagnosis` clearly states "DNS Issue".

**Scenario 7: High Latency (Bufferbloat)**
*   **Preconditions:** A speed test is running in the background, causing latency on the link to spike to >300ms.
*   **Command:** `isp-checker run --target 1.1.1.1`
*   **Expected JSON:** Score is `WARN` (e.g., 30-50). `ping` probe details show high `avg` and `max` latency. `diagnosis` suggests "LocalNetwork" congestion.

**Scenario 8: Invalid Target Hostname**
*   **Preconditions:** Network is healthy.
*   **Command:** `isp-checker run --target non-existent-domain-12345.com`
*   **Expected JSON:** `dns` probe is `CRIT` with a "NXDOMAIN" error. Score is high. `diagnosis` indicates a DNS issue or invalid target.

**Scenario 9: Blocked ICMP (Ping)**
*   **Preconditions:** A firewall is blocking ICMP echo requests to the target. HTTP access is fine.
*   **Command:** `isp-checker run --target firewall-protected.com`
*   **Expected JSON:** `ping` probe is `CRIT` (100% loss). `http` probe is `OK`. Score is moderately high. `diagnosis` suggests ICMP may be blocked and to check other protocols.

**Scenario 10: API Invalid JSON**
*   **Preconditions:** Backend is running.
*   **Command:** `curl -X POST http://localhost:8000/runs -d '{"invalid": "json"}'`
*   **Expected Result:** `422 Unprocessable Entity` error from the API.

**Scenario 11: Asymmetric Routing**
*   **Preconditions:** MTR simulation shows different return paths or high loss on return.
*   **Command:** `isp-checker run --target asymmetric-route.net`
*   **Expected JSON:** `mtr` probe is `WARN` or `CRIT`. `diagnosis` suggests "Asymmetric Routing" as a possibility.

**Scenario 12: Rootless Probe Failure**
*   **Preconditions:** CLI is run by a non-root user and cannot create raw ICMP sockets.
*   **Command:** `isp-checker run --target 8.8.8.8` (as user 'nobody')
*   **Expected JSON:** `ping` and `traceroute` probes have status `NA` with a detail message like "Root privileges required." Other probes (dns, http) run normally. The score is calculated based on available data.
