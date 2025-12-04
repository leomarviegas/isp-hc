# Security Architecture and Threat Model: ISP Health Checker

This document outlines the security architecture, threat landscape, and mitigations for the ISP Health Checker system. It is intended to guide secure development, deployment, and operational practices.

---

## 1. Executive Summary

The ISP Health Checker is a network diagnostics tool composed of a CLI probe runner, a backend API, a PostgreSQL database, and a web UI. Given its ability to probe network targets and store potentially sensitive network topology data, security is a critical concern. This model uses the STRIDE framework to analyze threats, assesses the system against the OWASP Top 10, and provides comprehensive controls for data protection, compliance, and incident response.

The primary attack surface consists of:
- The Backend API, which ingests and serves diagnostic data.
- The CLI tool, which can be used to execute network probes.
- The Web UI, which displays sensitive historical data.
- The underlying Kubernetes infrastructure.

---

## 2. STRIDE Threat Model

This section analyzes the system using the STRIDE (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege) methodology.

### 2.1. Spoofing

*   **Threat:** An attacker could spoof probe packets to distort network measurements or use the tool as a reflection vector in a DDoS attack.
    *   **Component:** CLI Probe Runner.
    *   **Mitigation:**
        *   **M1.1:** Prioritize TCP-based probes (e.g., HTTP checks) over ICMP where possible, as TCP handshakes are less susceptible to IP spoofing.
        *   **M1.2:** For probes requiring raw sockets (like `ping`), run the CLI with elevated capabilities (`cap_net_raw`) instead of as root, limiting the scope of compromise.
        *   **M1.3:** The backend must not trust IP addresses from submitted reports. All rate limiting and abuse detection should be based on authenticated API keys, not source IPs.

*   **Threat:** An attacker attempts to gain unauthorized access to the backend API by using a stolen or forged API key.
    *   **Component:** Backend API.
    *   **Mitigation:**
        *   **M1.4:** API keys must be high-entropy (e.g., 256-bit base64url encoded random strings) and stored securely in the database using at least SHA-256 hashing.
        *   **M1.5:** Implement a strict API key rotation policy and provide a mechanism for users to revoke compromised keys via the API or UI.

### 2.2. Tampering

