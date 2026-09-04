package validator

import (
	"strings"
)

// ValidateKeyword validates a model response using simple keyword matching.
func ValidateKeyword(expected *Expected, answer string) Result {

	result := Result{
		Benchmark:  expected.Benchmark,
		Missing:    []string{},
		Violations: []string{},
	}

	normalized := strings.ToLower(answer)

	// Check required keywords
	for _, keyword := range expected.Required {

		if strings.Contains(normalized, strings.ToLower(keyword)) {
			result.Passed++
		} else {
			result.Failed++
			result.Missing = append(result.Missing, keyword)
		}
	}

	// Check forbidden keywords
	for _, keyword := range expected.Forbidden {

		if strings.Contains(normalized, strings.ToLower(keyword)) {
			result.Violations = append(result.Violations, keyword)
		}
	}

	result.Score = score(result.Passed, len(expected.Required), len(result.Violations))

	return result
}
