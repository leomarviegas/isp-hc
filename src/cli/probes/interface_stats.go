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

// getProcPath returns the proc filesystem path, supporting container environments
// where host /proc may be mounted at a different location (e.g., /host/proc).
func getProcPath() string {
	if path := os.Getenv("HOST_PROC"); path != "" {
		return path
	}
	return "/proc"
}

// getSysPath returns the sysfs path, supporting container environments.
func getSysPath() string {
	if path := os.Getenv("HOST_SYS"); path != "" {
		return path
	}
	return "/sys"
}

// InterfaceStats holds network interface error counters.
type InterfaceStats struct {
	Interface string `json:"interface"`

	// Receive errors
	RxBytes      uint64 `json:"rx_bytes"`
	RxPackets    uint64 `json:"rx_packets"`
	RxErrors     uint64 `json:"rx_errors"`      // Total receive errors
	RxDropped    uint64 `json:"rx_dropped"`     // Packets dropped
	RxFIFO       uint64 `json:"rx_fifo"`        // FIFO buffer errors
	RxFrame      uint64 `json:"rx_frame"`       // Frame alignment errors
	RxCompressed uint64 `json:"rx_compressed"`
	RxMulticast  uint64 `json:"rx_multicast"`

	// Transmit errors
	TxBytes      uint64 `json:"tx_bytes"`
	TxPackets    uint64 `json:"tx_packets"`
	TxErrors     uint64 `json:"tx_errors"`      // Total transmit errors
	TxDropped    uint64 `json:"tx_dropped"`     // Packets dropped
	TxFIFO       uint64 `json:"tx_fifo"`        // FIFO buffer errors
	TxCollisions uint64 `json:"tx_collisions"`  // Collision count
	TxCarrier    uint64 `json:"tx_carrier"`     // Carrier errors
	TxCompressed uint64 `json:"tx_compressed"`

	// Extended stats (from ethtool/sysfs when available)
	RxCRCErrors      uint64 `json:"rx_crc_errors,omitempty"`       // CRC/checksum errors
	RxLengthErrors   uint64 `json:"rx_length_errors,omitempty"`    // Length errors
	RxOverErrors     uint64 `json:"rx_over_errors,omitempty"`      // Receiver ring buffer overflow
	RxMissedErrors   uint64 `json:"rx_missed_errors,omitempty"`    // Missed packets
	TxAbortedErrors  uint64 `json:"tx_aborted_errors,omitempty"`   // Aborted transmissions
	TxHeartbeatErrors uint64 `json:"tx_heartbeat_errors,omitempty"` // Heartbeat errors
	TxWindowErrors   uint64 `json:"tx_window_errors,omitempty"`    // Window errors
}

// RunInterfaceStats collects network interface error statistics.
func RunInterfaceStats(ctx context.Context, target string) Result {
	if runtime.GOOS != "linux" {
		return Result{
			Name:   "interface_stats",
			Status: StatusNA,
			Error:  "interface stats only supported on Linux",
		}
	}

	stats, err := collectInterfaceStats()
	if err != nil {
		return Result{
			Name:   "interface_stats",
			Status: StatusFail,
			Error:  err.Error(),
		}
	}

	// Analyze the stats for problems
	totalErrors := uint64(0)
	totalDropped := uint64(0)
	totalPackets := uint64(0)
	problemInterfaces := []string{}

	for _, s := range stats {
		// Skip loopback and virtual interfaces
		if s.Interface == "lo" || strings.HasPrefix(s.Interface, "veth") ||
			strings.HasPrefix(s.Interface, "docker") || strings.HasPrefix(s.Interface, "br-") {
			continue
		}

		errors := s.RxErrors + s.TxErrors + s.RxCRCErrors + s.RxFrame
		dropped := s.RxDropped + s.TxDropped
		packets := s.RxPackets + s.TxPackets

		totalErrors += errors
		totalDropped += dropped
		totalPackets += packets

		if errors > 0 || dropped > 0 {
			problemInterfaces = append(problemInterfaces, s.Interface)
		}
	}

	// Calculate error rate
	errorRate := float64(0)
	dropRate := float64(0)
	if totalPackets > 0 {
		errorRate = float64(totalErrors) / float64(totalPackets) * 100
		dropRate = float64(totalDropped) / float64(totalPackets) * 100
	}

	// Determine status
	status := StatusOK
	if errorRate > 1.0 || dropRate > 1.0 {
		status = StatusFail
	} else if errorRate > 0.1 || dropRate > 0.1 {
		status = "warn"
	}

	// Build details map
	details := map[string]interface{}{
		"interfaces":         stats,
		"total_errors":       totalErrors,
		"total_dropped":      totalDropped,
		"total_packets":      totalPackets,
		"error_rate_percent": errorRate,
		"drop_rate_percent":  dropRate,
	}

	if len(problemInterfaces) > 0 {
		details["problem_interfaces"] = problemInterfaces
	}

	return Result{
		Name:    "interface_stats",
		Status:  status,
		Details: details,
	}
}