*   **Threat:** A Man-in-the-Middle (MitM) attacker alters probe results in transit from the CLI to the backend API, leading to incorrect diagnostics.
    *   **Component:** CLI, Backend API.
    *   **Mitigation:**
        *   **M2.1:** Enforce TLS 1.2 or higher for all API communication. In Kubernetes, use `cert-manager` to automate provisioning and renewal of certificates from a trusted CA (e.g., Let's Encrypt).
        *   **M2.2:** Implement request payload integrity checks. The client should include a `X-Signature` header containing a HMAC-SHA256 hash of the request body, using the API key as the secret. The backend will verify this signature before processing the request.

*   **Threat:** An attacker with database access modifies historical run data, undermining the integrity of monitoring records.
    *   **Component:** Database.
    *   **Mitigation:**
        *   **M2.3:** The database connection string and credentials must be stored as Kubernetes Secrets, accessible only to the backend service account.
        *   **M2.4:** Enable PostgreSQL's `row-level security` and ensure the backend's database user has only `INSERT`, `SELECT`, `UPDATE` (on specific fields), and `DELETE` privileges, with no `DROP` or `ALTER` rights.
        *   **M2.5:** Configure continuous, point-in-time recovery backups for PostgreSQL to a separate, secure location to enable restoration after a tampering incident.

### 2.3. Repudiation

*   **Threat:** A user performs a malicious action (e.g., deleting all runs for a specific target) and denies having done so.
    *   **Component:** Backend API.
    *   **Mitigation:**
        *   **M3.1:** Implement an immutable, append-only audit log for all state-changing API requests (`POST`, `PUT`, `DELETE`). Each log entry must include: timestamp, API key identifier, endpoint, request body hash, and source IP.
        *   **M3.2:** Ship audit logs to a tamper-evident, external logging system (e.g., a centralized SIEM like Elasticsearch or a cloud provider's logging service) with write-once storage capabilities.

### 2.4. Information Disclosure

*   **Threat:** An unauthorized user accesses sensitive diagnostic reports, which could reveal internal network topology, IP addresses of critical assets, or service vulnerabilities.
    *   **Component:** Backend API, Database, Web UI.
    *   **Mitigation:**
        *   **M4.1:** All API endpoints must be protected by API key authentication. The Web UI must enforce session-based authentication with secure, HTTP-only cookies.
        *   **M4.2:** Enforce the principle of least privilege in Kubernetes using NetworkPolicies. For example, the UI pod should only be able to connect to the API pod on port 443, and the API pod should only connect to the database pod on port 5432.
        *   **M4.3:** Classify data and enforce access controls. The `raw` field in reports, which may contain sensitive command output, should have a more restrictive access policy, viewable only by high-privilege users.
        *   **M4.4:** Configure the backend to return generic, non-descriptive error messages (e.g., `{"error": "Invalid request"}`) to the client. Detailed stack traces and error codes should only be logged server-side.

### 2.5. Denial of Service (DoS)

*   **Threat:** An attacker overwhelms the backend API with a high volume of requests, exhausting worker resources and saturating the database.
    *   **Component:** Backend API.
    *   **Mitigation:**
        *   **M5.1:** Deploy a cloud-native API gateway (e.g., Kong, Ambassador) or use an Ingress controller with rate-limiting capabilities (e.g., NGINX Ingress with annotations). Configure strict per-API-key rate limits (e.g., 10 requests/minute).
        *   **M5.2:** Implement request body size limits (e.g., `nginx.ingress.kubernetes.io/proxy-body-size: "1m"`) to prevent resource exhaustion from large payloads.
        *   **M5.3:** Configure Kubernetes Horizontal Pod Autoscalers (HPA) for the backend API to automatically scale the number of pods based on CPU/memory utilization, providing resilience to traffic spikes.

*   **Threat:** An attacker abuses the CLI to launch a DDoS attack against a third-party target by specifying it as the probe target.
    *   **Component:** CLI, Backend Workers.
    *   **Mitigation:**
        *   **M5.4:** Implement a configurable allowlist/denylist for probe targets in the backend configuration. Requests to probe denylisted targets should be rejected immediately.
        *   **M5.5:** Enforce strict limits on probe frequency and packet rate within the CLI and worker logic to prevent the tool from being a high-rate attack vector.

### 2.6. Elevation of Privilege

*   **Threat:** A vulnerability in a third-party library (e.g., a YAML parser) allows an attacker to execute arbitrary code within the context of the backend API container.
    *   **Component:** Backend API, CLI.
    *   **Mitigation:**
        *   **M6.1:** Run all containers as a non-root user with a high UID/GID (e.g., `65532`). Define a read-only root filesystem and use an ephemeral `emptyDir` for `/tmp`.
        *   **M6.2:** Implement a robust software supply chain security process. Use tools like `Trivy` or `Snyk` in the CI/CD pipeline to scan container images for known vulnerabilities and fail the build if critical/high-severity vulnerabilities are found.
        *   **M6.3:** Employ a Kubernetes security policy like `PodSecurityAdmission` or `OPA/Gatekeeper` to enforce secure pod configurations (e.g., prohibiting `privileged: true`, requiring `readOnlyRootFilesystem`).

*   **Threat:** An attacker compromises a lower-privilege component (e.g., the Web UI) and uses that access to pivot and attack the backend API or database.
    *   **Component:** Web UI, Backend API.
    *   **Mitigation:**
        *   **M6.4:** Assign dedicated Kubernetes ServiceAccounts to each pod (UI, API, DB) with Role-Based Access Control (RBAC) rules that grant only the minimum necessary permissions (e.g., the UI ServiceAccount should have no permission to talk to the database Service).
        *   **M6.5:** Use network segmentation to prevent lateral movement. The UI pod should not be able to directly reach the database pod.

---

## 3. OWASP Top 10 Assessment

This section evaluates the system against the OWASP Top 10 (2021) vulnerabilities.

| OWASP Category | Assessment | Remediation Strategy |
| :--- | :--- | :--- |
| **A01: Broken Access Control** | **Medium Risk.** Relies solely on API keys. If a key is leaked, it provides unfettered access. No role-based access control is defined. | Implement role-based access control (RBAC) for API keys (e.g., `read`, `write`, `admin` roles). Enforce authorization checks on every endpoint. |
| **A02: Cryptographic Failures** | **Low Risk.** TLS is planned for data in transit. Data at rest encryption is dependent on PostgreSQL configuration. | Enforce TLS 1.3. Configure PostgreSQL for transparent data encryption (TDE) or use an encrypted storage layer (e.g., AWS EBS encryption). |
| **A03: Injection** | **Low Risk.** The use of an ORM (SQLAlchemy) and Pydantic models significantly reduces the risk of SQL injection. | Continue using parameterized queries/ORMs. Validate and sanitize all user inputs, even those not destined for the database (e.g., to prevent command injection in probe targets). |
| **A04: Insecure Design** | **Medium Risk.** The system's design to accept arbitrary network targets as input is inherently risky. | Harden the design by implementing strict target allowlisting/denylisting (M5.4). Design the API to be stateless and idempotent where possible. |
| **A05: Security Misconfiguration** | **Medium Risk.** Kubernetes and Docker configurations are complex and prone to misconfiguration (e.g., exposed dashboards, default secrets). | Use Kubernetes admission controllers (e.g., `PodSecurityAdmission`) to enforce secure configurations. Regularly scan cluster configurations for misconfigurations using tools like `Polaris`. |
| **A06: Vulnerable and Outdated Components** | **Medium Risk.** The system relies on numerous open-source libraries (Go, Python, Node.js) that may have vulnerabilities. | Integrate dependency scanning into the CI/CD pipeline. Subscribe to security advisories for all direct and indirect dependencies and update them promptly. |
| **A07: Identification and Authentication Failures**| **Medium Risk.** API keys are long-lived credentials. There is no mechanism for multi-factor authentication or session management. | Implement API key expiration and rotation. For the Web UI, use short-lived session tokens with secure refresh mechanisms. |
| **A08: Software and Data Integrity Failures**| **Low-Medium Risk.** There is no integrity check on submitted data (M2.2). CI/CD pipeline integrity is not defined. | Implement request signing (M2.2). Use a secure CI/CD pipeline with signed commits and signed container images to ensure the integrity of the deployed software. |
| **A09: Security Logging and Monitoring Failures**| **Medium Risk.** The current plan for logging is basic. A lack of centralized monitoring and alerting on security events is a gap. | Implement the comprehensive audit logging (M3.1, M3.2). Ship logs to a SIEM and configure alerts for suspicious activities (e.g., a single API key used from multiple IPs in a short time). |
| **A10: Server-Side Request Forgery (SSRF)**| **High Risk.** The core function of the tool is to make requests to user-specified targets, which is a classic SSRF vector. | This is the highest risk. Mitigations M5.4 and M5.5 are critical. Additionally, run probes in a tightly sandboxed environment with restricted network egress (e.g., using a dedicated CNI plugin or egress gateway). |

---

## 4. Data Classification & Encryption

All data processed by the ISP Health Checker must be classified to apply appropriate security controls.

### 4.1. Data Sensitivity Levels

| Classification | Definition | Examples |
| :--- | :--- | :--- |
| **Public** | Data that is intentionally public and poses no risk if disclosed. | Publicly available documentation, open-source code. |
| **Internal** | Data intended for internal use within an organization. Unauthorized disclosure could cause minor operational issues. | Generic health scores, aggregated metrics, internal IP ranges of the organization running the checker. |
| **Confidential** | Sensitive data that could cause significant harm if disclosed. This includes data regulated by privacy laws. | Specific target IP addresses/hostnames of external services, detailed probe results (`raw` field), network topology information, API keys, database credentials. |

### 4.2. Encryption Requirements

| Data State | Confidentiality Level | Encryption Standard |
| :--- | :--- | :--- |
| **Data in Transit** | Internal, Confidential | **TLS 1.3** with strong cipher suites (e.g., `TLS_AES_256_GCM_SHA384`). |
| **Data at Rest** | Internal, Confidential | **AES-256** encryption. This must be enforced at the storage layer (e.g., encrypted PostgreSQL volumes, encrypted EBS/Azure Disk). |
| **Data in Use (Secrets)** | Confidential | Secrets (API keys, DB passwords) must be stored in Kubernetes Secrets, which are etcd encrypted at rest. For higher security, integrate an external secret manager like HashiCorp Vault. |

---

## 5. Compliance Framework (GDPR & LGPD)

The ISP Health Checker processes IP addresses and hostnames, which can be considered Personal Data under GDPR (EU) and LGPD (Brazil) if they can be linked to an identifiable individual or household.

### 5.1. Lawful Basis for Processing

- **Legitimate Interest:** The primary lawful basis is "legitimate interests" for network monitoring and security diagnostics. This must be balanced against the rights and freedoms of the data subjects.
- **Consent:** If offered as a service to external users, explicit consent must be obtained before collecting and processing their data.

### 5.2. Data Subject Rights Implementation

| Right | Implementation in ISP Health Checker |
| :--- | :--- |
| **Right to be Informed** | Provide a clear privacy policy explaining what data is collected (IPs, hostnames, probe results), for what purpose (network diagnostics), and the retention period. |
| **Right of Access** | The API must provide an endpoint for authenticated users to retrieve all data associated with their API key. |
| **Right to Rectification** | Users should be able to submit corrected data via the API, though this is less relevant for diagnostic data. |
| **Right to Erasure ('Right to be Forgotten')** | Implement an API endpoint (`DELETE /api/v1/data`) that allows a user to request the deletion of all data associated with their API key. This must be a hard delete. |
| **Right to Restrict Processing** | Implement a flag in the `users` table to disable processing of new data for a specific user while retaining existing data. |
| **Right to Data Portability** | The API should allow users to export all their data in a machine-readable format (e.g., JSON). |
| **Right to Object** | Provide a mechanism for users to object to the processing of their data, which should trigger a review process. |

### 5.3. Data Protection by Design and Default

- **Data Minimization:** By default, do not store the `raw` probe output. This should be a configurable option that is explicitly enabled. The default retention policy should be aggressive.
- **Retention Policy:**
    - **Aggregated Metrics (Prometheus):** 90 days.
    - **Detailed Run Reports (JSON):** 30 days.
    - **Raw Probe Outputs:** 7 days (if enabled).
- **Pseudonymization:** Where possible, use internal, non-identifying IDs instead of storing raw IP addresses in frequently accessed tables, storing the mapping separately.

---

## 6. Security Testing

A continuous security testing program must be integrated into the CI/CD pipeline.

### 6.1. Static Application Security Testing (SAST)

- **Tools:** `Semgrep`, `CodeQL`, `gosec` (for Go), `Bandit` (for Python).
- **Process:**
    1. Run SAST tools on every pull request.
    2. Fail the build if any high or critical-severity vulnerabilities are found.
    3. Medium-severity findings should be reviewed and must be addressed before merging to the main branch.
- **Scope:** All source code for the CLI, backend, and UI.

### 6.2. Dynamic Application Security Testing (DAST)

- **Tools:** `OWASP ZAP`, `Burp Suite`.
- **Process:**
    1. Deploy the application to a staging environment that mirrors production.
    2. Run automated DAST scans against the staging API and UI on a nightly basis.
    3. Scan for OWASP Top 10 vulnerabilities, misconfigurations, and exposed secrets.
    4. Create tickets for any identified vulnerabilities and track their resolution.

### 6.3. Software Composition Analysis (SCA)

- **Tools:** `Snyk`, `Trivy`, `OWASP Dependency-Check`.
- **Process:**
    1. Scan all dependencies (Go modules, Python `requirements.txt`, Node.js `package.json`) in the CI/CD pipeline.
    2. Automatically fail the build if a critical or high-severity vulnerability is found in a direct dependency.
    3. For transitive dependencies, create a ticket for remediation based on the severity and exploitability.

### 6.4. Penetration Testing

- **Frequency:** Annually, or after any major architectural change.
- **Scope:**
    - **External Penetration Test:** Focus on the publicly exposed API and UI endpoints.
    - **Internal Penetration Test:** Simulate a compromised internal network to test for lateral movement between components.
- **Process:**
    1. Engage a reputable third-party security firm.
    2. Define clear rules of engagement and scope.
    3. Review the final report and create a remediation plan for all findings.

---

## 7. Incident Response Plan

This plan defines the workflow for handling security incidents.

### 7.1. Incident Response Team

| Role | Responsibility |
| :--- | :--- |
| **Incident Commander** | Overall coordination, communication, and decision-making. |
| **Technical Lead** | Leads the technical investigation, containment, and eradication. |
| **Communications Lead** | Manages internal and external communications. |
| **Legal/Compliance Officer** | Ensures compliance with legal and regulatory obligations. |

### 7.2. Incident Lifecycle

#### 7.2.1. Detection & Analysis

- **Sources of Detection:** SIEM alerts, Prometheus/Grafana alerts (e.g., spike in 5xx errors), user reports, SAST/DAST findings.
- **Triage:** The on-call engineer validates the alert, determines the scope and impact, and estimates the severity (Low, Medium, High, Critical).
- **Escalation:** If severity is High or Critical, the Incident Response Team is immediately paged.

#### 7.2.2. Containment, Eradication & Recovery

- **Containment (Short-Term):**
    - **API Abuse:** Block the offending API key or source IP at the API gateway.
    - **Data Exfiltration:** Revoke all API keys, force a password rotation for all users.
    - **Compromised Host:** Isolate the affected pod/node from the network.
- **Eradication (Long-Term):**
    - Patch the vulnerability that led to the incident.
    - Remove all malicious artifacts (e.g., malware, backdoors).
    - Ensure the root cause is fully addressed.
- **Recovery:**
    - Restore services from clean backups.
    - Monitor closely for any signs of reinfection.
    - Gradually restore normal operations.

#### 7.2.3. Post-Incident Activity

- **Reporting:** Write a detailed post-mortem report covering the timeline, root cause, impact, and lessons learned.
- **Remediation Tracking:** Create a backlog of all security improvement tasks identified during the incident and track them to completion.
- **Improvement:** Update security policies, threat models, and detection rules based on the incident.

---

## 8. Architectural Decisions & Trade-offs

| Decision | Rationale | Trade-off |
| :--- | :--- | :--- |
| **API Key Authentication** | Simple to implement, stateless, and sufficient for machine-to-machine communication. | Lacks the fine-grained control and user context of OAuth. Key compromise is a high-impact event. |
| **PostgreSQL over SQLite** | Offers robust access controls, encryption, and scalability required for a production system. | Increased operational complexity compared to a single-file SQLite database. |
| **Request Signing (M2.2)** | Provides strong integrity guarantees, preventing tampering in transit even if TLS is terminated prematurely. | Adds complexity to the client libraries, which must now compute and attach a signature. |
| **Aggressive Data Retention** | Minimizes the "blast radius" of a data breach and aids compliance with GDPR/LGPD. | Reduces the long-term historical data available for trend analysis and forensics. |
| **Kubernetes Deployment** | Provides a highly scalable, resilient, and feature-rich platform for deploying and managing the application. | Introduces significant complexity in terms of security configuration and management. |