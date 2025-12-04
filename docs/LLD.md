# Low-Level Design (LLD): ISP Health Checker

## 1. Introduction

This document provides the detailed technical specifications for the ISP Health Checker system. It defines the module boundaries, interfaces, data structures, API specifications, database schema, configuration management, and monitoring integration required for implementation.

## 2. Module Boundaries and Interfaces

### 2.1. CLI Probe Module

The CLI module is responsible for executing network diagnostic probes and generating standardized results. It consists of several sub-modules:

#### 2.1.1. Probe Interface

All probes implement a common interface defined in [`src/cli/probes/types.go`](src/cli/probes/types.go:3):

```go
type Result struct {
    Name      string                 `json:"name"`
    Status    string                 `json:"status"`
    LatencyMs float64                `json:"latency_ms,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Error     string                 `json:"error,omitempty"`
}

const (
    StatusOK   = "ok"
    StatusFail = "fail"
    StatusNA   = "na"
)
```

#### 2.1.2. Probe Implementations

Each probe type implements the following function signature:

```go
func RunProbe(ctx context.Context, target string) Result
```

Available probes:
- [`RunPing`](src/cli/probes/ping.go:14): ICMP ping test
- [`RunDNS`](src/cli/probes/dns.go): DNS resolution test
- [`RunTraceroute`](src/cli/probes/traceroute.go): Network path tracing

#### 2.1.3. Runner Module

The runner module coordinates probe execution:

```go
func ExecuteRun(ctx context.Context, opts RunOptions) (RunResult, error)
```

Where `RunOptions` and `RunResult` are defined in [`src/cli/runner.go`](src/cli/runner.go:18):

```go
type RunOptions struct {
    Target         string
    Mode           string
    OutPath        string
    SimulationPath string
}

type RunResult struct {
    RunID     string                   `json:"run_id"`
    Timestamp string                   `json:"timestamp"`
    Target    string                   `json:"target"`
    Mode      string                   `json:"mode"`
    Score     float64                  `json:"score"`
    Summary   string                   `json:"summary"`
    Probes    []probes.Result          `json:"probes"`
    Diagnosis []map[string]interface{} `json:"diagnosis"`
    Raw       map[string]interface{}   `json:"raw,omitempty"`
}
```

### 2.2. Backend Service Module

The backend service provides REST API endpoints for storing and retrieving diagnostic reports.

#### 2.2.1. Data Models

The backend uses Pydantic models defined in [`src/backend/models.py`](src/backend/models.py:1):

```python
class Probe(BaseModel):
    name: ProbeName
    status: Status
    details: Dict[str, Any] = Field(default_factory=dict)

class Diagnosis(BaseModel):
    component: DiagnosisComponent
    confidence: float
    explanation: str
    suggested_action: str

class Run(BaseModel):
    run_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    timestamp: datetime.datetime = Field(default_factory=datetime.datetime.utcnow)
    target: str
    mode: str
    score: float
    summary: str
    
    probes: List[Probe] = Field(default_factory=list)
    diagnosis: List[Diagnosis] = Field(default_factory=list)
    raw: Optional[Dict[str, Any]] = Field(default_factory=dict)
```

#### 2.2.2. Database Interface

The database interface is defined in [`src/backend/db/database.py`](src/backend/db/database.py:1):

```python
from databases import Database

database = Database(DATABASE_URL)

# Table schema
runs = Table(
    "runs",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("run_id", String, unique=True, index=True),
    Column("timestamp", DateTime),
    Column("target", String, index=True),
    Column("score", Float),
    Column("summary", String),
    Column("report", JSON), # Storing the full JSON report
)
```

## 3. IPC Protocols and JSON Payload Format

### 3.1. Standard JSON Result Schema

The CLI and backend communicate using a standardized JSON schema based on the `RunResult` structure:

```json
{
  "run_id": "uuid-string",
  "timestamp": "2025-12-03T21:42:38Z",
  "target": "8.8.8.8",
  "mode": "full",
  "score": 85.5,
  "summary": "Connection appears healthy with minor latency",
  "probes": [
    {
      "name": "ping",
      "status": "ok",
      "latency_ms": 25.3,
      "details": {
        "packets_sent": 3,
        "packets_received": 3,
        "packet_loss": 0
      }
    },
    {
      "name": "dns",
      "status": "ok",
      "latency_ms": 15.2,
      "details": {
        "resolution_time": "15.2ms",
        "resolved_ip": "8.8.8.8"
      }
    }
  ],
  "diagnosis": [
    {
      "component": "LocalNetwork",
      "confidence": 0.9,
      "explanation": "All probes completed successfully",
      "suggested_action": "No action required"
    }
  ],
  "raw": {
    "ping_output": "PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data...",
    "dns_output": ";; ->>HEADER<<- opcode: QUERY, status: NOERROR..."
  }
}
```

### 3.2. CLI-to-Backend Communication

The CLI submits results to the backend via HTTP POST to `/api/v1/runs` with the JSON payload above. The backend responds with:

```json
{
  "run_id": "uuid-string",
  "status": "accepted"
}
```

## 4. API Specifications

### 4.1. FastAPI Endpoints

The backend implements the following REST API endpoints:

#### 4.1.1. Submit Run

```
POST /api/v1/runs
Content-Type: application/json
Authorization: Bearer <api_key>

