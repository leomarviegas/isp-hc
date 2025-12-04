package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLoadSimulation(t *testing.T) {
	ctx := context.Background()
	simPath := filepath.Join("..", "..", "simulations", "healthy.json")
	res, err := ExecuteRun(ctx, RunOptions{SimulationPath: simPath, Mode: "simulation"})
	if err != nil {
		t.Fatalf("execute run returned error: %v", err)
	}
	if res.RunID == "" {
		t.Fatalf("expected run id to be set")
	}
	if len(res.Probes) == 0 {
		t.Fatalf("expected probes from simulation")
	}
	if res.Score <= 0 {
		t.Fatalf("expected positive score, got %v", res.Score)
	}
}

func TestSelectProbes(t *testing.T) {
	modes := []string{"full", "ping", "dns", "traceroute"}
	for _, m := range modes {
		got := selectProbes(m)
		if len(got) == 0 {
			t.Fatalf("expected probes for mode %s", m)
		}
	}
}
