package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Standard JSON result schema
type Result struct {
	RunID     string        `json:"run_id"`
	Timestamp string        `json:"timestamp"`
	Target    string        `json/:"target"`
	Mode      string        `json:"mode"`
	Score     float64       `json:"score"`
	Summary   string        `json:"summary"`
	Probes    []interface{} `json:"probes"`
	Diagnosis []interface{} `json:"diagnosis"`
	Raw       interface{}   `json:"raw"`
}

// Write saves the result to a file.
func Write(target, mode string, score float64, summary string, probes, diagnosis []interface{}) error {
	result := Result{
		RunID:     "placeholder-uuid", // Will be replaced with a real UUID
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Target:    target,
		Mode:      mode,
		Score:     score,
		Summary:   summary,
		Probes:    probes,
		Diagnosis: diagnosis,
		Raw:       make(map[string]interface{}),
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(jsonData))
	// In a real implementation, this would write to the file path from the --output flag
	return os.WriteFile("report.json", jsonData, 0644)
}
