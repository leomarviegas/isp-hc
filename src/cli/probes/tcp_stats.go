package probes

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// TCPStats holds TCP protocol statistics.
type TCPStats struct {
	// Connection stats
	ActiveOpens  uint64 `json:"active_opens"`   // Active connection openings
	PassiveOpens uint64 `json:"passive_opens"`  // Passive connection openings
	AttemptFails uint64 `json:"attempt_fails"`  // Failed connection attempts
	EstabResets  uint64 `json:"estab_resets"`   // Resets received on established connections
	CurrEstab    uint64 `json:"curr_estab"`     // Currently established connections

	// Segment stats
	InSegs       uint64 `json:"in_segs"`        // Segments received
	OutSegs      uint64 `json:"out_segs"`       // Segments sent
	RetransSegs  uint64 `json:"retrans_segs"`   // Segments retransmitted
	InErrs       uint64 `json:"in_errs"`        // Bad segments received
	OutRsts      uint64 `json:"out_rsts"`       // RST segments sent

	// Extended stats (from /proc/net/netstat)
	TCPLostRetransmit    uint64 `json:"tcp_lost_retransmit"`     // Retransmits lost
	TCPFastRetrans       uint64 `json:"tcp_fast_retrans"`        // Fast retransmissions
	TCPSlowStartRetrans  uint64 `json:"tcp_slow_start_retrans"`  // Slow start retransmissions
	TCPTimeouts          uint64 `json:"tcp_timeouts"`            // Timeout retransmissions

	// Packet reordering
	TCPReorderDetected   uint64 `json:"tcp_reorder_detected"`    // Reordering detected (FACK)
	TCPTSReorder         uint64 `json:"tcp_ts_reorder"`          // Reordering detected (timestamp)
	TCPSACKReorder       uint64 `json:"tcp_sack_reorder"`        // Reordering detected (SACK)
	TCPRenoReorder       uint64 `json:"tcp_reno_reorder"`        // Reordering detected (Reno)

	// Duplicate ACKs and SACKs
	TCPDSACKOldSent      uint64 `json:"tcp_dsack_old_sent"`      // DSACK sent (old)
	TCPDSACKOfoSent      uint64 `json:"tcp_dsack_ofo_sent"`      // DSACK sent (out of order)
	TCPDSACKRecv         uint64 `json:"tcp_dsack_recv"`          // DSACK received
	TCPDSACKOfoRecv      uint64 `json:"tcp_dsack_ofo_recv"`      // DSACK received (out of order)

	// Out of order packets
	TCPOFOQueue          uint64 `json:"tcp_ofo_queue"`           // Packets queued in OFO queue
	TCPOFODrop           uint64 `json:"tcp_ofo_drop"`            // Packets dropped from OFO queue
	TCPOFOMerge          uint64 `json:"tcp_ofo_merge"`           // Packets merged in OFO queue

	// Checksum errors
	TCPMDChecksumFail    uint64 `json:"tcp_md_checksum_fail"`    // MD5 checksum failures

	// Connection issues
	TCPAbortOnData       uint64 `json:"tcp_abort_on_data"`       // Connections aborted due to data
	TCPAbortOnClose      uint64 `json:"tcp_abort_on_close"`      // Connections aborted on close
	TCPAbortOnMemory     uint64 `json:"tcp_abort_on_memory"`     // Connections aborted on memory pressure
	TCPAbortOnTimeout    uint64 `json:"tcp_abort_on_timeout"`    // Connections aborted on timeout
	TCPAbortOnLinger     uint64 `json:"tcp_abort_on_linger"`     // Connections aborted on linger timeout
	TCPAbortFailed       uint64 `json:"tcp_abort_failed"`        // Abort failures

	// Memory pressure
	TCPMemoryPressures   uint64 `json:"tcp_memory_pressures"`    // Memory pressure events
	PruneCalled          uint64 `json:"prune_called"`            // Socket buffer pruned

	// Spurious retransmissions
	TCPSpuriousRTOs      uint64 `json:"tcp_spurious_rtos"`       // Spurious RTOs
	TCPSpuriousRtxHostQueues uint64 `json:"tcp_spurious_rtx_host_queues"` // Spurious retx (host queues)
}

