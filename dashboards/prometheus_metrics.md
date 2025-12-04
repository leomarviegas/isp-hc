# Prometheus Metrics Specification

This document provides the definitive specification for all Prometheus metrics exposed by the ISP Health Checker system.

## 1. Metric Naming Convention

All metrics follow the standard Prometheus naming convention: `isp_hc_<component>_<metric>_<unit>`.

*   **Application:** `isp_hc` (ISP Health Checker)
*   **Component:** `cli`, `backend`, `probe`, `api`, `db`
*   **Metric:** Descriptive name of what is being measured
*   **Unit:** Standard units like `seconds`, `total`, `bytes`, `percent`

## 2. CLI Metrics

### 2.1. Core Run Metrics

| Metric Name                                   | Type      | Labels                      | Description                                                  |
| --------------------------------------------- | --------- | --------------------------- | ------------------------------------------------------------ |
| `isp_hc_cli_runs_total`                       | Counter   | `target`, `mode`            | Total number of health check runs initiated by the CLI.     |
| `isp_hc_cli_run_duration_seconds`             | Histogram | `target`, `mode`            | Histogram of the duration of health check runs.             |
| `isp_hc_cli_score`                            | Gauge     | `target`                    | The most recent health score (0-100), where 100 is healthy and 0 is down. |
| `isp_hc_cli_diagnosis_total`                  | Counter   | `target`, `component`       | Total number of times a specific component was diagnosed as faulty. |

### 2.2. Probe-Specific Metrics

| Metric Name                                   | Type      | Labels                      | Description                                                  |
| --------------------------------------------- | --------- | --------------------------- | ------------------------------------------------------------ |
| `isp_hc_cli_probe_status_total`               | Counter   | `target`, `probe`, `status` | Total count of probe results by status (ok, fail, na).       |
| `isp_hc_cli_probe_duration_seconds`           | Histogram | `target`, `probe`           | Histogram of probe execution duration in seconds.           |
| `isp_hc_cli_probe_details`                    | Gauge     | `target`, `probe`, `key`    | Provides specific numeric details from probes.               |

#### `isp_hc_cli_probe_details` Keys

The `key` label will vary by probe:

*   **`probe="ping"`**
    *   `key="loss_percent"`: Packet loss percentage (e.g., 5.5 for 5.5%).
    *   `key="latency_avg_ms"`: Average round-trip latency in milliseconds.
    *   `key="latency_max_ms"`: Maximum round-trip latency in milliseconds.
*   **`probe="dns"`**
    *   `key="resolution_time_ms"`: Time taken to resolve the domain in milliseconds.
*   **`probe="http"`**
    *   `key="response_time_ms"`: Time to first byte for the HTTP response.
    *   `key="status_code"`: The HTTP status code returned by the server.
*   **`probe="traceroute"`**
    *   `key="hop_count"`: The number of hops to the destination.

## 3. Backend API Metrics

### 3.1. HTTP API Metrics

| Metric Name                                   | Type      | Labels                      | Description                                                  |
| --------------------------------------------- | --------- | --------------------------- | ------------------------------------------------------------ |
| `isp_hc_api_requests_total`                   | Counter   | `method`, `endpoint`, `status` | Total number of API requests received.                       |
| `isp_hc_api_request_duration_seconds`         | Histogram | `method`, `endpoint`        | Histogram of API request handling duration in seconds.       |
| `isp_hc_api_active_connections`               | Gauge     | -                           | Current number of active API connections.                    |

### 3.2. Database Metrics

| Metric Name                                   | Type      | Labels                      | Description                                                  |
| --------------------------------------------- | --------- | --------------------------- | ------------------------------------------------------------ |
| `isp_hc_db_queries_total`                     | Counter   | `operation`, `status`       | Total number of database queries executed.                   |
| `isp_hc_db_query_duration_seconds`            | Histogram | `operation`                 | Histogram of database query duration in seconds.            |
| `isp_hc_db_connections_active`                | Gauge     | -                           | Current number of active database connections.               |
| `isp_hc_db_runs_stored_total`                 | Counter   | -                           | Total number of diagnostic runs stored in the database.      |

### 3.3. Worker Pool Metrics

| Metric Name                                   | Type      | Labels                      | Description                                                  |
| --------------------------------------------- | --------- | --------------------------- | ------------------------------------------------------------ |
| `isp_hc_workers_active`                       | Gauge     | -                           | Current number of active probe runner workers.               |
| `isp_hc_workers_queue_depth`                  | Gauge     | -                           | Current depth of the work queue (number of pending jobs).    |
| `isp_hc_workers_jobs_total`                   | Counter   | `status`                    | Total number of jobs processed by workers (completed, failed). |

## 4. Histogram Configurations

### 4.1. Latency Histograms

For all duration metrics (`*_duration_seconds`), use the following bucket configurations:

#### CLI Run Duration Buckets
```
buckets: [0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300]  # 0.1s to 5min
```

#### Probe Duration Buckets
```
buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]  # 1ms to 10s
```

#### API Request Duration Buckets
```
buckets: [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5]  # 1ms to 5s
```

#### Database Query Duration Buckets
```
buckets: [0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1]  # 0.1ms to 1s
```

### 4.2. Bucket Justification

- **CLI Run Duration**: Captures quick local checks (<1s) to comprehensive remote tests (>5min)
- **Probe Duration**: Covers fast DNS queries (ms) to slow traceroutes (seconds)
- **API Request**: Handles fast responses (<10ms) to slow operations (>1s)
- **Database Query**: Captures simple lookups (<1ms) to complex aggregations (>100ms)

## 5. Alert Rule Definitions

### 5.1. Critical Alerts

```yaml
groups:
- name: isp_hc_critical
  rules:
  - alert: HighPacketLoss
    expr: isp_hc_cli_probe_details{probe="ping", key="loss_percent"} > 10
    for: 2m
    labels:
      severity: critical
      service: isp-health-checker
    annotations:
      summary: "High packet loss detected for {{ .Labels.target }}"
      description: "Packet loss is {{ .Value }}% for target {{ .Labels.target }} over the last 2 minutes."

  - alert: ProbeFailureRate
    expr: rate(isp_hc_cli_probe_status_total{status="fail"}[5m]) > 0.05
    for: 5m
    labels:
      severity: critical
      service: isp-health-checker
    annotations:
      summary: "High probe failure rate detected"
      description: "Probe failure rate is {{ .Value | humanizePercentage }} over the last 5 minutes."

  - alert: LowHealthScore
    expr: isp_hc_cli_score < 30
    for: 5m
    labels:
      severity: critical
      service: isp-health-checker
    annotations:
      summary: "Very low health score for {{ .Labels.target }}"
      description: "Health score is {{ .Value }} for target {{ .Labels.target }} over the last 5 minutes."

  - alert: APIDown
    expr: up{job="isp-hc-backend"} == 0
    for: 1m
    labels:
      severity: critical
      service: isp-health-checker
    annotations:
      summary: "ISP Health Checker API is down"
      description: "The ISP Health Checker backend API has been down for more than 1 minute."
```

### 5.2. Warning Alerts

```yaml
groups:
- name: isp_hc_warning
  rules:
  - alert: ElevatedLatency
    expr: isp_hc_cli_probe_details{probe="ping", key="latency_avg_ms"} > 100
    for: 5m
    labels:
      severity: warning
      service: isp-health-checker
    annotations:
      summary: "Elevated latency for {{ .Labels.target }}"
      description: "Average latency is {{ .Value }}ms for target {{ .Labels.target }} over the last 5 minutes."

  - alert: APIHighLatency
    expr: histogram_quantile(0.95, rate(isp_hc_api_request_duration_seconds_bucket[5m])) > 1
    for: 5m
    labels:
      severity: warning
      service: isp-health-checker
    annotations:
      summary: "High API response latency"
      description: "95th percentile API response time is {{ .Value }}s over the last 5 minutes."

  - alert: DatabaseSlowQueries
    expr: histogram_quantile(0.95, rate(isp_hc_db_query_duration_seconds_bucket[5m])) > 0.1
    for: 5m
    labels:
      severity: warning
      service: isp-health-checker
    annotations:
      summary: "Slow database queries detected"
      description: "95th percentile database query time is {{ .Value }}s over the last 5 minutes."

  - alert: WorkerQueueBacklog
    expr: isp_hc_workers_queue_depth > 10
    for: 5m
    labels:
      severity: warning
      service: isp-health-checker
    annotations:
      summary: "Worker queue backlog detected"
      description: "Worker queue depth is {{ .Value }} jobs over the last 5 minutes."
```

## 6. Scrape Configuration

### 6.1. Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "isp_hc_critical.yml"
  - "isp_hc_warning.yml"

scrape_configs:
  # CLI metrics (when running in serve mode)
  - job_name: 'isp-hc-cli'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
    metrics_path: /metrics
    honor_labels: true

  # Backend API metrics
  - job_name: 'isp-hc-backend'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
    metrics_path: /metrics
    honor_labels: true

  # Production deployment example
  - job_name: 'isp-hc-cli-prod'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - monitoring
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: isp-hc-cli
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics

  - job_name: 'isp-hc-backend-prod'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - monitoring
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: isp-hc-backend
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics
```

### 6.2. Docker Compose Example

```yaml
# docker-compose.metrics.yml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts:/etc/prometheus/rules
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=30d'
      - '--web.enable-lifecycle'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana

volumes:
  grafana-storage:
```

## 7. Retention Policies

### 7.1. Metrics Retention

| Metric Category                | Retention Period | Rationale                                                     |
| ------------------------------ | ---------------- | ------------------------------------------------------------- |
| Health scores                  | 90 days          | Track long-term trends and identify recurring issues          |
| Probe results                  | 30 days          | Detailed probe data for troubleshooting recent issues         |
| API performance metrics        | 30 days          | Monitor API performance and identify bottlenecks              |
| Database metrics               | 30 days          | Track database performance and query optimization             |
| Worker pool metrics            | 14 days          | Monitor worker utilization and queue management               |
| Raw histograms (high-res)      | 7 days           | High-resolution data for detailed performance analysis        |
| Aggregated histograms          | 90 days          | Long-term performance trend analysis                          |

### 7.2. Data Retention Implementation

```yaml
# Prometheus retention configuration
global:
  # Default retention for all metrics
  retention_time: 30d

# Override retention for specific metrics using metric_relabel_configs
scrape_configs:
  - job_name: 'isp-hc-cli'
    metric_relabel_configs:
      # Keep health scores longer
      - source_labels: [__name__]
        regex: 'isp_hc_cli_score'
        target_label: __tmp_retention
        replacement: '90d'
      
      # Keep probe results for standard period
      - source_labels: [__name__]
        regex: 'isp_hc_cli_probe_.*'
        target_label: __tmp_retention
        replacement: '30d'
```

### 7.3. Database Data Retention

For the PostgreSQL database storing diagnostic reports:

```sql
-- Create a partitioned table for time-based retention
CREATE TABLE runs (
    id SERIAL,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    target VARCHAR(255) NOT NULL,
    score FLOAT,
    report JSONB,
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create monthly partitions
CREATE TABLE runs_2025_12 PARTITION OF runs
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Automated retention policy (run daily)
DELETE FROM runs WHERE timestamp < NOW() - INTERVAL '90 days';
```

## 8. Sample Metrics Exposition

### 8.1. CLI Metrics Example

```
# HELP isp_hc_cli_runs_total Total number of health check runs initiated by the CLI.
# TYPE isp_hc_cli_runs_total counter
isp_hc_cli_runs_total{target="8.8.8.8",mode="live"} 15
isp_hc_cli_runs_total{target="1.1.1.1",mode="live"} 12

# HELP isp_hc_cli_score The most recent health score (0-100), where 100 is healthy and 0 is down.
# TYPE isp_hc_cli_score gauge
isp_hc_cli_score{target="8.8.8.8"} 92.0
isp_hc_cli_score{target="1.1.1.1"} 55.0

# HELP isp_hc_cli_probe_status_total Total count of probe results by status.
# TYPE isp_hc_cli_probe_status_total counter
isp_hc_cli_probe_status_total{target="1.1.1.1",probe="ping",status="ok"} 10
isp_hc_cli_probe_status_total{target="1.1.1.1",probe="ping",status="fail"} 2

# HELP isp_hc_cli_probe_details Provides specific numeric details from probes.
# TYPE isp_hc_cli_probe_details gauge
isp_hc_cli_probe_details{target="8.8.8.8",probe="ping",key="loss_percent"} 0.0
isp_hc_cli_probe_details{target="8.8.8.8",probe="ping",key="latency_avg_ms"} 10.5
isp_hc_cli_probe_details{target="1.1.1.1",probe="ping",key="loss_percent"} 6.0
isp_hc_cli_probe_details{target="1.1.1.1",probe="ping",key="latency_avg_ms"} 85.3
```

### 8.2. Backend Metrics Example

```
# HELP isp_hc_api_requests_total Total number of API requests received.
# TYPE isp_hc_api_requests_total counter
isp_hc_api_requests_total{method="POST",endpoint="/api/v1/runs",status="202"} 150
isp_hc_api_requests_total{method="GET",endpoint="/api/v1/runs",status="200"} 300

# HELP isp_hc_db_runs_stored_total Total number of diagnostic runs stored in the database.
# TYPE isp_hc_db_runs_stored_total counter
isp_hc_db_runs_stored_total 1250

# HELP isp_hc_workers_active Current number of active probe runner workers.
# TYPE isp_hc_workers_active gauge
isp_hc_workers_active 3
```

## 9. Implementation Notes

### 9.1. Metric Cardinality Considerations

- The `target` label can have high cardinality. Consider using `target_group` for aggregation in large deployments.
- Use `recording rules` to pre-compute expensive aggregations.
- Monitor metric cardinality to avoid performance issues.

### 9.2. Security Considerations

- Metrics endpoints should not expose sensitive information (API keys, passwords).
- Consider using authentication for metrics endpoints in production.
- Network policies should restrict access to metrics endpoints.

### 9.3. Performance Optimization

- Use histograms instead of summaries for quantile calculations.
- Implement metric caching where appropriate.
- Consider using exemplars for trace correlation.