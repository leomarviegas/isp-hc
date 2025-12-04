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
	StatusFail = "fail"
	StatusNA   = "na"
)
