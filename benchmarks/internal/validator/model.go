package validator

import "encoding/json"

// Expected represents the expected validation rules for a benchmark.
type Expected struct {
	Benchmark string `json:"benchmark"`
	Validator string `json:"validator"`

	// Keyword validator fields.
	Required  []string `json:"required,omitempty"`
	Forbidden []string `json:"forbidden,omitempty"`

	// JSON validator ground truth: an object whose top-level fields
	// must appear (deep-equal) in the model's JSON answer.
	Expected json.RawMessage `json:"expected,omitempty"`
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
