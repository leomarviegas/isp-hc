package analyzer

import "fmt"

// Analyze processes the probe results and generates a score and diagnosis.
func Analyze(results map[string]interface{}) (float64, []string) {
	fmt.Println("Analyzing probe results...")
	// Placeholder for analysis logic
	score := 0.0
	diagnosis := []string{}

	pingResult, ok := results["ping"].(map[string]interface{})
	if ok {
		if loss, ok := pingResult["loss_percent"].(float64); ok && loss > 10 {
			score += 50
			diagnosis = append(diagnosis, "High packet loss detected.")
		}
	}

	return score, diagnosis
}
