package probes

import (
	"context"
	"net"
	"time"
)

// RunDNS performs a simple A lookup against the system resolver.
func RunDNS(ctx context.Context, target string) Result {
	resolver := &net.Resolver{}
	start := time.Now()
	_, err := resolver.LookupHost(ctx, target)
	if err != nil {
		return Result{Name: "dns", Status: StatusFail, Error: err.Error()}
	}
	return Result{Name: "dns", Status: StatusOK, LatencyMs: float64(time.Since(start).Milliseconds())}
}
