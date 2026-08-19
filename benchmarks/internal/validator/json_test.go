package validator

import (
	"encoding/json"
	"testing"
)

func jsonExpected(t *testing.T, ground string) *Expected {
	t.Helper()

	return &Expected{
		Benchmark: "B012",
		Validator: "json",
		Expected:  json.RawMessage(ground),
	}
}

func TestValidateJSON(t *testing.T) {

	ground := `{
		"cve": "CVE-2024-99999",
		"score": 9.8,
		"affected_versions": ["1.0.0 through 1.4.2"],
		"cvss": {"version": "3.1", "severity": "CRITICAL"}
	}`

	t.Run("exact match scores 100", func(t *testing.T) {
		answer := `{
			"cve": "CVE-2024-99999",
			"score": 9.8,
			"affected_versions": ["1.0.0 through 1.4.2"],
			"cvss": {"version": "3.1", "severity": "CRITICAL"}
		}`

		result, err := ValidateJSON(jsonExpected(t, ground), answer)
		if err != nil {
			t.Fatal(err)
		}

		if result.Score != 100 {
			t.Errorf("Score = %d, want 100; missing %v", result.Score, result.Missing)
		}
	})

	t.Run("answer wrapped in prose and code fences", func(t *testing.T) {
		answer := "Here is the extraction:\n```json\n" +
			`{"cve": "CVE-2024-99999", "score": 9.8,
			  "affected_versions": ["1.0.0 through 1.4.2"],
			  "cvss": {"version": "3.1", "severity": "CRITICAL"}}` +
			"\n```\nLet me know if you need more."

		result, err := ValidateJSON(jsonExpected(t, ground), answer)
		if err != nil {
			t.Fatal(err)
		}

		if result.Score != 100 {
			t.Errorf("Score = %d, want 100; missing %v", result.Score, result.Missing)
		}
	})

	t.Run("wrong and missing fields fail", func(t *testing.T) {
		answer := `{"cve": "CVE-2024-11111", "score": 9.8}`

		result, err := ValidateJSON(jsonExpected(t, ground), answer)
		if err != nil {
			t.Fatal(err)
		}

		if result.Passed != 1 || result.Failed != 3 {
			t.Errorf("Passed/Failed = %d/%d, want 1/3", result.Passed, result.Failed)
		}
		if result.Score != 25 {
			t.Errorf("Score = %d, want 25", result.Score)
		}
	})

	t.Run("non-JSON answer fails every field", func(t *testing.T) {
		result, err := ValidateJSON(jsonExpected(t, ground), "I cannot help with that.")
		if err != nil {
			t.Fatal(err)
		}

		if result.Score != 0 || result.Failed != 4 {
			t.Errorf("Score/Failed = %d/%d, want 0/4", result.Score, result.Failed)
		}
		if len(result.Violations) == 0 {
			t.Error("expected a violation explaining the invalid JSON")
		}
	})

	t.Run("string comparison trims whitespace", func(t *testing.T) {
		result, err := ValidateJSON(
			jsonExpected(t, `{"cve": "CVE-2024-99999"}`),
			`{"cve": "  CVE-2024-99999  "}`,
		)
		if err != nil {
			t.Fatal(err)
		}

		if result.Score != 100 {
			t.Errorf("Score = %d, want 100", result.Score)
		}
	})

	t.Run("invalid ground truth is an error", func(t *testing.T) {
		if _, err := ValidateJSON(jsonExpected(t, `not json`), "{}"); err == nil {
			t.Error("expected error for invalid ground truth")
		}
	})

	t.Run("empty ground truth is an error", func(t *testing.T) {
		if _, err := ValidateJSON(jsonExpected(t, `{}`), "{}"); err == nil {
			t.Error("expected error for empty ground truth")
		}
	})
}