// RunTCPStats collects TCP protocol statistics.
func RunTCPStats(ctx context.Context, target string) Result {
	if runtime.GOOS != "linux" {
		return Result{
			Name:   "tcp_stats",
			Status: StatusNA,
			Error:  "TCP stats only supported on Linux",
		}
	}

	stats, err := collectTCPStats()
	if err != nil {
		return Result{
			Name:   "tcp_stats",
			Status: StatusFail,
			Error:  err.Error(),
		}
	}

	// Calculate key metrics
	totalSegs := stats.InSegs + stats.OutSegs
	retransRate := float64(0)
	errorRate := float64(0)
	ofoRate := float64(0)

	if totalSegs > 0 {
		retransRate = float64(stats.RetransSegs) / float64(stats.OutSegs) * 100
		errorRate = float64(stats.InErrs) / float64(stats.InSegs) * 100
	}

	totalReorder := stats.TCPReorderDetected + stats.TCPTSReorder + stats.TCPSACKReorder + stats.TCPRenoReorder
	if stats.InSegs > 0 {
		ofoRate = float64(stats.TCPOFOQueue) / float64(stats.InSegs) * 100
	}

	// Determine status based on thresholds
	status := StatusOK
	issues := []string{}

	// Retransmission rate > 5% is critical, > 1% is warning
	if retransRate > 5.0 {
		status = StatusFail
		issues = append(issues, fmt.Sprintf("high retransmission rate: %.2f%%", retransRate))
	} else if retransRate > 1.0 {
		if status != StatusFail {
			status = "warn"
		}
		issues = append(issues, fmt.Sprintf("elevated retransmission rate: %.2f%%", retransRate))
	}

	// Error rate > 1% is critical
	if errorRate > 1.0 {
		status = StatusFail
		issues = append(issues, fmt.Sprintf("high TCP error rate: %.2f%%", errorRate))
	} else if errorRate > 0.1 {
		if status != StatusFail {
			status = "warn"
		}
		issues = append(issues, fmt.Sprintf("elevated TCP error rate: %.2f%%", errorRate))
	}

	// Out-of-order packets
	if totalReorder > 0 || stats.TCPOFOQueue > 0 {
		if ofoRate > 1.0 {
			if status != StatusFail {
				status = "warn"
			}
			issues = append(issues, fmt.Sprintf("packet reordering detected: %d events", totalReorder))
		}
	}

	// Timeouts and aborts
	totalAborts := stats.TCPAbortOnData + stats.TCPAbortOnClose + stats.TCPAbortOnTimeout + stats.TCPAbortOnMemory
	if totalAborts > 100 { // Threshold for significant aborts
		issues = append(issues, fmt.Sprintf("connection aborts: %d", totalAborts))
	}

	// Memory pressure
	if stats.TCPMemoryPressures > 0 {
		issues = append(issues, fmt.Sprintf("TCP memory pressure events: %d", stats.TCPMemoryPressures))
	}

	// Build details
	details := map[string]interface{}{
		"stats":                    stats,
		"retransmission_rate":      retransRate,
		"error_rate":               errorRate,
		"out_of_order_rate":        ofoRate,
		"total_reorder_events":     totalReorder,
		"current_connections":      stats.CurrEstab,
	}

	if len(issues) > 0 {
		details["issues"] = issues
	}

	return Result{
		Name:    "tcp_stats",
		Status:  status,
		Details: details,
	}
}

