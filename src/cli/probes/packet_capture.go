package probes

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// PacketCaptureStats holds results from packet capture analysis.
type PacketCaptureStats struct {
	// Capture metadata
	Duration      float64 `json:"duration_seconds"`
	Interface     string  `json:"interface"`
	PacketCount   uint64  `json:"packet_count"`

	// TCP analysis
	TCPPackets        uint64  `json:"tcp_packets"`
	TCPSynPackets     uint64  `json:"tcp_syn_packets"`
	TCPFinPackets     uint64  `json:"tcp_fin_packets"`
	TCPRstPackets     uint64  `json:"tcp_rst_packets"`
	TCPRetransmits    uint64  `json:"tcp_retransmits"`
	TCPDuplicateAcks  uint64  `json:"tcp_duplicate_acks"`
	TCPOutOfOrder     uint64  `json:"tcp_out_of_order"`
	TCPZeroWindow     uint64  `json:"tcp_zero_window"`
	TCPWindowFull     uint64  `json:"tcp_window_full"`
	TCPKeepAlive      uint64  `json:"tcp_keep_alive"`

	// ICMP analysis
	ICMPPackets           uint64 `json:"icmp_packets"`
	ICMPUnreachable       uint64 `json:"icmp_unreachable"`
	ICMPTimeExceeded      uint64 `json:"icmp_time_exceeded"`
	ICMPRedirect          uint64 `json:"icmp_redirect"`

	// Error indicators
	MalformedPackets      uint64 `json:"malformed_packets"`
	ChecksumErrors        uint64 `json:"checksum_errors"`
	FragmentedPackets     uint64 `json:"fragmented_packets"`
	FragmentReassemblyFails uint64 `json:"fragment_reassembly_fails"`

	// Latency analysis (if target ping included)
	AvgLatencyMs    float64 `json:"avg_latency_ms,omitempty"`
	MaxLatencyMs    float64 `json:"max_latency_ms,omitempty"`
	MinLatencyMs    float64 `json:"min_latency_ms,omitempty"`
	Jitter          float64 `json:"jitter_ms,omitempty"`
}

// RunPacketCapture performs a brief packet capture and analysis.
// Requires tcpdump and appropriate privileges (root or CAP_NET_RAW).
func RunPacketCapture(ctx context.Context, target string) Result {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return Result{
			Name:   "packet_capture",
			Status: StatusNA,
			Error:  "packet capture only supported on Linux and macOS",
		}
	}

	// Check if tcpdump is available
	tcpdumpPath, err := exec.LookPath("tcpdump")
	if err != nil {
		return Result{
			Name:   "packet_capture",
			Status: StatusNA,
			Error:  "tcpdump not available",
		}
	}

	// Determine the interface to capture on
	iface, err := getDefaultInterface()
	if err != nil {
		iface = "any" // Fallback to capture on all interfaces
	}

	// Create a context with timeout for capture duration
	captureCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Run tcpdump with output suitable for parsing
	// -nn: don't resolve names
	// -v: verbose (for checksum info)
	// -c 1000: capture max 1000 packets
	// -i: interface
	args := []string{
		"-nn", "-v",
		"-c", "1000",
		"-i", iface,
	}

	// Filter for target if specified (not just 'any')
	if target != "" && target != "any" {
		args = append(args, "host", target)
	}

	cmd := exec.CommandContext(captureCtx, tcpdumpPath, args...)
	output, err := cmd.CombinedOutput()

	// Context timeout is expected - we capture for a duration
	if err != nil && captureCtx.Err() != context.DeadlineExceeded {
		// Check for permission issues
		if strings.Contains(string(output), "permission denied") ||
			strings.Contains(string(output), "Operation not permitted") {
			return Result{
				Name:   "packet_capture",
				Status: StatusNA,
				Error:  "insufficient privileges for packet capture (requires root or CAP_NET_RAW)",
			}
		}
	}

	// Parse tcpdump output
	stats := parsePacketCapture(string(output), iface)

	// Analyze results
	status := StatusOK
	issues := []string{}

	// Check for problems
	if stats.PacketCount > 0 {
		// Retransmission rate
		if stats.TCPPackets > 0 {
			retransRate := float64(stats.TCPRetransmits) / float64(stats.TCPPackets) * 100
			if retransRate > 5.0 {
				status = StatusFail
				issues = append(issues, fmt.Sprintf("high retransmission rate: %.1f%%", retransRate))
			} else if retransRate > 1.0 {
				status = "warn"
				issues = append(issues, fmt.Sprintf("elevated retransmissions: %.1f%%", retransRate))
			}
		}

		// Out of order packets
		if stats.TCPOutOfOrder > 10 {
			if status != StatusFail {
				status = "warn"
			}
			issues = append(issues, fmt.Sprintf("out-of-order packets: %d", stats.TCPOutOfOrder))
		}

		// Duplicate ACKs (may indicate loss)
		if stats.TCPDuplicateAcks > 20 {
			issues = append(issues, fmt.Sprintf("duplicate ACKs: %d (possible packet loss)", stats.TCPDuplicateAcks))
		}

		// RST packets (connection resets)
		if stats.TCPRstPackets > 10 {
			issues = append(issues, fmt.Sprintf("TCP resets: %d", stats.TCPRstPackets))
		}

		// Zero window (receiver overloaded)
		if stats.TCPZeroWindow > 0 {
			issues = append(issues, fmt.Sprintf("TCP zero window events: %d (receiver buffer full)", stats.TCPZeroWindow))
		}

		// Malformed packets
		if stats.MalformedPackets > 0 {
			status = StatusFail
			issues = append(issues, fmt.Sprintf("malformed packets: %d", stats.MalformedPackets))
		}

		// Checksum errors
		if stats.ChecksumErrors > 0 {
			status = StatusFail
			issues = append(issues, fmt.Sprintf("checksum errors: %d (packet corruption)", stats.ChecksumErrors))
		}

		// ICMP unreachable
		if stats.ICMPUnreachable > 0 {
			issues = append(issues, fmt.Sprintf("ICMP unreachable: %d", stats.ICMPUnreachable))
		}
	} else {
		status = "warn"
		issues = append(issues, "no packets captured (check interface/permissions)")
	}

	details := map[string]interface{}{
		"stats": stats,
	}
	if len(issues) > 0 {
		details["issues"] = issues
	}

	return Result{
		Name:    "packet_capture",
		Status:  status,
		Details: details,
	}
}

