# Product Brief: ISP Health Checker

## 1. Overview

**ISP Health Checker** is a diagnostics and troubleshooting tool designed to detect, diagnose, and report instabilities on ISP connections. It provides network administrators and support personnel with a suite of tools to identify issues like packet loss, latency spikes, DNS problems, and upstream outages, enabling faster resolution and improved network reliability.

## 2. Target Personas

*   **NOC Engineer:** Responsible for monitoring large-scale network infrastructure. Needs a reliable, scriptable tool to integrate into existing monitoring systems. Values quick, accurate diagnostics and clear, actionable data to minimize Mean Time to Resolution (MTTR).
*   **ISP Support Technician:** Troubleshoots customer-reported connection issues. Needs a simple yet powerful tool to run on-demand checks from a customer's site or a central location to validate reported problems and pinpoint the source of failure (e.g., local network, last mile, or upstream).
*   **Small-to-Medium Business (SMB) Admin:** A "jack-of-all-trades" IT professional managing a business's network. Needs an easy-to-use tool to verify ISP service quality, gather evidence for support tickets, and rule out the local network as the source of a problem.

## 3. Core Features

*   **Multi-Probe Diagnostics:** Executes a battery of tests (Ping, Traceroute, MTR, DNS, HTTP, Speedtest, BGP) to build a comprehensive picture of connection health.
*   **Automated Diagnosis & Scoring:** Analyzes probe results to generate a simple health score (0-100) and provides a high-level diagnosis of the likely problem area (e.g., Local Network, DNS, Upstream Transit).
*   **CLI-First Interface:** A powerful and scriptable command-line interface for running checks, viewing results, and exporting data.
*   **Web UI Dashboard:** A minimal web interface for visualizing historical run data, viewing detailed reports, and sharing results.
*   **Simulation Mode:** Allows for running the tool with pre-defined scenarios to test a-nd validate the tool's behavior without requiring a live network connection.
*   **Prometheus Integration:** Exposes key metrics for integration with standard monitoring and alerting stacks.

## 4. Primary User Flows

### Flow 1: Ad-Hoc Triage (CLI)

1.  A **NOC Engineer** receives an alert for a potential issue at a remote site.
2.  She runs `isp-checker run --target <problem-ip> --out report.json` from a jump box.
3.  She inspects the `summary` and `diagnosis` fields in `report.json` to get an immediate assessment.
4.  If the issue points to an upstream provider, she attaches the JSON report to an escalation ticket.

### Flow 2: Scheduled Health Monitoring (CLI + Backend)

1.  An **SMB Admin** sets up a cron job to run `isp-checker run --target 8.8.8.8` every 15 minutes and POSTs the result to the backend API.
2.  He configures alerts in Grafana based on the `isp_checker_score` metric exposed by the backend.
3.  When an alert fires, he logs into the Web UI to view the detailed drilldown for the failed run and compares it with historical trends.

## 5. README Introduction

The ISP Health Checker is a powerful, open-source diagnostics tool for network professionals and enthusiasts. It helps you quickly identify and troubleshoot Internet connection problems by running a suite of tests and providing a clear, actionable diagnosis. Whether you're a NOC engineer managing a global network or an SMB admin trying to hold your ISP accountable, this tool gives you the data you need.

## 6. CLI Help Messages

### `isp-checker run`

```
USAGE:
    isp-checker run [OPTIONS]

DESCRIPTION:
    Runs a comprehensive health check against a target host or IP address.

OPTIONS:
    --target <hostname-or-ip>   The destination to test against (e.g., 8.8.8.8).
    --output <file-path>        Path to save the JSON result file.
    --mode <live|simulation>    Run in 'live' mode or use a 'simulation' file.
    --summary                   Display a short summary to stdout.
```

### `isp-checker analyze`

```
USAGE:
    isp-checker analyze [OPTIONS] <json-file>

DESCRIPTION:
    Analyzes a previously generated JSON report to provide additional insights or comparisons.

OPTIONS:
    --compare <other-json-file>   Compares two reports to identify changes.
    --format <text|html>          Sets the output format for the analysis.
```

### `isp-checker report`

```
USAGE:
    isp-checker report [OPTIONS] <run-id>

DESCRIPTION:
    Fetches and displays a report from a running backend instance.

OPTIONS:
    --api-url <url>             The base URL of the ISP Checker backend.
    --raw                       Fetch the raw probe output in addition to the report.
```
