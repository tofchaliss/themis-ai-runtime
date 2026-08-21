package evaluator

import "github.com/tofchaliss/themis/benchmarks/internal/model"

// Metrics is the normalized execution metrics of a run.
type Metrics = model.Metrics

// Result is the normalized benchmark result.
type Result struct {
	Benchmark string `json:"benchmark"`
	Model     string `json:"model"`
	Runtime   string `json:"runtime"`

	Answer string `json:"answer"`

	Metrics Metrics `json:"metrics"`
}
