# UX Wireframes & Components

This document outlines the wireframes for the ISP Health Checker's CLI and Web UI, followed by a list of React components using Tailwind CSS.

## 1. CLI Output Sample

The CLI provides a compact, human-readable summary. Verbose output is handled by the JSON file.

```
$ isp-checker run --target 8.8.8.8 --summary

[ ISP Health Checker ]
► Run ID:      1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d
► Timestamp:   2025-12-03T10:00:00Z
► Target:      8.8.8.8 (google-dns)

[ Health Score: 15/100 (OK) ]

[ Diagnosis ]
✔ DNS Resolution: OK
✔ Upstream Connectivity: OK
⚠ Peering/Transit: Possible intermittent latency spikes detected.

[ Probe Summary ]
- ping:        OK (avg: 12.5ms, loss: 0.0%)
- traceroute:  WARN (hop 5 latency > 150ms)
- dns:         OK (resolved in 8ms)
- http:        OK (200 OK in 120ms)
- bgp:         NA
- speedtest:   NA

Full report saved to: ./results/8.8.8.8-2025-12-03T100000.json
```

---

## 2. Web UI: Main Dashboard

This screen provides an at-a-glance view of the most recent health checks.

```
+---------------------------------------------------------------------------------------------------+
| [ISP Health Checker]                                                        [Search by Target...] |
|                                                                                                   |
|  / Recent Runs --------------------------------------------------------------------------------- / |
|                                                                                                   |
|  +------------------+------------------+---------------------+----------------+-----------------+ |
|  | TARGET           | STATUS           | SCORE (0-100)       | DIAGNOSIS      | TIMESTAMP (UTC) | |
|  +------------------+------------------+---------------------+----------------+-----------------+ |
|  | 8.8.8.8          | [OK]             | 15                  | Peering/Tran.. | 2m ago          | |
|  | 1.1.1.1          | [OK]             | 8                   | OK             | 12m ago         | |
|  | my-api.prod.net  | [CRIT]           | 85                  | Upstream Outage| 1h ago          | |
|  | office.mycorp.com| [WARN]           | 42                  | DNS Issue      | 3h ago          | |
|  | ...              | ...              | ...                 | ...            | ...             | |
|  +------------------+------------------+---------------------+----------------+-----------------+ |
|                                                                                                   |
|  [< Prev] [1] [2] [3] [Next >]                                                                     |
|                                                                                                   |
+---------------------------------------------------------------------------------------------------+
```

---

## 3. Web UI: Detailed Run Drilldown

This view shows the complete details for a single diagnostic run.

```
+---------------------------------------------------------------------------------------------------+
| [ISP Health Checker] > Run 1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d                                     |
|                                                                                                   |
|  / Summary ------------------------------------------------------------------------------------- / |
|  |                                                                                                 |
|  |  Target: 8.8.8.8                                        Score: [15] OK                          |
|  |  Timestamp: 2025-12-03T10:00:00Z                        [Download JSON] [Run Again]             |
|  |                                                                                                 |
|  |  Diagnosis: Possible intermittent latency spikes detected in peering/transit.                   |
|  |  Suggested Action: Monitor hop 5 (asn-xxxx.net) and escalate to ISP if persistent.              |
|  |                                                                                                 |
|  / Probe Details ------------------------------------------------------------------------------- / |
|  |                                                                                                 |
|  |  ▼ PING (OK)                                                                                    |
|  |    - Status: OK, Packets: 50/50, Loss: 0.0%, Latency (min/avg/max): 10/12.5/25 ms                |
|  |                                                                                                 |
|  |  ▼ TRACEROUTE (WARN)                                                                            |
|  |    - Status: WARN, Hops: 10                                                                     |
|  |    - Path: 1. local (1ms) > 2. isp-gw (5ms) > ... > 5. asn-xxxx.net (180ms) > ... > 10. 8.8.8.8   |
|  |                                                                                                 |
|  |  ▶ DNS (OK)                                                                                     |
|  |  ▶ HTTP (OK)                                                                                    |
|  |                                                                                                 |
+---------------------------------------------------------------------------------------------------+
```

---

## 4. React + Tailwind Component List

| Component          | Description                                                    | Props                             | State (useState)                |
| ------------------ | -------------------------------------------------------------- | --------------------------------- | ------------------------------- |
| **`RunsTable`**      | Displays a table of recent runs (Main Dashboard).              | `runs: Run[]`                     | `sorting`, `pagination`         |
| **`RunRow`**         | A single row in the `RunsTable`.                               | `run: Run`                        | `isHovered`                     |
| **`ScoreBadge`**     | A color-coded badge (green/yellow/red) for the health score.   | `score: number`                   | -                               |
| **`StatusPill`**     | A pill-shaped indicator for status (OK, WARN, CRIT).           | `status: string`                  | -                               |
| **`SearchBar`**      | An input field for searching/filtering runs by target.         | `onSearch: (query) => void`       | `query`                         |
| **`RunDetailView`**  | The main view for the drilldown page.                          | `runId: string`                   | `runData`, `isLoading`, `error` |
| **`SummaryCard`**    | Displays the top-level summary and diagnosis for a run.        | `run: Run`                        | -                               |
| **`ProbeAccordion`** | An accordion component to show/hide details for each probe.    | `probes: Probe[]`                 | `openProbe`                     |
| **`ProbeDetail`**    | Renders the specific key/value details for a single probe.     | `probe: Probe`                    | -                               |
| **`Button`**         | A generic, styled button.                                      | `onClick`, `variant`, `children`  | `isDisabled`                    |
| **`PageLayout`**     | Main container with header, footer, and content area.          | `children`                        | -                               |