Request Body: Run JSON schema
Response: 202 Accepted
{
  "run_id": "uuid-string",
  "status": "accepted"
}
```

#### 4.1.2. List Recent Runs

```
GET /api/v1/runs?limit=20&offset=0
Authorization: Bearer <api_key>

Response: 200 OK
[
  {
    "run_id": "uuid-string",
    "timestamp": "2025-12-03T21:42:38Z",
    "target": "8.8.8.8",
    "mode": "full",
    "score": 85.5,
    "summary": "Connection appears healthy with minor latency"
  }
]
```

#### 4.1.3. Get Run Details

```
GET /api/v1/runs/{run_id}
Authorization: Bearer <api_key>

Response: 200 OK
Full Run JSON schema
```

#### 4.1.4. Get Raw Probe Output

```
GET /api/v1/runs/{run_id}/raw
Authorization: Bearer <api_key>

Response: 200 OK
{
  "ping_output": "PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data...",
  "dns_output": ";; ->>HEADER<<- opcode: QUERY, status: NOERROR...",
  "traceroute_output": "traceroute to 8.8.8.8 (8.8.8.8)..."
}
```

### 4.2. Authentication

All API endpoints require authentication via API key in the Authorization header:

```
Authorization: Bearer <api_key>
```

API keys are stored in the database with associated user accounts.

## 5. Database ERD

### 5.1. Entity-Relationship Diagram

```mermaid
erDiagram
    Users ||--o{ APIKeys : has
    Users ||--o{ Runs : creates
    Runs ||--o{ Probes : contains
    
    Users {
        int id PK
        string username UK
        string email UK
        string password_hash
        datetime created_at
        datetime updated_at
        boolean is_active
    }
    
    APIKeys {
        int id PK
        string key UK
        int user_id FK
        string name
        datetime created_at
        datetime expires_at
        boolean is_active
    }
    
    Runs {
        int id PK
        string run_id UK
        int user_id FK
        datetime timestamp
        string target
        string mode
        float score
        string summary
        json report
        datetime created_at
    }
    
    Probes {
        int id PK
        int run_id FK
        string name
        string status
        float latency_ms
        json details
        text error
    }
```

### 5.2. Table Definitions

#### 5.2.1. Users Table

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);
```

#### 5.2.2. API Keys Table

```sql
CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);
```

#### 5.2.3. Runs Table

```sql
CREATE TABLE runs (
    id SERIAL PRIMARY KEY,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    target VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL,
    score FLOAT NOT NULL,
    summary TEXT,
    report JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### 5.2.4. Probes Table

```sql
CREATE TABLE probes (
    id SERIAL PRIMARY KEY,
    run_id INTEGER REFERENCES runs(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    latency_ms FLOAT,
    details JSONB,
    error TEXT
);
```

### 5.3. Indexes

```sql
CREATE INDEX idx_runs_user_id ON runs(user_id);
CREATE INDEX idx_runs_timestamp ON runs(timestamp DESC);
CREATE INDEX idx_runs_target ON runs(target);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_probes_run_id ON probes(run_id);
```

## 6. Configuration Management

### 6.1. Environment Variables

The application uses the following environment variables:

#### 6.1.1. Backend Configuration

```
DATABASE_URL=postgresql://user:password@localhost/isp_health_checker
SECRET_KEY=your-secret-key
DEBUG=false
LOG_LEVEL=info
PROMETHEUS_PORT=9090
```

#### 6.1.2. CLI Configuration

```
ISP_CHECKER_API_URL=http://localhost:8000
ISP_CHECKER_API_KEY=your-api-key
ISP_CHECKER_TIMEOUT=30s
ISP_CHECKER_OUTPUT_FORMAT=json
```

### 6.2. Configuration Files

The application supports configuration via YAML files:

#### 6.2.1. Backend Config (`config.yaml`)

```yaml
database:
  url: ${DATABASE_URL}
  pool_size: 10
  max_overflow: 20

server:
  host: "0.0.0.0"
  port: 8000
  debug: false

logging:
  level: "info"
  format: "json"

prometheus:
  enabled: true
  port: 9090
  path: "/metrics"

auth:
  secret_key: ${SECRET_KEY}
  algorithm: "HS256"
  token_expiry: 86400  # 24 hours

retention:
  days: 30
  cleanup_interval: 86400  # 24 hours
```

#### 6.2.2. CLI Config (`~/.isp-checker/config.yaml`)

```yaml
api:
  url: "http://localhost:8000"
  key: "${ISP_CHECKER_API_KEY}"
  timeout: "30s"

probes:
  ping:
    count: 3
    timeout: "4s"
  dns:
    timeout: "5s"
    server: "8.8.8.8"
  traceroute:
    timeout: "30s"
    max_hops: 30

output:
  format: "json"
  include_raw: true

metrics:
  enabled: true
  port: 8080
  path: "/metrics"
```

### 6.3. Configuration Loading

The application loads configuration in the following order of precedence:

1. Command-line arguments
2. Environment variables
3. Configuration file
4. Default values

## 7. Logging and Monitoring

### 7.1. Structured Logging

The application uses structured JSON logging with the following fields:

```json
{
  "timestamp": "2025-12-03T21:42:38Z",
  "level": "info",
  "service": "isp-checker-backend",
  "component": "api",
  "message": "Run submitted successfully",
  "run_id": "uuid-string",
  "user_id": 123,
  "target": "8.8.8.8",
  "duration_ms": 150
}
```

### 7.2. Prometheus Metrics

The application exposes the following Prometheus metrics:

#### 7.2.1. CLI Metrics

```
# Probe execution duration
isp_checker_probe_duration_seconds{probe="ping",target="8.8.8.8"} 0.025

# Probe execution status
isp_checker_probe_status_total{probe="ping",status="ok"} 1

# Run execution duration
isp_checker_run_duration_seconds{mode="full"} 1.5

# Health score
isp_checker_health_score{target="8.8.8.8"} 85.5
```

#### 7.2.2. Backend Metrics

```
# API request duration
isp_checker_api_request_duration_seconds{endpoint="/api/v1/runs",method="POST"} 0.05

# API request count
isp_checker_api_requests_total{endpoint="/api/v1/runs",method="POST",status="202"} 1

# Database query duration
isp_checker_db_query_duration_seconds{query="insert_run"} 0.01

# Active runs
isp_checker_active_runs 5

# Runs stored
isp_checker_runs_stored_total 1000
```

### 7.3. Alerting Rules

The following Prometheus alerting rules are recommended:

```yaml
groups:
- name: isp-checker
  rules:
  - alert: HighFailureRate
    expr: rate(isp_checker_probe_status_total{status="fail"}[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High probe failure rate detected"
      
  - alert: LowHealthScore
    expr: isp_checker_health_score < 50
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Low health score detected"
      
  - alert: APIHighLatency
    expr: histogram_quantile(0.95, rate(isp_checker_api_request_duration_seconds_bucket[5m])) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "API response time is high"
```

## 8. Error Handling

### 8.1. CLI Error Handling

The CLI implements the following error handling strategy:

1. **Probe Errors**: Individual probe failures are captured in the Result struct
2. **Context Cancellation**: All probes respect context cancellation
3. **Timeout Handling**: Each probe has a configurable timeout
4. **Graceful Degradation**: System continues even if some probes fail

### 8.2. Backend Error Handling

The backend implements the following error handling strategy:

1. **Validation Errors**: 400 Bad Request for invalid input
2. **Authentication Errors**: 401 Unauthorized for invalid API keys
3. **Authorization Errors**: 403 Forbidden for insufficient permissions
4. **Not Found Errors**: 404 Not Found for missing resources
5. **Server Errors**: 500 Internal Server Error for unexpected failures

## 9. Security Considerations

### 9.1. API Security

1. **API Key Authentication**: All API endpoints require valid API keys
2. **Input Validation**: All inputs are validated using Pydantic models
3. **SQL Injection Prevention**: Parameterized queries are used for all database operations
4. **Rate Limiting**: API endpoints are rate-limited to prevent abuse

### 9.2. Data Protection

1. **Password Hashing**: User passwords are hashed using bcrypt
2. **API Key Encryption**: API keys are stored encrypted in the database
3. **Data Retention**: Old data is automatically purged based on retention policy
4. **Secure Headers**: Security headers are set on all API responses

## 10. Performance Considerations

### 10.1. CLI Performance

1. **Concurrent Probe Execution**: Probes are executed concurrently to minimize total runtime
2. **Context Cancellation**: Long-running operations can be cancelled
3. **Resource Limits**: Resource usage is limited to prevent system overload

### 10.2. Backend Performance

1. **Database Connection Pooling**: Database connections are pooled for efficiency
2. **Async Operations**: All I/O operations are asynchronous
3. **Caching**: Frequently accessed data is cached
4. **Pagination**: List endpoints support pagination to prevent large result sets

## 11. Deployment Considerations

### 11.1. Containerization

The application is containerized using Docker with the following considerations:

1. **Multi-stage Builds**: Minimize image size
2. **Non-root User**: Run containers as non-root user
3. **Health Checks**: Include health checks in container definitions
4. **Resource Limits**: Set appropriate resource limits

### 11.2. Kubernetes Deployment

The application can be deployed to Kubernetes with the following considerations:

1. **Liveness and Readiness Probes**: Include appropriate probes
2. **Resource Requests and Limits**: Set appropriate resource requests and limits
3. **Horizontal Pod Autoscaling**: Configure HPA based on CPU/memory usage
4. **Persistent Volumes**: Use persistent volumes for database storage