// parsePacketCapture analyzes tcpdump output.
func parsePacketCapture(output string, iface string) *PacketCaptureStats {
	stats := &PacketCaptureStats{
		Interface: iface,
	}

	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex patterns for common issues
	retransPattern := regexp.MustCompile(`(?i)retrans|retransmit`)
	dupAckPattern := regexp.MustCompile(`(?i)dup\s*ack|duplicate\s*ack`)
	outOfOrderPattern := regexp.MustCompile(`(?i)out.of.order|ooo`)
	zeroWindowPattern := regexp.MustCompile(`(?i)win\s*0\s|zero.window`)
	windowFullPattern := regexp.MustCompile(`(?i)window.full`)
	keepAlivePattern := regexp.MustCompile(`(?i)keep.?alive`)
	checksumPattern := regexp.MustCompile(`(?i)bad\s*cksum|incorrect|checksum`)
	malformedPattern := regexp.MustCompile(`(?i)malformed|truncated|bogus`)

	for scanner.Scan() {
		line := scanner.Text()

		// Count packet
		if strings.Contains(line, "IP ") || strings.Contains(line, "IP6 ") {
			stats.PacketCount++
		}

		// TCP packets
		if strings.Contains(line, "Flags [") {
			stats.TCPPackets++

			// Analyze flags
			if strings.Contains(line, "Flags [S]") || strings.Contains(line, "Flags [S.]") {
				stats.TCPSynPackets++
			}
			if strings.Contains(line, "Flags [F]") || strings.Contains(line, "Flags [F.]") {
				stats.TCPFinPackets++
			}
			if strings.Contains(line, "Flags [R]") || strings.Contains(line, "Flags [R.]") {
				stats.TCPRstPackets++
			}
		}

		// ICMP packets
		if strings.Contains(line, "ICMP") {
			stats.ICMPPackets++
			if strings.Contains(line, "unreachable") {
				stats.ICMPUnreachable++
			}
			if strings.Contains(line, "time exceeded") {
				stats.ICMPTimeExceeded++
			}
			if strings.Contains(line, "redirect") {
				stats.ICMPRedirect++
			}
		}

		// Error conditions
		if retransPattern.MatchString(line) {
			stats.TCPRetransmits++
		}
		if dupAckPattern.MatchString(line) {
			stats.TCPDuplicateAcks++
		}
		if outOfOrderPattern.MatchString(line) {
			stats.TCPOutOfOrder++
		}
		if zeroWindowPattern.MatchString(line) {
			stats.TCPZeroWindow++
		}
		if windowFullPattern.MatchString(line) {
			stats.TCPWindowFull++
		}
		if keepAlivePattern.MatchString(line) {
			stats.TCPKeepAlive++
		}
		if checksumPattern.MatchString(line) {
			stats.ChecksumErrors++
		}
		if malformedPattern.MatchString(line) {
			stats.MalformedPackets++
		}

		// Fragmentation
		if strings.Contains(line, "frag ") {
			stats.FragmentedPackets++
		}
	}

	// Parse summary line if present
	// Example: "1000 packets captured"
	summaryPattern := regexp.MustCompile(`(\d+)\s+packets?\s+captured`)
	if match := summaryPattern.FindStringSubmatch(output); len(match) > 1 {
		if count, err := strconv.ParseUint(match[1], 10, 64); err == nil {
			stats.PacketCount = count
		}
	}

	return stats
}

