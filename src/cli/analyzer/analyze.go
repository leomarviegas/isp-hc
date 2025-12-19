package analyzer

import (
	"fmt"

	"isp-checker/probes"
)

// DiagnosticResult provides detailed diagnosis information.
type DiagnosticResult struct {
	Component       string  `json:"component"`
	Confidence      float64 `json:"confidence"`
	Explanation     string  `json:"explanation"`
	SuggestedAction string  `json:"suggested_action"`
	Severity        string  `json:"severity"` // info, warning, critical
}

// Analyze converts probe results into a score, summary, and diagnosis.
func Analyze(results []probes.Result) (float64, string, []string) {
	if len(results) == 0 {
		return 0, "no probes executed", []string{"no data"}
	}

	// Calculate weighted score based on probe results
	totalWeight := 0.0
	weightedScore := 0.0
	diag := []string{}

	for _, p := range results {
		weight := getProbeWeight(p.Name)
		totalWeight += weight

		switch p.Status {
		case probes.StatusOK:
			weightedScore += weight * 100
		case probes.StatusWarn:
			weightedScore += weight * 60
			diag = append(diag, formatDiagnosis(p, "warning"))
		case probes.StatusFail:
			weightedScore += weight * 0
			diag = append(diag, formatDiagnosis(p, "critical"))
		case probes.StatusNA:
			// N/A probes don't count toward score
			totalWeight -= weight
		default:
			diag = append(diag, fmt.Sprintf("%s: unknown status", p.Name))
		}

		// Add detailed diagnostics from probe details
		diag = append(diag, extractDetailedDiagnosis(p)...)
	}

	score := 0.0
	if totalWeight > 0 {
		score = weightedScore / totalWeight
	}

	summary := generateSummary(score, results)
	if len(diag) == 0 {
		diag = []string{"all probes succeeded - no issues detected"}
	}

	return score, summary, diag
}

// AnalyzeDetailed returns structured diagnostic results.
func AnalyzeDetailed(results []probes.Result) (float64, string, []DiagnosticResult) {
	score, summary, simpleDiag := Analyze(results)

	diagnostics := []DiagnosticResult{}

	// Collect specific issues from detailed analysis
	for _, p := range results {
		diag := analyzeProbeResult(p)
		diagnostics = append(diagnostics, diag...)
	}

	// Add overall health status based on score
	if score >= 90 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "Overall",
			Confidence:      1.0,
			Explanation:     "All network probes completed successfully",
			SuggestedAction: "No action required - network is healthy",
			Severity:        "info",
		})
	} else if score >= 70 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "Overall",
			Confidence:      0.85,
			Explanation:     "Network is functional with minor issues",
			SuggestedAction: "Review warnings and monitor for patterns",
			Severity:        "info",
		})
	} else {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "Overall",
			Confidence:      0.9,
			Explanation:     "Network has significant issues requiring attention",
			SuggestedAction: "Address critical issues identified above",
			Severity:        "warning",
		})
	}

	// Include any additional info messages from simple analysis
	for _, msg := range simpleDiag {
		// Skip the generic "all probes succeeded" message
		if msg == "all probes succeeded - no issues detected" {
			continue
		}
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "Network",
			Confidence:      0.8,
			Explanation:     msg,
			SuggestedAction: "Review for potential issues",
			Severity:        "info",
		})
	}

	return score, summary, diagnostics
}

// getProbeWeight returns the weight of a probe for scoring.
func getProbeWeight(name string) float64 {
	weights := map[string]float64{
		"ping":            1.0,
		"dns":             1.0,
		"traceroute":      0.8,
		"interface_stats": 1.2, // Higher weight - hardware issues are serious
		"tcp_stats":       1.2, // Higher weight - transport issues affect everything
		"socket_stats":    0.8,
		"packet_capture":  1.0,
	}
	if w, ok := weights[name]; ok {
		return w
	}
	return 1.0
}

// formatDiagnosis creates a diagnostic message from a probe result.
func formatDiagnosis(p probes.Result, severity string) string {
	message := p.Error
	if message == "" {
		message = fmt.Sprintf("%s check %s", p.Name, severity)
	}
	return fmt.Sprintf("[%s] %s: %s", severity, p.Name, message)
}