// collectInterfaceStats reads from /proc/net/dev and sysfs.
func collectInterfaceStats() ([]InterfaceStats, error) {
	procPath := filepath.Join(getProcPath(), "net", "dev")
	file, err := os.Open(procPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", procPath, err)
	}
	defer file.Close()

	var stats []InterfaceStats
	scanner := bufio.NewScanner(file)

	// Skip header lines
	scanner.Scan() // "Inter-|   Receive ..."
	scanner.Scan() // " face |bytes ..."

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse: "eth0: 123 456 789 ..."
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		s := InterfaceStats{Interface: iface}

		// Receive fields (0-7)
		s.RxBytes, _ = strconv.ParseUint(fields[0], 10, 64)
		s.RxPackets, _ = strconv.ParseUint(fields[1], 10, 64)
		s.RxErrors, _ = strconv.ParseUint(fields[2], 10, 64)
		s.RxDropped, _ = strconv.ParseUint(fields[3], 10, 64)
		s.RxFIFO, _ = strconv.ParseUint(fields[4], 10, 64)
		s.RxFrame, _ = strconv.ParseUint(fields[5], 10, 64)
		s.RxCompressed, _ = strconv.ParseUint(fields[6], 10, 64)
		s.RxMulticast, _ = strconv.ParseUint(fields[7], 10, 64)

		// Transmit fields (8-15)
		s.TxBytes, _ = strconv.ParseUint(fields[8], 10, 64)
		s.TxPackets, _ = strconv.ParseUint(fields[9], 10, 64)
		s.TxErrors, _ = strconv.ParseUint(fields[10], 10, 64)
		s.TxDropped, _ = strconv.ParseUint(fields[11], 10, 64)
		s.TxFIFO, _ = strconv.ParseUint(fields[12], 10, 64)
		s.TxCollisions, _ = strconv.ParseUint(fields[13], 10, 64)
		s.TxCarrier, _ = strconv.ParseUint(fields[14], 10, 64)
		s.TxCompressed, _ = strconv.ParseUint(fields[15], 10, 64)

		// Try to get extended stats from sysfs
		readExtendedStats(&s)

		stats = append(stats, s)
	}

	return stats, scanner.Err()
}

// readExtendedStats attempts to read extended statistics from sysfs.
func readExtendedStats(s *InterfaceStats) {
	base := filepath.Join(getSysPath(), "class", "net", s.Interface, "statistics")

	s.RxCRCErrors = readSysfsCounter(filepath.Join(base, "rx_crc_errors"))
	s.RxLengthErrors = readSysfsCounter(filepath.Join(base, "rx_length_errors"))
	s.RxOverErrors = readSysfsCounter(filepath.Join(base, "rx_over_errors"))
	s.RxMissedErrors = readSysfsCounter(filepath.Join(base, "rx_missed_errors"))
	s.TxAbortedErrors = readSysfsCounter(filepath.Join(base, "tx_aborted_errors"))
	s.TxHeartbeatErrors = readSysfsCounter(filepath.Join(base, "tx_heartbeat_errors"))
	s.TxWindowErrors = readSysfsCounter(filepath.Join(base, "tx_window_errors"))
}

// readSysfsCounter reads a single counter value from sysfs.
func readSysfsCounter(path string) uint64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	return val
}
