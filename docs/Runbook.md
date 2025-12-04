# Operational Runbook & Playbooks

This document provides operational guidance for NOC engineers and support staff on how to interpret and act on results from the ISP Health Checker. It covers failure identification, automated remediation, incident management, performance tuning, backup procedures, and maintenance tasks.

## Table of Contents

1. [Failure Signatures](#1-failure-signatures)
2. [Automated Remediation](#2-automated-remediation)
3. [Incident Management Workflow](#3-incident-management-workflow)
4. [Performance Tuning](#4-performance-tuning)
5. [Backup & Disaster Recovery](#5-backup--disaster-recovery)
6. [Maintenance Procedures](#6-maintenance-procedures)

---

## 1. Failure Signatures

Each failure signature is a pattern of probe results that points to a specific type of problem. Understanding these patterns is crucial for quick diagnosis and resolution.

### 1.1. High Latency Issues

#### Symptoms:
- **Metrics**: `isp_checker_probe_details{key="latency_avg_ms"}` > 150ms for sustained periods
- **Ping Probe**: `WARN` or `CRIT` status with high latency values
- **Traceroute**: Shows latency increasing progressively toward the destination
- **Health Score**: `isp_checker_score` between 40-70 (degraded performance)

#### Log Patterns:
```json
{
  "probes": [
    {
      "name": "ping",
      "status": "warn",
      "details": {
        "latency_avg_ms": 185.3,
        "latency_max_ms": 295.7,
        "packet_loss": 0
      }
    }
  ],
  "diagnosis": [
    {
      "component": "Transit",
      "confidence": 0.8,
      "explanation": "High latency detected in transit path",
      "suggested_action": "Investigate transit provider performance"
    }
  ]
}
```

#### Investigation Steps:
1. Check if latency is consistent across multiple targets
2. Analyze traceroute data to identify problematic hops
3. Verify if latency affects all services or specific destinations
4. Compare with baseline metrics from the same time period

### 1.2. Packet Loss Issues

#### Symptoms:
- **Metrics**: `isp_checker_probe_details{key="loss_percent"}` > 2%
- **Ping Probe**: `WARN` (2-10% loss) or `CRIT` (>10% loss)
- **Health Score**: `isp_checker_score` between 30-80 depending on loss percentage
- **Traceroute**: Shows asterisks (*) at specific hops indicating packet loss

#### Log Patterns:
```json
{
  "probes": [
    {
      "name": "ping",
      "status": "crit",
      "details": {
        "packets_sent": 10,
        "packets_received": 7,
        "packet_loss": 30
      }
    }
  ],
  "diagnosis": [
    {
      "component": "Peering/Transit",
      "confidence": 0.9,
      "explanation": "Significant packet loss detected at hop 4",
      "suggested_action": "Escalate to ISP with traceroute evidence"
    }
  ]
}
```

#### Investigation Steps:
1. Determine if loss is consistent or intermittent
2. Identify the hop where loss begins in traceroute
3. Test against multiple destinations to isolate the issue
4. Check for correlation with network utilization peaks

### 1.3. DNS Resolution Failures

#### Symptoms:
- **Metrics**: `isp_checker_probe_status{probe="dns",status="crit"}` = 1
- **DNS Probe**: `CRIT` status with timeout or NXDOMAIN errors
- **HTTP Probes**: May fail if they rely on hostname resolution
- **Health Score**: `isp_checker_score` > 60 if DNS is critical

#### Log Patterns:
```json
{
  "probes": [
    {
      "name": "dns",
      "status": "crit",
      "error": "timeout: no nameservers responded"
    },
    {
      "name": "ping",
      "status": "ok",
      "details": {
        "target_ip": "8.8.8.8"
      }
    }
  ],
  "diagnosis": [
    {
      "component": "DNS",
      "confidence": 0.95,
      "explanation": "DNS resolution failure detected",
      "suggested_action": "Switch to alternative DNS resolvers"
    }
  ]
}
```

#### Investigation Steps:
1. Test DNS resolution against multiple servers (8.8.8.8, 1.1.1.1)
2. Verify local DNS configuration
3. Check if the issue affects specific domains or all resolutions
4. Test with direct IP connectivity to confirm network is functional

### 1.4. Database Connection Errors

#### Symptoms:
- **Backend Logs**: Connection timeout or authentication errors
- **Metrics**: `isp_checker_api_requests_total{status="5xx"}` increasing
- **API Response**: 500 Internal Server Error on `/api/v1/runs`
- **Health Score**: Not available due to data storage failure

#### Log Patterns:
```json
{
  "timestamp": "2025-12-03T21:42:38Z",
  "level": "error",
  "service": "isp-checker-backend",
  "message": "Database connection failed",
  "error": "connection to server at \"db\" (172.20.0.2), port 5432 failed: Connection timed out"
}
```

#### Investigation Steps:
1. Check database pod status and resource utilization
2. Verify network connectivity between backend and database
3. Check database credentials and connection string
4. Examine database logs for connection issues or resource constraints

### 1.5. Complete Service Outage

#### Symptoms:
- **All Probes**: `CRIT` status with 100% failure rate
- **Health Score**: `isp_checker_score` = 100
- **Multiple Targets**: Failures across all monitored destinations
- **Traceroute**: Timeouts after initial hops

#### Log Patterns:
```json
{
  "probes": [
    {
      "name": "ping",
      "status": "crit",
      "error": "100% packet loss"
    },
    {
      "name": "dns",
      "status": "crit",
      "error": "Network unreachable"
    }
  ],
  "diagnosis": [
    {
      "component": "Upstream",
      "confidence": 0.98,
      "explanation": "Complete connectivity failure detected",
      "suggested_action": "Verify ISP status and local network equipment"
    }
  ]
}
```

#### Investigation Steps:
1. Check local network equipment (routers, switches)
2. Verify ISP status page and social media for outage reports
3. Test from alternative network paths (mobile hotspot)
4. Check for widespread issues affecting multiple users

---

## 2. Automated Remediation

The ISP Health Checker includes automated remediation capabilities that can self-heal common issues. These are triggered based on specific failure patterns and confidence levels.

### 2.1. Retry Logic Implementation

#### Configuration:
```yaml
retry_policy:
  max_attempts: 3
  backoff_factor: 2
  initial_delay: 1s
  max_delay: 30s
  retryable_errors:
    - timeout
    - connection_refused
    - temporary_failure
```

#### Implementation:
- **Transient Failures**: Automatically retry with exponential backoff
- **Network Timeouts**: Increase timeout values for subsequent attempts
- **DNS Failures**: Fall back to secondary DNS servers
- **Database Timeouts**: Retry with increased connection timeout

### 2.2. Circuit Breaker Pattern

#### Configuration:
```yaml
circuit_breaker:
  failure_threshold: 5
  recovery_timeout: 60s
  half_open_max_calls: 3
  success_threshold: 2
```

#### Behavior:
1. **Closed State**: Normal operation, tracking failures
2. **Open State**: Fail fast without attempting operations after threshold
3. **Half-Open State**: Limited calls to test service recovery
4. **Recovery**: Return to closed state if successful

#### Implementation:
```python
class CircuitBreaker:
    def __init__(self, failure_threshold=5, recovery_timeout=60):
        self.failure_threshold = failure_threshold
        self.recovery_timeout = recovery_timeout
        self.failure_count = 0
        self.last_failure_time = None
        self.state = "closed"  # closed, open, half_open
    
    def call(self, func, *args, **kwargs):
        if self.state == "open":
            if time.time() - self.last_failure_time > self.recovery_timeout:
                self.state = "half_open"
            else:
                raise Exception("Circuit breaker is open")
        
        try:
            result = func(*args, **kwargs)
            if self.state == "half_open":
                self.state = "closed"
                self.failure_count = 0
            return result
        except Exception as e:
            self.failure_count += 1
            self.last_failure_time = time.time()
            if self.failure_count >= self.failure_threshold:
                self.state = "open"
            raise
```

### 2.3. Self-Healing Procedures

#### Database Connection Pool Recovery:
```python
async def recover_database_pool():
    """Automatically recover from database connection issues"""
    try:
        # Test current connection
        await database.execute("SELECT 1")
    except Exception as e:
        logger.warning(f"Database connection failed: {e}")
        
        # Close existing connections
        await database.disconnect()
        
        # Reconnect with exponential backoff
        for attempt in range(3):
            try:
                await database.connect()
                logger.info("Database connection recovered")
                return True
            except Exception as e:
                wait_time = 2 ** attempt
                logger.warning(f"Reconnection attempt {attempt + 1} failed: {e}")
                await asyncio.sleep(wait_time)
        
        logger.error("Failed to recover database connection")
        return False
```

#### DNS Resolver Failover:
```python
def get_dns_resolvers():
    """Get list of DNS resolvers with fallback"""
    primary = ["8.8.8.8", "1.1.1.1"]  # Primary resolvers
    fallback = ["9.9.9.9", "208.67.222.222"]  # Fallback resolvers
    return primary + fallback

async def resolve_with_fallback(hostname):
    """Resolve hostname with fallback resolvers"""
    for resolver in get_dns_resolvers():
        try:
            resolver = dns.resolver.Resolver()
            resolver.nameservers = [resolver]
            result = resolver.resolve(hostname, 'A')
            return [ip.to_text() for ip in result]
        except Exception as e:
            logger.warning(f"DNS resolution failed with {resolver}: {e}")
            continue
    
    raise Exception("All DNS resolvers failed")
```

### 2.4. Automated Scaling

#### Horizontal Pod Autoscaler Configuration:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: isp-checker-backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: isp-checker-backend
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

---

## 3. Incident Management Workflow

This section defines the standard operating procedure for handling incidents detected by the ISP Health Checker system.

### 3.1. Incident Triage Process

#### 3.1.1. Alert Reception and Initial Assessment

1. **Alert Receipt**:
   - Monitor Prometheus alerts via Alertmanager
   - Check PagerDuty/Opsgenie for high-priority incidents
   - Review Slack notifications for lower-severity issues

2. **Initial Assessment** (First 5 minutes):
   ```bash
   # Check system health dashboard
   kubectl get pods -n isp-checker
   kubectl top pods -n isp-checker
   
   # Check recent metrics
   curl -s "http://prometheus:9090/api/v1/query?query=isp_checker_score" | jq .
   
   # Review recent logs
   kubectl logs -n isp-checker -l app=backend --tail=100
   ```

3. **Severity Classification**:
   - **Critical (P1)**: Complete service outage, health score > 80
   - **High (P2)**: Significant degradation, health score 40-80
   - **Medium (P3)**: Minor issues, health score 20-40
   - **Low (P4)**: Informational, health score < 20

#### 3.1.2. Incident Documentation

Create incident ticket with following template:
```markdown
## Incident Summary
- **ID**: INC-{timestamp}
- **Severity**: {P1/P2/P3/P4}
- **Start Time**: {timestamp}
- **Reporter**: {name}
- **Affected Services**: ISP Health Checker

## Initial Assessment
- **Health Score**: {current_score}
- **Affected Targets**: {list_of_targets}
- **Primary Symptoms**: {description}
- **Customer Impact**: {description}

## Initial Actions
- [ ] Checked pod status
- [ ] Reviewed recent logs
- [ ] Verified database connectivity
- [ ] Checked network connectivity

## Next Steps
- [ ] Investigate root cause
- [ ] Implement temporary fix
- [ ] Plan permanent resolution
```

### 3.2. Investigation Process

#### 3.2.1. Data Collection

1. **System Metrics**:
   ```bash
   # Export relevant metrics for analysis
   curl -G "http://prometheus:9090/api/v1/query_range" \
     -d "query=isp_checker_score" \
     -d "start=$(date -d '2 hours ago' +%s)" \
     -d "end=$(date +%s)" \
     -d "step=60" > metrics.json
   ```

2. **Log Analysis**:
   ```bash
   # Collect logs from all components
   kubectl logs -n isp-checker deployment/backend --since=2h > backend.log
   kubectl logs -n isp-checker deployment/ui --since=2h > ui.log
   kubectl logs -n isp-checker deployment/db --since=2h > db.log
   ```

3. **Network Diagnostics**:
   ```bash
   # Run manual diagnostics from affected pod
   kubectl exec -n isp-checker deployment/backend -- \
     /app/isp-checker run --target 8.8.8.8 --mode live --output manual-check.json
   ```

#### 3.2.2. Root Cause Analysis

1. **Component Isolation**:
   - Check if issue is with CLI, backend, database, or infrastructure
   - Verify if problem affects all targets or specific ones
   - Determine if issue is internal or external to the system

2. **Timeline Reconstruction**:
   ```bash
   # Create timeline of events
   grep "ERROR\|WARN" backend.log | \
     awk '{print $1 " " $2 " " $3}' | \
     sort -u > timeline.txt
   ```

3. **Correlation Analysis**:
   - Check for correlation with deployments or configuration changes
   - Review recent Git commits and CI/CD pipeline runs
   - Analyze network provider maintenance windows

### 3.3. Resolution Process

#### 3.3.1. Immediate Mitigation

1. **Service Restoration**:
   ```bash
   # Restart affected services
   kubectl rollout restart deployment/backend -n isp-checker
   kubectl rollout status deployment/backend -n isp-checker
   
   # Scale up if resource constrained
   kubectl scale deployment/backend --replicas=5 -n isp-checker
   ```

2. **Configuration Rollback**:
   ```bash
   # Rollback to previous deployment
   kubectl rollout undo deployment/backend -n isp-checker
   ```

3. **Emergency Patches**:
   ```bash
   # Apply hotfix
   kubectl patch deployment backend -n isp-checker -p '{"spec":{"template":{"spec":{"containers":[{"name":"backend","env":[{"name":"EMERGENCY_MODE","value":"true"}]}]}}}}'
   ```

#### 3.3.2. Permanent Resolution

1. **Code Fixes**:
   - Create feature branch with fix
   - Implement automated tests for the issue
   - Submit pull request for review
   - Deploy through CI/CD pipeline

2. **Configuration Updates**:
   - Review and update resource limits
   - Adjust timeout and retry parameters
   - Update monitoring thresholds

3. **Infrastructure Changes**:
   - Scale database resources if needed
   - Update network policies
   - Implement additional monitoring

### 3.4. Post-Incident Review

#### 3.4.1. Incident Report Template

```markdown
# Post-Incident Report: INC-{id}

## Executive Summary
- **Duration**: {start_time} to {end_time} ({total_time})
- **Impact**: {description of customer impact}
- **Root Cause**: {concise root cause}
- **Resolution**: {summary of fix applied}

## Timeline
| Time | Event | Owner |
|------|-------|-------|
| {timestamp} | Alert triggered | System |
| {timestamp} | Incident acknowledged | {name} |
| {timestamp} | Investigation started | {name} |
| {timestamp} | Root cause identified | {name} |
| {timestamp} | Fix implemented | {name} |
| {timestamp} | Service restored | {name} |

## Root Cause Analysis
### Primary Cause
{detailed explanation of root cause}

### Contributing Factors
{list of contributing factors}

### Detection Gaps
{why wasn't this detected earlier}

## Resolution Actions
### Immediate Actions
{list of immediate fixes applied}

### Long-term Actions
{list of permanent fixes planned}

## Lessons Learned
### What Went Well
{positive aspects of response}

### Areas for Improvement
{areas that need improvement}

### Action Items
| Item | Owner | Due Date | Status |
|------|-------|----------|--------|
| {action_item} | {owner} | {date} | {status} |
```

#### 3.4.2. Follow-up Tasks

1. **Technical Actions**:
   - [ ] Update monitoring and alerting rules
   - [ ] Implement automated tests for the failure scenario
   - [ ] Document new operational procedures
   - [ ] Review and update runbook sections

2. **Process Improvements**:
   - [ ] Conduct team training on the incident
   - [ ] Review on-call rotation and escalation procedures
   - [ ] Update incident response playbooks
   - [ ] Schedule follow-up review meeting

---

## 4. Performance Tuning

This section provides guidelines for optimizing the ISP Health Checker system performance based on workload patterns and resource utilization.

### 4.1. Probe Interval Optimization

#### 4.1.1. Interval Configuration

The frequency of health checks should be balanced between detection speed and resource consumption:

```yaml
probe_intervals:
  critical_targets: 60s    # Critical infrastructure
  important_targets: 300s   # Important services
  normal_targets: 900s      # Regular monitoring
  background_targets: 3600s # Low priority targets
```

#### 4.1.2. Adaptive Interval Adjustment

Implement adaptive intervals based on target health:

```python
def calculate_adaptive_interval(target_health_score, base_interval):
    """Adjust probe interval based on target health"""
    if target_health_score > 80:
        # Critical state - probe more frequently
        return max(base_interval // 4, 30)
    elif target_health_score > 40:
        # Degraded state - moderate frequency
        return max(base_interval // 2, 60)
    elif target_health_score > 20:
        # Warning state - normal frequency
        return base_interval
    else:
        # Healthy state - reduced frequency
        return min(base_interval * 2, 3600)
```

### 4.2. Database Connection Pool Tuning

#### 4.2.1. Connection Pool Configuration

```python
# Backend database configuration
DATABASE_CONFIG = {
    "pool_size": 20,              # Base connection pool size
    "max_overflow": 30,           # Additional connections under load
    "pool_timeout": 30,           # Timeout for getting connection
    "pool_recycle": 3600,         # Recycle connections after 1 hour
    "pool_pre_ping": True,        # Validate connections before use
}
```

#### 4.2.2. Monitoring Connection Pool Health

```python
def monitor_connection_pool():
    """Monitor database connection pool health"""
    pool = engine.pool
    
    metrics = {
        "pool_size": pool.size(),
        "checked_in": pool.checkedin(),
        "checked_out": pool.checkedout(),
        "overflow": pool.overflow(),
        "invalid": pool.invalid()
    }
    
    # Alert if pool utilization is high
    utilization = metrics["checked_out"] / (metrics["pool_size"] + metrics["overflow"])
    if utilization > 0.8:
        logger.warning(f"High connection pool utilization: {utilization:.2%}")
    
    return metrics
```

### 4.3. Resource Allocation Tuning

#### 4.3.1. Container Resource Limits

```yaml
# Backend deployment resources
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi

# Database resources
database:
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 4Gi
```

#### 4.3.2. Performance Monitoring

Key metrics to monitor:
- CPU utilization: `container_cpu_usage_seconds_total`
- Memory usage: `container_memory_working_set_bytes`
- Disk I/O: `container_fs_io_time_seconds_total`
- Network I/O: `container_network_receive_bytes_total`

### 4.4. Caching Strategy

#### 4.4.1. Response Caching

```python
from functools import lru_cache
import time

@lru_cache(maxsize=128)
def get_cached_target_status(target, cache_duration=300):
    """Cache target status for 5 minutes"""
    cache_key = f"{target}_{int(time.time() // cache_duration)}"
    return check_target_health(target)
```

#### 4.4.2. Database Query Optimization

```sql
-- Create indexes for common queries
CREATE INDEX CONCURRENTLY idx_runs_timestamp_target ON runs(timestamp DESC, target);
CREATE INDEX CONCURRENTLY idx_runs_score ON runs(score) WHERE score > 50;

-- Partition large tables by time
CREATE TABLE runs_2025_01 PARTITION OF runs
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### 4.5. Scaling Strategies

#### 4.5.1. Horizontal Scaling

```yaml
# Auto-scaling based on custom metrics
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: isp-checker-custom-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: isp-checker-backend
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: active_probes_per_pod
      target:
        type: AverageValue
        averageValue: "10"
```

#### 4.5.2. Vertical Scaling

Monitor resource usage and adjust container limits:

```bash
# Analyze resource usage patterns
kubectl top pods -n isp-checker --containers
kubectl describe pod <pod-name> -n isp-checker

# Update resource limits based on usage
kubectl patch deployment backend -n isp-checker -p '
{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "backend",
          "resources": {
            "requests": {"cpu": "300m", "memory": "384Mi"},
            "limits": {"cpu": "1500m", "memory": "768Mi"}
          }
        }]
      }
    }
  }
}'
```

---

## 5. Backup & Disaster Recovery

This section outlines procedures for backing up the ISP Health Checker system and restoring it in case of disaster.

### 5.1. Backup Strategy

#### 5.1.1. Database Backups

**Automated Daily Backups**:
```bash
#!/bin/bash
# backup_database.sh

BACKUP_DIR="/backups/database"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="isp_health_checker_${DATE}.sql"

# Create backup directory if it doesn't exist
mkdir -p $BACKUP_DIR

# Perform database backup
kubectl exec -n isp-checker deployment/db -- \
  pg_dump -U user isp_health_checker > "${BACKUP_DIR}/${BACKUP_FILE}"

# Compress backup
gzip "${BACKUP_DIR}/${BACKUP_FILE}"

# Upload to cloud storage (AWS S3 example)
aws s3 cp "${BACKUP_DIR}/${BACKUP_FILE}.gz" \
  s3://isp-checker-backups/database/

# Clean up local files older than 7 days
find $BACKUP_DIR -name "*.gz" -mtime +7 -delete

echo "Database backup completed: ${BACKUP_FILE}.gz"
```

**Point-in-Time Recovery Configuration**:
```sql
-- Enable WAL archiving for point-in-time recovery
ALTER SYSTEM SET wal_level = replica;
ALTER SYSTEM SET archive_mode = on;
ALTER SYSTEM SET archive_command = 'cp %p /backups/wal/%f';
SELECT pg_reload_conf();
```

#### 5.1.2. Configuration Backups

```bash
#!/bin/bash
# backup_config.sh

BACKUP_DIR="/backups/config"
DATE=$(date +%Y%m%d_%H%M%S)
CONFIG_DIR="${BACKUP_DIR}/${DATE}"

mkdir -p $CONFIG_DIR

# Backup Kubernetes configurations
kubectl get all -n isp-checker -o yaml > "${CONFIG_DIR}/k8s-resources.yaml"
kubectl get configmaps -n isp-checker -o yaml > "${CONFIG_DIR}/configmaps.yaml"
kubectl get secrets -n isp-checker -o yaml > "${CONFIG_DIR}/secrets.yaml"

# Backup application configurations
kubectl get configmap backend-config -n isp-checker -o jsonpath='{.data.config\.yaml}' > "${CONFIG_DIR}/backend-config.yaml"

# Backup Helm values
helm get values isp-checker -n isp-checker > "${CONFIG_DIR}/helm-values.yaml"

# Create archive
tar -czf "${BACKUP_DIR}/config_${DATE}.tar.gz" -C $BACKUP_DIR $DATE

# Upload to cloud storage
aws s3 cp "${BACKUP_DIR}/config_${DATE}.tar.gz" \
  s3://isp-checker-backups/config/

echo "Configuration backup completed: config_${DATE}.tar.gz"
```

### 5.2. Recovery Objectives

| Metric | Target | Justification |
|--------|--------|---------------|
| **RTO (Recovery Time Objective)** | 4 hours | Maximum acceptable downtime for critical monitoring |
| **RPO (Recovery Point Objective)** | 1 hour | Maximum acceptable data loss for monitoring data |
| **Backup Retention** | 30 days daily, 12 weeks weekly, 12 months monthly | Balance between storage costs and recovery options |
| **Recovery Testing** | Monthly | Ensure backup integrity and recovery procedures |

### 5.3. Disaster Recovery Procedures

#### 5.3.1. Complete System Recovery

**Step 1: Infrastructure Preparation**
```bash
# Create new namespace
kubectl create namespace isp-checker-dr

# Install required operators
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install postgres bitnami/postgresql \
  --namespace isp-checker-dr \
  --set auth.postgresPassword="secure_password" \
  --set auth.database="isp_health_checker"
```

**Step 2: Database Restoration**
```bash
#!/bin/bash
# restore_database.sh

BACKUP_FILE=$1
NAMESPACE="isp-checker-dr"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# Download backup from cloud storage
aws s3 cp "s3://isp-checker-backups/database/${BACKUP_FILE}" ./

# Extract backup if compressed
if [[ $BACKUP_FILE == *.gz ]]; then
    gunzip $BACKUP_FILE
    BACKUP_FILE=${BACKUP_FILE%.gz}
fi

# Restore database
kubectl exec -n $NAMESPACE deployment/postgres -- \
  psql -U postgres -d isp_health_checker < $BACKUP_FILE

echo "Database restoration completed"
```

**Step 3: Application Deployment**
```bash
# Deploy application using backed up configuration
helm install isp-checker ./charts/isp-checker \
  --namespace isp-checker-dr \
  --values config_${DATE}/helm-values.yaml \
  --set database.url="postgresql://postgres:secure_password@postgres:5432/isp_health_checker"

# Verify deployment
kubectl rollout status deployment/backend -n isp-checker-dr
kubectl rollout status deployment/ui -n isp-checker-dr
```

#### 5.3.2. Partial Recovery Scenarios

**Database Only Recovery**:
```bash
# Scale down backend to prevent data corruption
kubectl scale deployment backend --replicas=0 -n isp-checker

# Restore database
./restore_database.sh isp_health_checker_20251203_120000.sql

# Scale up backend
kubectl scale deployment backend --replicas=2 -n isp-checker
```

**Configuration Recovery**:
```bash
# Restore from configuration backup
kubectl apply -f config_${DATE}/k8s-resources.yaml
kubectl apply -f config_${DATE}/configmaps.yaml

# Restore secrets (manual intervention required for sensitive data)
kubectl apply -f config_${DATE}/secrets.yaml
```

### 5.4. Recovery Testing

#### 5.4.1. Monthly Recovery Drill

```bash
#!/bin/bash
# monthly_recovery_test.sh

TEST_NAMESPACE="isp-checker-test"
BACKUP_DATE=$(date -d "1 day ago" +%Y%m%d)

echo "Starting monthly recovery drill..."

# Create test namespace
kubectl create namespace $TEST_NAMESPACE

# Deploy test infrastructure
helm install postgres-test bitnami/postgresql \
  --namespace $TEST_NAMESPACE \
  --set auth.postgresPassword="test_password" \
  --set auth.database="isp_health_checker"

# Restore yesterday's backup
./restore_database.sh "isp_health_checker_${BACKUP_DATE}_120000.sql"

# Deploy application
helm install isp-checker-test ./charts/isp-checker \
  --namespace $TEST_NAMESPACE \
  --set database.url="postgresql://postgres:test_password@postgres-test:5432/isp_health_checker"

# Run health checks
sleep 60
HEALTH_CHECK=$(kubectl exec -n $TEST_NAMESPACE deployment/backend -- \
  curl -s http://localhost:8000/health)

if [[ $HEALTH_CHECK == *"healthy"* ]]; then
    echo "Recovery test PASSED"
else
    echo "Recovery test FAILED"
    exit 1
fi

# Cleanup test environment
kubectl delete namespace $TEST_NAMESPACE

echo "Monthly recovery drill completed successfully"
```

#### 5.4.2. Backup Verification

```bash
#!/bin/bash
# verify_backups.sh

# Verify database backup integrity
LATEST_BACKUP=$(aws s3 ls s3://isp-checker-backups/database/ | \
  sort | tail -n 1 | awk '{print $4}')

# Download and test backup
aws s3 cp "s3://isp-checker-backups/database/${LATEST_BACKUP}" ./

# Test backup file integrity
if [[ $LATEST_BACKUP == *.gz ]]; then
    if gzip -t $LATEST_BACKUP; then
        echo "Database backup integrity verified: ${LATEST_BACKUP}"
    else
        echo "Database backup CORRUPTED: ${LATEST_BACKUP}"
        exit 1
    fi
fi

# Verify configuration backup completeness
LATEST_CONFIG=$(aws s3 ls s3://isp-checker-backups/config/ | \
  sort | tail -n 1 | awk '{print $4}')

aws s3 cp "s3://isp-checker-backups/config/${LATEST_CONFIG}" ./
tar -tzf $LATEST_CONFIG | wc -l

echo "Configuration backup verified: ${LATEST_CONFIG}"
```

---

## 6. Maintenance Procedures

This section outlines routine maintenance tasks required to keep the ISP Health Checker system running optimally.

### 6.1. Database Maintenance

#### 6.1.1. Routine Database Tasks

**Weekly Vacuum and Analyze**:
```bash
#!/bin/bash
# weekly_db_maintenance.sh

NAMESPACE="isp-checker"

echo "Starting weekly database maintenance..."

# Connect to database and run maintenance
kubectl exec -n $NAMESPACE deployment/db -- psql -U user -d isp_health_checker << EOF
-- Update table statistics
ANALYZE;

-- Reclaim storage from dead tuples
VACUUM VERBOSE;

-- Rebuild indexes if needed
REINDEX DATABASE isp_health_checker;

-- Check for bloat
SELECT schemaname, tablename, 
       ROUND(CASE WHEN otta=0 THEN 0.0 ELSE sml.relpages/otta::numeric END,1) AS bloat,
       CASE WHEN relpages < otta THEN 0 ELSE relpages::bigint - otta END AS wasted_pages
FROM (
  SELECT 
    s.schemaname, s.tablename, 
    cc.reltuples, cc.relpages, 
    CEIL((cc.reltuples*((datahdr+ma-
      (CASE WHEN datahdr%ma=0 THEN ma ELSE datahdr%ma END))+nullhdr2+4))/(bs-20::float)) AS otta
  FROM (
    SELECT 
      ma,bs,schemaname,tablename,
      (datawidth+(hdr+ma-(CASE WHEN hdr%ma=0 THEN ma ELSE hdr%ma END)))::numeric AS datahdr,
      (maxfracsum*(nullhdr+ma-(CASE WHEN nullhdr%ma=0 THEN ma ELSE nullhdr%ma END))) AS nullhdr2
    FROM (
      SELECT 
        schemaname, tablename, hdr, ma, bs,
        SUM((1-null_frac)*avg_width) AS datawidth,
        MAX(null_frac) AS maxfracsum,
        hdr+(
          SELECT 1+COUNT(*)*(8-CASE WHEN avg_width<=248 THEN 1 ELSE 2 END)
          FROM pg_stats s2 
          WHERE s2.schemaname=s.schemaname AND s2.tablename=s.tablename AND null_frac<>0
        ) AS nullhdr
      FROM pg_stats s, (
        SELECT 
          (SELECT current_setting('block_size')::numeric) AS bs,
          CASE WHEN substring(v,12,3) IN ('8.0','8.1','8.2') THEN 27 ELSE 23 END AS hdr,
          CASE WHEN v ~ 'mingw32' THEN 8 ELSE 4 END AS ma
        FROM (SELECT version() AS v) AS foo
      ) AS constants
      GROUP BY 1,2,3,4,5
    ) AS foo
  ) AS rs
  JOIN pg_class cc ON cc.relname = rs.tablename
  JOIN pg_namespace nn ON cc.relnamespace = nn.oid AND nn.nspname = rs.schemaname AND nn.nspname <> 'information_schema'
) AS sml
WHERE sml.schemaname='public'
ORDER BY wasted_pages DESC;

EOF

echo "Database maintenance completed"
```

**Monthly Index Optimization**:
```sql
-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- Rebuild unused or fragmented indexes
REINDEX INDEX CONCURRENTLY index_name;
```

#### 6.1.2. Data Retention Management

```bash
#!/bin/bash
# data_retention.sh

NAMESPACE="isp-checker"
RETENTION_DAYS=30

echo "Cleaning up data older than ${RETENTION_DAYS} days..."

# Connect to database and clean up old data
kubectl exec -n $NAMESPACE deployment/db -- psql -U user -d isp_health_checker << EOF
-- Delete old runs based on retention policy
DELETE FROM runs 
WHERE timestamp < NOW() - INTERVAL '${RETENTION_DAYS} days';

-- Record cleanup statistics
DO \$\$
DECLARE
    deleted_count INTEGER;
BEGIN
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RAISE NOTICE 'Deleted % old records', deleted_count;
    
    -- Log the cleanup
    INSERT INTO maintenance_log (action, record_count, timestamp)
    VALUES ('data_retention', deleted_count, NOW());
END \$\$;

-- Optimize table after deletion
VACUUM ANALYZE runs;

EOF

echo "Data retention cleanup completed"
```

### 6.2. Log Management

#### 6.2.1. Log Rotation

```yaml
# Configure log rotation in deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  template:
    spec:
      containers:
      - name: backend
        env:
        - name: LOG_LEVEL
          value: "info"
        - name: LOG_MAX_SIZE
          value: "100M"
        - name: LOG_MAX_FILES
          value: "5"
        volumeMounts:
        - name: logs
          mountPath: /app/logs
      volumes:
      - name: logs
        emptyDir: {}
```

#### 6.2.2. Log Aggregation Setup

```yaml
# Fluentd configuration for log aggregation
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluentd-config
data:
  fluent.conf: |
    <source>
      @type tail
      path /var/log/containers/*isp-checker*.log
      pos_file /var/log/fluentd-containers.log.pos
      tag kubernetes.*
      format json
      time_format %Y-%m-%dT%H:%M:%S.%NZ
    </source>
    
    <filter kubernetes.**>
      @type kubernetes_metadata
    </filter>
    
    <match kubernetes.**>
      @type elasticsearch
      host elasticsearch.logging.svc.cluster.local
      port 9200
      index_name isp-checker-logs
      type_name _doc
    </match>
```

### 6.3. Software Updates

#### 6.3.1. Rolling Update Procedure

```bash
#!/bin/bash
# rolling_update.sh

COMPONENT=$1
NEW_VERSION=$2
NAMESPACE="isp-checker"

if [ -z "$COMPONENT" ] || [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <component> <new_version>"
    exit 1
fi

echo "Starting rolling update for ${COMPONENT} to version ${NEW_VERSION}..."

# Update image version
kubectl set image deployment/${COMPONENT} \
  ${COMPONENT}=your-repo/isp-checker-${COMPONENT}:${NEW_VERSION} \
  -n $NAMESPACE

# Monitor rollout progress
kubectl rollout status deployment/${COMPONENT} -n $NAMESPACE --timeout=300s

# Verify deployment health
kubectl get pods -n $NAMESPACE -l app=${COMPONENT}

# Run smoke tests
./smoke_tests.sh $COMPONENT

echo "Rolling update completed successfully"
```

#### 6.3.2. Database Schema Updates

```bash
#!/bin/bash
# database_migration.sh

MIGRATION_FILE=$1
NAMESPACE="isp-checker"

if [ -z "$MIGRATION_FILE" ]; then
    echo "Usage: $0 <migration_file>"
    exit 1
fi

echo "Applying database migration: ${MIGRATION_FILE}"

# Create backup before migration
./backup_database.sh

# Apply migration
kubectl exec -n $NAMESPACE deployment/db -- \
  psql -U user -d isp_health_checker -f /migrations/${MIGRATION_FILE}

# Verify migration success
kubectl exec -n $NAMESPACE deployment/db -- \
  psql -U user -d isp_health_checker -c "SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1;"

echo "Database migration completed"
```

### 6.4. Security Maintenance

#### 6.4.1. Certificate Rotation

```bash
#!/bin/bash
# certificate_rotation.sh

NAMESPACE="isp-checker"
CERT_NAME="isp-checker-tls"

echo "Starting certificate rotation..."

# Generate new certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=isp-checker.local"

# Create Kubernetes secret
kubectl create secret tls $CERT_NAME \
  --cert=tls.crt \
  --key=tls.key \
  -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Restart services to pick up new certificate
kubectl rollout restart deployment/backend -n $NAMESPACE
kubectl rollout restart deployment/ui -n $NAMESPACE

echo "Certificate rotation completed"
```

#### 6.4.2. API Key Rotation

```bash
#!/bin/bash
# api_key_rotation.sh

NAMESPACE="isp-checker"
KEY_NAME="monitoring-key"

echo "Rotating API key: ${KEY_NAME}"

# Generate new API key
NEW_KEY=$(openssl rand -hex 32)

# Update secret
kubectl create secret generic api-keys \
  --from-literal=${KEY_NAME}=${NEW_KEY} \
  -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Restart backend to pick up new key
kubectl rollout restart deployment/backend -n $NAMESPACE

echo "API key rotation completed"
echo "New API key: ${NEW_KEY}"
```

### 6.5. Performance Monitoring

#### 6.5.1. Weekly Performance Review

```bash
#!/bin/bash
# weekly_performance_review.sh

DATE=$(date -d "1 week ago" +%Y-%m-%d)
END_DATE=$(date +%Y-%m-%d)

echo "Weekly Performance Review: ${DATE} to ${END_DATE}"

# Generate performance report
curl -G "http://prometheus:9090/api/v1/query_range" \
  -d "query=avg_over_time(isp_checker_score[1h])" \
  -d "start=${DATE}T00:00:00Z" \
  -d "end=${END_DATE}T23:59:59Z" \
  -d "step=1h" > weekly_scores.json

# Analyze performance trends
python3 analyze_performance.py weekly_scores.json

# Check for resource utilization trends
kubectl top pods -n isp-checker --no-headers | \
  awk '{print $1, $2, $3}' > weekly_resource_usage.txt

echo "Performance review completed"
```

#### 6.5.2. Capacity Planning

```bash
#!/bin/bash
# capacity_planning.sh

# Analyze growth trends
kubectl exec -n isp-checker deployment/db -- \
  psql -U user -d isp_health_checker << EOF
-- Analyze data growth
SELECT 
    DATE_TRUNC('month', timestamp) AS month,
    COUNT(*) AS run_count,
    AVG(pg_size_pretty(pg_relation_size('runs'))) AS avg_table_size
FROM runs 
WHERE timestamp >= NOW() - INTERVAL '12 months'
GROUP BY month
ORDER BY month DESC;

-- Predict future growth
SELECT 
    DATE_TRUNC('month', timestamp) AS month,
    COUNT(*) AS runs,
    LAG(COUNT(*)) OVER (ORDER BY DATE_TRUNC('month', timestamp)) AS previous_month,
    (COUNT(*)::FLOAT / LAG(COUNT(*)) OVER (ORDER BY DATE_TRUNC('month', timestamp)) - 1) * 100 AS growth_percentage
FROM runs 
WHERE timestamp >= NOW() - INTERVAL '12 months'
GROUP BY month
ORDER BY month DESC;

EOF

echo "Capacity planning analysis completed"
```

---

## Conclusion

This operational runbook provides comprehensive guidance for managing the ISP Health Checker system in production. Regular review and updates to this document are essential to ensure it remains current with system changes and operational experience.

For questions or suggestions regarding this runbook, please contact the operations team or create an issue in the project repository.