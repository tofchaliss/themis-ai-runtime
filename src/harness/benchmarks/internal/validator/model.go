package validator

import "encoding/json"

// Expected represents the expected validation rules for a benchmark.
type Expected struct {
	Benchmark string `json:"benchmark"`
	Validator string `json:"validator"`

	// Keyword and regex validator fields. For the keyword validator
	// these are case-insensitive substrings; for the regex validator
	// they are Go regular expressions.
	Required  []string `json:"required,omitempty"`
	Forbidden []string `json:"forbidden,omitempty"`

	// JSON validator ground truth: an object whose top-level fields
	// must appear (deep-equal) in the model's JSON answer.
	Expected json.RawMessage `json:"expected,omitempty"`

	// Options relaxes JSON comparison. The zero value is fully strict.
	Options JSONOptions `json:"options,omitempty"`
}

// JSONOptions configures value comparison for the json validator.
type JSONOptions struct {
	// CoerceNumbers accepts a numeric string where a number is
	// expected (e.g. "9.8" matches 9.8).
	CoerceNumbers bool `json:"coerce_numbers,omitempty"`

	// NumberTolerance accepts numbers within this absolute distance
	// of the expected value.
	NumberTolerance float64 `json:"number_tolerance,omitempty"`

	// CaseInsensitive compares strings ignoring case (both sides are
	// always compared with surrounding whitespace trimmed).
	CaseInsensitive bool `json:"case_insensitive,omitempty"`
}

// Result represents the validation outcome for a benchmark.
type Result struct {
	Benchmark string `json:"benchmark"`

	Score int `json:"score"`

	Passed int `json:"passed"`
	Failed int `json:"failed"`

	Missing    []string `json:"missing"`
	Violations []string `json:"violations"`
}

// score computes a benchmark score as passed checks over the total of
// required checks plus violations: forbidden content an answer emits
// counts against it exactly like a failed required check (owner
// decision F3, 2026-09-04). Zero total yields zero.
func score(passed, required, violations int) int {
	total := required + violations
	if total == 0 {
		return 0
	}
	return (passed * 100) / total
}