// getDefaultInterface attempts to find the default network interface.
func getDefaultInterface() (string, error) {
	if runtime.GOOS == "linux" {
		// Read default route to find interface
		cmd := exec.Command("ip", "route", "show", "default")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		// Parse: "default via X.X.X.X dev eth0 ..."
		fields := strings.Fields(string(output))
		for i, f := range fields {
			if f == "dev" && i+1 < len(fields) {
				return fields[i+1], nil
			}
		}
	} else if runtime.GOOS == "darwin" {
		// macOS: use route command
		cmd := exec.Command("route", "-n", "get", "default")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		// Parse: "interface: en0"
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "interface:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "interface:")), nil
			}
		}
	}

	return "", fmt.Errorf("could not determine default interface")
}

// RunSocketStats collects per-socket TCP statistics using ss command.
// This provides connection-specific metrics like RTT, retransmits, etc.
func RunSocketStats(ctx context.Context, target string) Result {
	if runtime.GOOS != "linux" {
		return Result{
			Name:   "socket_stats",
			Status: StatusNA,
			Error:  "socket stats only supported on Linux",
		}
	}

	ssPath, err := exec.LookPath("ss")
	if err != nil {
		return Result{
			Name:   "socket_stats",
			Status: StatusNA,
			Error:  "ss command not available",
		}
	}

	// ss -ti: TCP with internal info
	args := []string{"-ti"}
	if target != "" {
		args = append(args, "dst", target)
	}

	cmd := exec.CommandContext(ctx, ssPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return Result{
			Name:   "socket_stats",
			Status: StatusFail,
			Error:  fmt.Sprintf("ss command failed: %v", err),
		}
	}

	stats := parseSocketStats(string(output))

	status := StatusOK
	issues := []string{}

	// Analyze socket stats
	for _, s := range stats {
		if s.Retransmits > 5 {
			status = "warn"
			issues = append(issues, fmt.Sprintf("socket %s->%s: %d retransmits", s.Local, s.Remote, s.Retransmits))
		}
		if s.RTTMs > 200 {
			issues = append(issues, fmt.Sprintf("socket %s->%s: high RTT %.1fms", s.Local, s.Remote, s.RTTMs))
		}
	}

	return Result{
		Name:   "socket_stats",
		Status: status,
		Details: map[string]interface{}{
			"sockets": stats,
			"issues":  issues,
		},
	}
}

// SocketInfo holds per-connection TCP statistics.
type SocketInfo struct {
	Local       string  `json:"local"`
	Remote      string  `json:"remote"`
	State       string  `json:"state"`
	RTTMs       float64 `json:"rtt_ms"`
	RTTVarMs    float64 `json:"rttvar_ms"`
	Retransmits int     `json:"retransmits"`
	SendQueue   int     `json:"send_queue"`
	RecvQueue   int     `json:"recv_queue"`
	CwndSegs    int     `json:"cwnd_segments"`
}

// parseSocketStats parses ss -ti output.
func parseSocketStats(output string) []SocketInfo {
	var sockets []SocketInfo

	lines := strings.Split(output, "\n")
	var currentSocket *SocketInfo

	rttPattern := regexp.MustCompile(`rtt:(\d+\.?\d*)/(\d+\.?\d*)`)
	retransPattern := regexp.MustCompile(`retrans:(\d+)/(\d+)`)
	cwndPattern := regexp.MustCompile(`cwnd:(\d+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Socket header line: "ESTAB  0  0  10.0.0.1:1234  10.0.0.2:80"
		if strings.HasPrefix(line, "ESTAB") || strings.HasPrefix(line, "SYN-") ||
			strings.HasPrefix(line, "FIN-") || strings.HasPrefix(line, "TIME-") ||
			strings.HasPrefix(line, "CLOSE") {
			if currentSocket != nil {
				sockets = append(sockets, *currentSocket)
			}

			fields := strings.Fields(line)
			if len(fields) >= 5 {
				sendQ, _ := strconv.Atoi(fields[1])
				recvQ, _ := strconv.Atoi(fields[2])
				currentSocket = &SocketInfo{
					State:     fields[0],
					SendQueue: sendQ,
					RecvQueue: recvQ,
					Local:     fields[3],
					Remote:    fields[4],
				}
			}
			continue
		}

		// Extended info line (indented)
		if currentSocket != nil && (strings.HasPrefix(line, "cubic") || strings.HasPrefix(line, "bbr") ||
			strings.Contains(line, "rtt:")) {
			// Parse RTT
			if match := rttPattern.FindStringSubmatch(line); len(match) > 2 {
				currentSocket.RTTMs, _ = strconv.ParseFloat(match[1], 64)
				currentSocket.RTTVarMs, _ = strconv.ParseFloat(match[2], 64)
			}

			// Parse retransmits
			if match := retransPattern.FindStringSubmatch(line); len(match) > 2 {
				currentSocket.Retransmits, _ = strconv.Atoi(match[2])
			}

			// Parse cwnd
			if match := cwndPattern.FindStringSubmatch(line); len(match) > 1 {
				currentSocket.CwndSegs, _ = strconv.Atoi(match[1])
			}
		}
	}

	if currentSocket != nil {
		sockets = append(sockets, *currentSocket)
	}

	return sockets
}
