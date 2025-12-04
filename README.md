# ISP Health Checker

The ISP Health Checker is a powerful, open-source diagnostics tool for network professionals and enthusiasts. It helps you quickly identify and troubleshoot Internet connection problems by running a suite of tests and providing a clear, actionable diagnosis. Whether you're a NOC engineer managing a global network or an SMB admin trying to hold your ISP accountable, this tool gives you the data you need.

This repository contains the full source code for the CLI, backend API, and web UI, as well as all design and architecture documents.

## Features

*   **Multi-Probe Diagnostics:** Executes a battery of tests (Ping, Traceroute, DNS, HTTP, etc.).
*   **Automated Diagnosis & Scoring:** Generates a simple health score (0-100) and provides a high-level diagnosis.
*   **CLI-First Interface:** A powerful and scriptable command-line interface.
*   **Web UI Dashboard:** A minimal web interface for visualizing historical data.
*   **Prometheus Integration:** Exposes key metrics for monitoring and alerting.

## Getting Started

### Prerequisites

*   Docker and Docker Compose
*   Go (1.19+ for local CLI development)
*   Python (3.9+ for local backend development)
*   Node.js and npm (for local UI development)

### Quick Start with Docker Compose

This is the easiest way to get the entire application stack (backend, database, UI) running locally.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/your-repo/isp-health-checker.git
    cd isp-health-checker
    ```

2.  **Start the services:**
    ```sh
    docker-compose up --build
    ```

3.  **Access the services:**
    *   **Backend API Docs:** `http://localhost:8000/docs`
    *   **Web UI:** `http://localhost:3000`

### Building and Running the CLI

You can build the Go CLI tool locally to run on-demand checks.

1.  **Navigate to the CLI directory:**
    ```sh
    cd src/cli
    ```

2.  **Build the binary:**
    ```sh
    go build -o ../../build/isp-checker .
    ```

3.  **Run a health check:**
    ```sh
    ./build/isp-checker run --target 8.8.8.8 --output report.json
    ```

## Usage

### CLI Commands

*   `isp-checker run`: Run a health check.
    *   `--target`: The hostname or IP to test against.
    *   `--output`: Path to save the JSON result.
    *   `--mode`: `live` or `simulation`.
*   `isp-checker serve`: Expose a Prometheus metrics endpoint on `localhost:8080`.

### Example CLI Run

```sh
# Run a full test against 8.8.8.8 and save the result
./build/isp-checker run --target 8.8.8.8 --output ./results/8.8.8.8-2025-12-03.json

# Submit the result to the backend
curl -X POST http://localhost:8000/api/v1/runs \
     -H "Content-Type: application/json" \
     --data @./results/8.8.8.8-2025-12-03.json
```

## Project Structure

```
.
├── charts/         # Helm chart for Kubernetes deployment
├── docker/         # Dockerfiles for each service
├── docs/           # All design, architecture, and runbook documents
├── simulations/    # JSON files for simulation mode
├── src/            # Source code
│   ├── backend/    # Python FastAPI backend
│   ├── cli/        # Go CLI tool
│   └── ui/         # React frontend
├── tests/          # Unit and integration tests
├── .github/        # GitHub Actions CI/CD workflows
└── docker-compose.yml
```

## Contributing

Please read `CONTRIBUTING.md` for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the `LICENSE` file for details.