package validator

import (
	"reflect"
	"testing"
)

func TestValidateKeyword(t *testing.T) {

	expected := &Expected{
		Benchmark: "B001",
		Validator: "keyword",
		Required:  []string{"Log4j", "CVE-2021-44228", "JNDI"},
		Forbidden: []string{"SQL Injection"},
	}

	t.Run("all required present", func(t *testing.T) {
		result := ValidateKeyword(
			expected,
			"log4j is affected by cve-2021-44228 via jndi lookups",
		)

		if result.Score != 100 {
			t.Errorf("Score = %d, want 100", result.Score)
		}
		if result.Passed != 3 || result.Failed != 0 {
			t.Errorf("Passed/Failed = %d/%d, want 3/0", result.Passed, result.Failed)
		}
		if len(result.Missing) != 0 || len(result.Violations) != 0 {
			t.Errorf("Missing/Violations = %v/%v, want empty", result.Missing, result.Violations)
		}
	})

	t.Run("missing keywords lower the score", func(t *testing.T) {
		result := ValidateKeyword(expected, "log4j is vulnerable")

		if result.Score != 33 {
			t.Errorf("Score = %d, want 33", result.Score)
		}
		if !reflect.DeepEqual(result.Missing, []string{"CVE-2021-44228", "JNDI"}) {
			t.Errorf("Missing = %v", result.Missing)
		}
	})

	t.Run("forbidden keywords are recorded", func(t *testing.T) {
		result := ValidateKeyword(
			expected,
			"log4j cve-2021-44228 jndi, similar to sql injection",
		)

		if !reflect.DeepEqual(result.Violations, []string{"SQL Injection"}) {
			t.Errorf("Violations = %v", result.Violations)
		}
		// F3: violations penalize the score — 3 passed of 3 required
		// plus 1 violation = 300/4.
		if result.Score != 75 {
			t.Errorf("Score = %d, want 75 (violation penalty)", result.Score)
		}
	})

	t.Run("matching is case-insensitive", func(t *testing.T) {
		result := ValidateKeyword(
			expected,
			"LOG4J CVE-2021-44228 JNDI",
		)

		if result.Score != 100 {
			t.Errorf("Score = %d, want 100", result.Score)
		}
	})

	t.Run("no required keywords yields zero score", func(t *testing.T) {
		result := ValidateKeyword(
			&Expected{Benchmark: "X"},
			"anything",
		)

		if result.Score != 0 || result.Passed != 0 {
			t.Errorf("Score/Passed = %d/%d, want 0/0", result.Score, result.Passed)
		}
	})
}
