package validator

import (
	"reflect"
	"testing"
)

func TestValidateRegex(t *testing.T) {

	expected := &Expected{
		Benchmark: "B014",
		Validator: "regex",
		Required:  []string{"P-301", "(?i)security feature x", "rotat|revok"},
		Forbidden: []string{"P-304"},
	}

	t.Run("all patterns match", func(t *testing.T) {
		result, err := ValidateRegex(
			expected,
			"P-301 hinges on Security Feature X; rotate the key.",
		)
		if err != nil {
			t.Fatal(err)
		}

		if result.Score != 100 || result.Failed != 0 {
			t.Errorf("Score/Failed = %d/%d, want 100/0; missing %v",
				result.Score, result.Failed, result.Missing)
		}
	})

	t.Run("alternation matches either phrasing", func(t *testing.T) {
		result, err := ValidateRegex(
			expected,
			"P-301, security feature x, revoke the credentials.",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.Score != 100 {
			t.Errorf("Score = %d, want 100; missing %v", result.Score, result.Missing)
		}
	})

	t.Run("missing patterns are reported", func(t *testing.T) {
		result, err := ValidateRegex(expected, "P-301 only.")
		if err != nil {
			t.Fatal(err)
		}

		if result.Passed != 1 || result.Failed != 2 {
			t.Errorf("Passed/Failed = %d/%d, want 1/2", result.Passed, result.Failed)
		}
		want := []string{"(?i)security feature x", "rotat|revok"}
		if !reflect.DeepEqual(result.Missing, want) {
			t.Errorf("Missing = %v, want %v", result.Missing, want)
		}
	})

	t.Run("forbidden pattern is a violation", func(t *testing.T) {
		result, err := ValidateRegex(
			expected,
			"P-301 P-304 security feature x rotate",
		)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result.Violations, []string{"P-304"}) {
			t.Errorf("Violations = %v", result.Violations)
		}
	})

	t.Run("invalid pattern is an error", func(t *testing.T) {
		bad := &Expected{Benchmark: "X", Required: []string{"("}}

		if _, err := ValidateRegex(bad, "anything"); err == nil {
			t.Error("expected error for invalid pattern")
		}
	})
}
