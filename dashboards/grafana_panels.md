# Grafana Dashboard Layout & Queries

This document specifies the layout and Prometheus queries for the ISP Health Checker Grafana dashboard.

**Dashboard Title:** ISP Health Monitoring
**Dashboard Variables:**
*   `$target`: A query variable for filtering by target host.
    *   Query: `label_values(isp_checker_score, target)`

---

## Panel Layout

| Row 1: Key Performance Indicators (KPIs)                                                                                             |
| ------------------------------------------------------------------------------------------------------------------------------------ |
| **Panel 1.1: Overall Health Score (Gauge)** <br> Shows the most recent health score for the selected target.                              |
| **Panel 1.2: Average Packet Loss (Stat)** <br> Shows the average packet loss from the most recent ping probe.                            |
| **Panel 1.3: Average Latency (Stat)** <br> Shows the average round-trip latency from the most recent ping probe.                          |
| **Panel 1.4: DNS Resolution Time (Stat)** <br> Shows the average DNS query time from the most recent run.                                |
|                                                                                                                                      |
| **Row 2: Time Series Analysis**                                                                                                      |
| **Panel 2.1: Health Score Over Time (Time series)** <br> Plots the health score, helping to visualize trends and outages.               |
| **Panel 2.2: Latency & Packet Loss Over Time (Time series)** <br> Correlates latency (line) and packet loss (bar) on a single graph.    |
|                                                                                                                                      |
| **Row 3: Probe Status & Diagnosis**                                                                                                  |
| **Panel 3.1: Probe Status History (State timeline)** <br> Shows the UP/DOWN status history for each individual probe over time.       |
| **Panel 3.2: Diagnosis Breakdown (Pie chart)** <br> Shows the distribution of diagnoses (e.g., DNS, Upstream, Local) over the selected time range. |
|                                                                                                                                      |
| **Row 4: Run Log**                                                                                                                   |
| **Panel 4.1: Recent Runs (Table)** <br> A table view of the last 100 runs with key details like score, summary, and timestamp.         |

---

## Prometheus Queries

### Panel 1.1: Overall Health Score (Gauge)

*   **Title:** Health Score
*   **Type:** Gauge
*   **Query:** `isp_checker_score{target=~"$target"}`
*   **Options:** Show last value. Set thresholds: 80=Red, 40=Yellow.

### Panel 1.2: Average Packet Loss (Stat)

*   **Title:** Packet Loss
*   **Type:** Stat
*   **Query:** `isp_checker_probe_details{target=~"$target", probe="ping", key="loss_percent"}`
*   **Unit:** Percent (0.0-1.0) -> Standard field option 'Percent (0.0-100.0)' in Grafana
*   **Options:** Show last value.

### Panel 1.3: Average Latency (Stat)

*   **Title:** Latency
*   **Type:** Stat
*   **Query:** `isp_checker_probe_details{target=~"$target", probe="ping", key="latency_avg_ms"}`
*   **Unit:** Milliseconds (ms)
*   **Options:** Show last value.

### Panel 1.4: DNS Resolution Time (Stat)

*   **Title:** DNS Query Time
*   **Type:** Stat
*   **Query:** `isp_checker_probe_details{target=~"$target", probe="dns", key="resolution_time_ms"}`
*   **Unit:** Milliseconds (ms)
*   **Options:** Show last value.

### Panel 2.1: Health Score Over Time (Time series)

*   **Title:** Health Score Over Time
*   **Type:** Time series
*   **Query:** `avg_over_time(isp_checker_score{target=~"$target"}[$__interval])`
*   **Legend:** `{{target}}`

### Panel 2.2: Latency & Packet Loss Over Time (Time series)

*   **Title:** Latency & Packet Loss
*   **Type:** Time series
*   **Queries:**
    1.  `avg_over_time(isp_checker_probe_details{target=~"$target", probe="ping", key="latency_avg_ms"}[$__interval])` (Y-Axis 1, Unit: ms)
    2.  `avg_over_time(isp_checker_probe_details{target=~"$target", probe="ping", key="loss_percent"}[$__interval])` (Y-Axis 2, Unit: %)
*   **Options:** Set series override for packet loss to be rendered as bars.

### Panel 3.1: Probe Status History (State timeline)

*   **Title:** Probe Status
*   **Type:** State timeline
*   **Query:** `isp_checker_probe_status{target=~"$target"}`
*   **Options:**
    *   Map values: `0` to `OK` (Green), `1` to `WARN` (Yellow), `2` to `CRIT` (Red), `3` to `NA` (Blue).
    *   Legend: `{{probe}}`

### Panel 3.2: Diagnosis Breakdown (Pie chart)

*   **Title:** Diagnosis Distribution
*   **Type:** Pie chart
*   **Query:** `sum(count_over_time(isp_checker_diagnosis_total{target=~"$target"}[$__range])) by (component)`
*   **Legend:** `{{component}}`

### Panel 4.1: Recent Runs (Table)

*   **Title:** Run Log
*   **Type:** Table
*   **Queries:**
    *   A: `isp_checker_score{target=~"$target"}`
    *   B: `isp_checker_run_summary{target=~"$target"}`
*   **Options:** Use "Outer join" transform on the `timestamp` field. Add columns for `target`, `score`, and `summary`.
