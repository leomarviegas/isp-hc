package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"isp-checker/analyzer"
	"isp-checker/probes"
)

// RunOptions captures CLI parameters.
type RunOptions struct {
	Target         string
	Mode           string
	OutPath        string
	SimulationPath string
}

// RunResult is the aggregate output schema emitted by the CLI.
type RunResult struct {
	RunID     string                   `json:"run_id"`
	Timestamp string                   `json:"timestamp"`
	Target    string                   `json:"target"`
	Mode      string                   `json:"mode"`
	Score     float64                  `json:"score"`
	Summary   string                   `json:"summary"`
	Probes    []probes.Result          `json:"probes"`
	Diagnosis []map[string]interface{} `json:"diagnosis"`
	Raw       map[string]interface{}   `json:"raw,omitempty"`
}

// JSON renders the result as pretty JSON.
func (r RunResult) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func ExecuteRun(ctx context.Context, opts RunOptions) (RunResult, error) {
	if opts.SimulationPath != "" {
		return loadSimulation(opts.SimulationPath)
	}

	runID := uuid.NewString()
	probeResults := []probes.Result{}
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	selected := selectProbes(strings.ToLower(opts.Mode))
	if len(selected) == 0 {
		return RunResult{}, fmt.Errorf("unknown mode: %s", opts.Mode)
	}

	for _, probeFn := range selected {
		wg.Add(1)
		go func(p probeFunc) {
			defer wg.Done()
			result := p(ctx, opts.Target)
			recordProbeMetrics(result)
			mu.Lock()
			probeResults = append(probeResults, result)
			mu.Unlock()
		}(probeFn)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return RunResult{}, ctx.Err()
	case <-done:
	}

	result := RunResult{
		RunID:     runID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Target:    opts.Target,
		Mode:      opts.Mode,
		Probes:    probeResults,
		Raw:       map[string]interface{}{},
	}

	score, summary, detailedDiag := analyzer.AnalyzeDetailed(probeResults)
	result.Score = score
	result.Summary = summary
	// Use detailed diagnosis with confidence, fallback to simple if empty
	if len(detailedDiag) > 0 {
		result.Diagnosis = toDetailedDiagnosis(detailedDiag)
	} else {
		_, _, simpleDiag := analyzer.Analyze(probeResults)
		result.Diagnosis = toDiagnosis(simpleDiag)
	}
	recordRunMetrics(result)
	return result, nil
}

func writeResult(result RunResult, outPath string) error {
	if outPath == "" {
		return nil
	}
	b, err := result.JSON()
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, b, 0o644)
}

func loadSimulation(path string) (RunResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return RunResult{}, err
	}
	var res RunResult
	if err := json.Unmarshal(b, &res); err != nil {
		return RunResult{}, err
	}
	if res.RunID == "" {
		res.RunID = uuid.NewString()
	}
	if res.Timestamp == "" {
		res.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if res.Target == "" {
		res.Target = "simulation"
	}
	if res.Mode == "" {
		res.Mode = "simulation"
	}
	// Ensure probes slice exists to keep schema stable.
	if res.Probes == nil {
		res.Probes = []probes.Result{}
	}
	score, summary, detailedDiag := analyzer.AnalyzeDetailed(res.Probes)
	if res.Score == 0 {
		res.Score = score
	}
	if res.Summary == "" {
		res.Summary = summary
	}
	if len(res.Diagnosis) == 0 {
		if len(detailedDiag) > 0 {
			res.Diagnosis = toDetailedDiagnosis(detailedDiag)
		} else {
			_, _, simpleDiag := analyzer.Analyze(res.Probes)
			res.Diagnosis = toDiagnosis(simpleDiag)
		}
	}
	return res, nil
}

type probeFunc func(context.Context, string) probes.Result

func selectProbes(mode string) []probeFunc {
	switch mode {
	case "ping":
		return []probeFunc{probes.RunPing}
	case "dns":
		return []probeFunc{probes.RunDNS}
	case "traceroute":
		return []probeFunc{probes.RunTraceroute}
	case "interface", "interface_stats":
		return []probeFunc{probes.RunInterfaceStats}
	case "tcp", "tcp_stats":
		return []probeFunc{probes.RunTCPStats}
	case "socket", "socket_stats":
		return []probeFunc{probes.RunSocketStats}
	case "capture", "packet_capture":
		return []probeFunc{probes.RunPacketCapture}
	case "packet", "packet_health":
		// All packet health probes (interface + TCP + socket + capture)
		return []probeFunc{
			probes.RunInterfaceStats,
			probes.RunTCPStats,
			probes.RunSocketStats,
			probes.RunPacketCapture,
		}
	case "full", "default", "":
		// Basic connectivity probes
		return []probeFunc{probes.RunPing, probes.RunDNS, probes.RunTraceroute}
	case "comprehensive", "all":
		// All probes including packet health
		return []probeFunc{
			probes.RunPing,
			probes.RunDNS,
			probes.RunTraceroute,
			probes.RunInterfaceStats,
			probes.RunTCPStats,
			probes.RunSocketStats,
			probes.RunPacketCapture,
		}
	default:
		return nil
	}
}

func toDiagnosis(diag []string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(diag))
	for _, d := range diag {
		out = append(out, map[string]interface{}{"message": d})
	}
	return out
}

func toDetailedDiagnosis(diag []analyzer.DiagnosticResult) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(diag))
	for _, d := range diag {
		out = append(out, map[string]interface{}{
			"component":        d.Component,
			"confidence":       d.Confidence,
			"explanation":      d.Explanation,
			"suggested_action": d.SuggestedAction,
			"severity":         d.Severity,
		})
	}
	return out
}
