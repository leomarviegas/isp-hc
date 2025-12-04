package probes

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RunPing executes a system ping. If ping binary is missing, returns StatusNA.
func RunPing(ctx context.Context, target string) Result {
	binary := "ping"
	args := []string{"-c", "3", "-w", "4", target}
	if runtime.GOOS == "windows" {
		args = []string{"-n", "3", target}
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return Result{Name: "ping", Status: StatusNA, Error: "ping binary not available"}
	}

	cmd := exec.CommandContext(ctx, path, args...)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Result{Name: "ping", Status: StatusFail, Error: "ping timeout"}
	}
	if err != nil {
		return Result{Name: "ping", Status: StatusFail, Error: strings.TrimSpace(string(out))}
	}

	return Result{Name: "ping", Status: StatusOK, LatencyMs: float64(elapsed.Milliseconds()), Details: map[string]interface{}{"raw": string(out)}}
}

// ParseLatencyMs is a helper to pull latency from common ping output.
func ParseLatencyMs(output string) float64 {
	// Best-effort parse of "time=XX ms" token.
	parts := strings.Fields(output)
	for _, p := range parts {
		if strings.HasPrefix(p, "time=") {
			val := strings.TrimPrefix(p, "time=")
			val = strings.TrimSuffix(val, "ms")
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		}
	}
	return 0
}