// extractDetailedDiagnosis pulls specific issues from probe details.
func extractDetailedDiagnosis(p probes.Result) []string {
	diag := []string{}

	if p.Details == nil {
		return diag
	}

	// Extract issues array if present
	if issues, ok := p.Details["issues"].([]string); ok {
		for _, issue := range issues {
			diag = append(diag, fmt.Sprintf("[%s] %s", p.Name, issue))
		}
	}

	// Extract issues from interface slice
	if issuesI, ok := p.Details["issues"].([]interface{}); ok {
		for _, issue := range issuesI {
			if s, ok := issue.(string); ok {
				diag = append(diag, fmt.Sprintf("[%s] %s", p.Name, s))
			}
		}
	}

	// Check for specific problem indicators
	switch p.Name {
	case "interface_stats":
		if errRate, ok := p.Details["error_rate_percent"].(float64); ok && errRate > 0.1 {
			diag = append(diag, fmt.Sprintf("[interface] packet error rate: %.2f%% (check cables/NIC)", errRate))
		}
		if dropRate, ok := p.Details["drop_rate_percent"].(float64); ok && dropRate > 0.1 {
			diag = append(diag, fmt.Sprintf("[interface] packet drop rate: %.2f%% (check buffer/congestion)", dropRate))
		}

	case "tcp_stats":
		if retrans, ok := p.Details["retransmission_rate"].(float64); ok && retrans > 1.0 {
			diag = append(diag, fmt.Sprintf("[tcp] retransmission rate: %.2f%% (network congestion or loss)", retrans))
		}
		if ofo, ok := p.Details["out_of_order_rate"].(float64); ok && ofo > 0.5 {
			diag = append(diag, fmt.Sprintf("[tcp] out-of-order rate: %.2f%% (path instability)", ofo))
		}

	case "packet_capture":
		if stats, ok := p.Details["stats"].(map[string]interface{}); ok {
			if cksum, ok := stats["checksum_errors"].(float64); ok && cksum > 0 {
				diag = append(diag, fmt.Sprintf("[capture] checksum errors: %.0f (packet corruption detected)", cksum))
			}
			if malformed, ok := stats["malformed_packets"].(float64); ok && malformed > 0 {
				diag = append(diag, fmt.Sprintf("[capture] malformed packets: %.0f (protocol errors)", malformed))
			}
		}
	}

	return diag
}

// analyzeProbeResult creates structured diagnostics from a probe result.
func analyzeProbeResult(p probes.Result) []DiagnosticResult {
	diagnostics := []DiagnosticResult{}

	switch p.Name {
	case "ping":
		diagnostics = append(diagnostics, analyzePing(p)...)
	case "dns":
		diagnostics = append(diagnostics, analyzeDNS(p)...)
	case "interface_stats":
		diagnostics = append(diagnostics, analyzeInterfaceStats(p)...)
	case "tcp_stats":
		diagnostics = append(diagnostics, analyzeTCPStats(p)...)
	case "packet_capture":
		diagnostics = append(diagnostics, analyzePacketCapture(p)...)
	}

	return diagnostics
}

func analyzePing(p probes.Result) []DiagnosticResult {
	if p.Status == probes.StatusOK {
		return nil
	}

	return []DiagnosticResult{{
		Component:       "Connectivity",
		Confidence:      0.9,
		Explanation:     "Ping failed - target may be unreachable or blocking ICMP",
		SuggestedAction: "Check network connectivity, firewall rules, and target availability",
		Severity:        "critical",
	}}
}

func analyzeDNS(p probes.Result) []DiagnosticResult {
	if p.Status == probes.StatusOK {
		return nil
	}

	return []DiagnosticResult{{
		Component:       "DNS",
		Confidence:      0.85,
		Explanation:     "DNS resolution failed",
		SuggestedAction: "Check DNS server configuration, try alternate DNS (8.8.8.8, 1.1.1.1)",
		Severity:        "critical",
	}}
}

func analyzeInterfaceStats(p probes.Result) []DiagnosticResult {
	diagnostics := []DiagnosticResult{}

	if p.Details == nil {
		return diagnostics
	}

	errRate, _ := p.Details["error_rate_percent"].(float64)
	dropRate, _ := p.Details["drop_rate_percent"].(float64)

	if errRate > 1.0 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "NetworkInterface",
			Confidence:      0.9,
			Explanation:     fmt.Sprintf("High packet error rate (%.2f%%) - likely hardware or cable issue", errRate),
			SuggestedAction: "Check network cable connections, test with different cable, check NIC health",
			Severity:        "critical",
		})
	} else if errRate > 0.1 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "NetworkInterface",
			Confidence:      0.8,
			Explanation:     fmt.Sprintf("Elevated packet error rate (%.2f%%) - possible interference or degraded cable", errRate),
			SuggestedAction: "Inspect cables for damage, check for electromagnetic interference",
			Severity:        "warning",
		})
	}

	if dropRate > 1.0 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "NetworkInterface",
			Confidence:      0.85,
			Explanation:     fmt.Sprintf("High packet drop rate (%.2f%%) - buffer overflow or congestion", dropRate),
			SuggestedAction: "Check for bandwidth saturation, increase ring buffer size, check for driver issues",
			Severity:        "critical",
		})
	}

	return diagnostics
}

