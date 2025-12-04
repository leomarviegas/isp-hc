package probes

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// RunTraceroute executes traceroute/tracepath.
func RunTraceroute(ctx context.Context, target string) Result {
	candidateBins := []string{"traceroute", "tracepath"}
	var bin string
	for _, b := range candidateBins {
		if path, err := exec.LookPath(b); err == nil {
			bin = path
			break
		}
	}
	if bin == "" {
		return Result{Name: "traceroute", Status: StatusNA, Error: "traceroute/tracepath not available"}
	}

	args := []string{target}
	cmd := exec.CommandContext(ctx, bin, args...)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Result{Name: "traceroute", Status: StatusFail, Error: "traceroute timeout"}
	}
	if err != nil {
		return Result{Name: "traceroute", Status: StatusFail, Error: strings.TrimSpace(string(out))}
	}

	return Result{Name: "traceroute", Status: StatusOK, LatencyMs: float64(time.Since(start).Milliseconds()), Details: map[string]interface{}{"raw": string(out)}}
}
