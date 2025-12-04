package probes

// Result captures the outcome of a probe execution.
type Result struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	LatencyMs float64                `json:"latency_ms,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

const (
	StatusOK   = "ok"
	StatusWarn = "warn"
	StatusFail = "fail"
	StatusNA   = "na"
)

// ProbeCategory groups probes by type.
type ProbeCategory string

const (
	CategoryConnectivity ProbeCategory = "connectivity"
	CategoryPacketHealth ProbeCategory = "packet_health"
	CategoryPerformance  ProbeCategory = "performance"
)

// ProbeInfo provides metadata about a probe.
type ProbeInfo struct {
	Name        string        `json:"name"`
	Category    ProbeCategory `json:"category"`
	Description string        `json:"description"`
	RequiresRoot bool         `json:"requires_root"`
}

// AvailableProbes lists all probe types and their metadata.
var AvailableProbes = []ProbeInfo{
	{Name: "ping", Category: CategoryConnectivity, Description: "ICMP ping for basic connectivity and latency"},
	{Name: "dns", Category: CategoryConnectivity, Description: "DNS resolution test"},
	{Name: "traceroute", Category: CategoryConnectivity, Description: "Network path analysis"},
	{Name: "interface_stats", Category: CategoryPacketHealth, Description: "Network interface error counters (CRC, frame, drops)"},
	{Name: "tcp_stats", Category: CategoryPacketHealth, Description: "TCP protocol statistics (retransmits, reordering)"},
	{Name: "socket_stats", Category: CategoryPerformance, Description: "Per-socket TCP metrics (RTT, congestion)"},
	{Name: "packet_capture", Category: CategoryPacketHealth, Description: "Deep packet inspection", RequiresRoot: true},
}