func analyzeTCPStats(p probes.Result) []DiagnosticResult {
	diagnostics := []DiagnosticResult{}

	if p.Details == nil {
		return diagnostics
	}

	retransRate, _ := p.Details["retransmission_rate"].(float64)
	ofoRate, _ := p.Details["out_of_order_rate"].(float64)
	reorderEvents, _ := p.Details["total_reorder_events"].(float64)

	if retransRate > 5.0 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "TCPTransport",
			Confidence:      0.9,
			Explanation:     fmt.Sprintf("High TCP retransmission rate (%.2f%%) - significant packet loss on network path", retransRate),
			SuggestedAction: "Check for network congestion, faulty equipment along path, or ISP issues",
			Severity:        "critical",
		})
	} else if retransRate > 1.0 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "TCPTransport",
			Confidence:      0.85,
			Explanation:     fmt.Sprintf("Elevated TCP retransmissions (%.2f%%) - some packet loss occurring", retransRate),
			SuggestedAction: "Monitor for pattern (time of day, specific destinations), check local network first",
			Severity:        "warning",
		})
	}

	if ofoRate > 1.0 || reorderEvents > 100 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "TCPTransport",
			Confidence:      0.8,
			Explanation:     fmt.Sprintf("Packet reordering detected (%.2f%% OFO, %.0f events) - path instability or load balancing", ofoRate, reorderEvents),
			SuggestedAction: "This may indicate asymmetric routing or multi-path issues, typically not actionable",
			Severity:        "warning",
		})
	}

	return diagnostics
}

func analyzePacketCapture(p probes.Result) []DiagnosticResult {
	diagnostics := []DiagnosticResult{}

	if p.Details == nil {
		return diagnostics
	}

	stats, ok := p.Details["stats"].(map[string]interface{})
	if !ok {
		return diagnostics
	}

	checksumErrors, _ := stats["checksum_errors"].(float64)
	malformed, _ := stats["malformed_packets"].(float64)
	retransmits, _ := stats["tcp_retransmits"].(float64)
	ooo, _ := stats["tcp_out_of_order"].(float64)
	zeroWindow, _ := stats["tcp_zero_window"].(float64)

	if checksumErrors > 0 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "PacketIntegrity",
			Confidence:      0.95,
			Explanation:     fmt.Sprintf("Checksum errors detected (%.0f packets) - PACKET CORRUPTION occurring", checksumErrors),
			SuggestedAction: "This is serious: check cables, NIC, switches. May indicate failing hardware",
			Severity:        "critical",
		})
	}

	if malformed > 0 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "PacketIntegrity",
			Confidence:      0.9,
			Explanation:     fmt.Sprintf("Malformed packets detected (%.0f) - protocol violations or corruption", malformed),
			SuggestedAction: "Check for MTU issues, middlebox interference, or faulty network equipment",
			Severity:        "critical",
		})
	}

	if zeroWindow > 10 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "TCPFlow",
			Confidence:      0.85,
			Explanation:     fmt.Sprintf("TCP zero window events (%.0f) - receiver cannot keep up", zeroWindow),
			SuggestedAction: "The receiving application is overwhelmed, check target server performance",
			Severity:        "warning",
		})
	}

	if retransmits > 50 && ooo > 20 {
		diagnostics = append(diagnostics, DiagnosticResult{
			Component:       "NetworkPath",
			Confidence:      0.8,
			Explanation:     fmt.Sprintf("High retransmits (%.0f) with reordering (%.0f) - unstable network path", retransmits, ooo),
			SuggestedAction: "Path is experiencing issues - could be congestion, flapping route, or failing link",
			Severity:        "warning",
		})
	}

	return diagnostics
}

// generateSummary creates a human-readable summary based on score and results.
func generateSummary(score float64, results []probes.Result) string {
	// Count issues by severity
	critical := 0
	warnings := 0
	packetIssues := false

	for _, p := range results {
		switch p.Status {
		case probes.StatusFail:
			critical++
		case probes.StatusWarn:
			warnings++
		}

		// Check for packet-level issues
		if p.Name == "interface_stats" || p.Name == "tcp_stats" || p.Name == "packet_capture" {
			if p.Status != probes.StatusOK && p.Status != probes.StatusNA {
				packetIssues = true
			}
		}
	}

	if score >= 90 {
		return "Network health is excellent - all systems operational"
	} else if score >= 70 {
		if packetIssues {
			return fmt.Sprintf("Network functional with packet-level issues (%d warnings)", warnings)
		}
		return fmt.Sprintf("Network health is good with minor issues (%d warnings)", warnings)
	} else if score >= 50 {
		if packetIssues {
			return fmt.Sprintf("Network degraded - packet corruption or loss detected (%d critical, %d warnings)", critical, warnings)
		}
		return fmt.Sprintf("Network health degraded (%d critical issues, %d warnings)", critical, warnings)
	} else {
		if packetIssues {
			return fmt.Sprintf("Network severely impaired - significant packet issues detected (%d critical)", critical)
		}
		return fmt.Sprintf("Network health critical (%d failures)", critical)
	}
}
