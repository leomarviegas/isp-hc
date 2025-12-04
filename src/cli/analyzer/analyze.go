package analyzer

import (
	"fmt"

	"isp-checker/probes"
)

// Analyze converts probe results into a simple score and diagnosis.
func Analyze(results []probes.Result) (float64, string, []string) {
	if len(results) == 0 {
		return 0, "no probes executed", []string{"no data"}
	}

	successes := 0
	diag := []string{}
	for _, p := range results {
		if p.Status == probes.StatusOK {
			successes++
			continue
		}
		message := p.Error
		if message == "" {
			message = "unknown error"
		}
		diag = append(diag, fmt.Sprintf("%s: %s", p.Name, message))
	}

	score := float64(successes) / float64(len(results)) * 100
	summary := fmt.Sprintf("%d/%d probes succeeded", successes, len(results))
	if len(diag) == 0 {
		diag = []string{"all probes succeeded"}
	}

	return score, summary, diag
}