// collectTCPStats reads from /proc/net/snmp and /proc/net/netstat.
func collectTCPStats() (*TCPStats, error) {
	stats := &TCPStats{}

	// Read /proc/net/snmp for basic TCP stats
	if err := readSNMPStats(stats); err != nil {
		return nil, fmt.Errorf("failed to read /proc/net/snmp: %w", err)
	}

	// Read /proc/net/netstat for extended TCP stats
	if err := readNetstatStats(stats); err != nil {
		// Not fatal - extended stats may not be available
		// Just continue with basic stats
	}

	return stats, nil
}

// readSNMPStats reads TCP stats from /proc/net/snmp.
func readSNMPStats(stats *TCPStats) error {
	file, err := os.Open(filepath.Join(getProcPath(), "net", "snmp"))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var tcpHeader []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "Tcp:") {
			fields := strings.Fields(line)
			if tcpHeader == nil {
				// This is the header line
				tcpHeader = fields
			} else {
				// This is the values line
				for i, header := range tcpHeader {
					if i >= len(fields) {
						break
					}
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					switch header {
					case "ActiveOpens":
						stats.ActiveOpens = val
					case "PassiveOpens":
						stats.PassiveOpens = val
					case "AttemptFails":
						stats.AttemptFails = val
					case "EstabResets":
						stats.EstabResets = val
					case "CurrEstab":
						stats.CurrEstab = val
					case "InSegs":
						stats.InSegs = val
					case "OutSegs":
						stats.OutSegs = val
					case "RetransSegs":
						stats.RetransSegs = val
					case "InErrs":
						stats.InErrs = val
					case "OutRsts":
						stats.OutRsts = val
					}
				}
			}
		}
	}

	return scanner.Err()
}

// readNetstatStats reads extended TCP stats from /proc/net/netstat.
func readNetstatStats(stats *TCPStats) error {
	file, err := os.Open(filepath.Join(getProcPath(), "net", "netstat"))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var tcpExtHeader []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "TcpExt:") {
			fields := strings.Fields(line)
			if tcpExtHeader == nil {
				// Header line
				tcpExtHeader = fields
			} else {
				// Values line
				for i, header := range tcpExtHeader {
					if i >= len(fields) {
						break
					}
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					switch header {
					case "TCPLostRetransmit":
						stats.TCPLostRetransmit = val
					case "TCPFastRetrans":
						stats.TCPFastRetrans = val
					case "TCPSlowStartRetrans":
						stats.TCPSlowStartRetrans = val
					case "TCPTimeouts":
						stats.TCPTimeouts = val
					case "TCPReorderDetected":
						stats.TCPReorderDetected = val
					case "TCPTSReorder":
						stats.TCPTSReorder = val
					case "TCPSACKReorder":
						stats.TCPSACKReorder = val
					case "TCPRenoReorder":
						stats.TCPRenoReorder = val
					case "TCPDSACKOldSent":
						stats.TCPDSACKOldSent = val
					case "TCPDSACKOfoSent":
						stats.TCPDSACKOfoSent = val
					case "TCPDSACKRecv":
						stats.TCPDSACKRecv = val
					case "TCPDSACKOfoRecv":
						stats.TCPDSACKOfoRecv = val
					case "TCPOFOQueue":
						stats.TCPOFOQueue = val
					case "TCPOFODrop":
						stats.TCPOFODrop = val
					case "TCPOFOMerge":
						stats.TCPOFOMerge = val
					case "TCPAbortOnData":
						stats.TCPAbortOnData = val
					case "TCPAbortOnClose":
						stats.TCPAbortOnClose = val
					case "TCPAbortOnMemory":
						stats.TCPAbortOnMemory = val
					case "TCPAbortOnTimeout":
						stats.TCPAbortOnTimeout = val
					case "TCPAbortOnLinger":
						stats.TCPAbortOnLinger = val
					case "TCPAbortFailed":
						stats.TCPAbortFailed = val
					case "TCPMemoryPressures":
						stats.TCPMemoryPressures = val
					case "PruneCalled":
						stats.PruneCalled = val
					case "TCPSpuriousRTOs":
						stats.TCPSpuriousRTOs = val
					}
				}
			}
		}
	}

	return scanner.Err()
}
