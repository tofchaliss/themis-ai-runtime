package model

import "encoding/json"

// RunRecord is the runtime-agnostic envelope written to
// runs/<date>/<model>/<benchmark>.json. It carries everything the
// evaluate stage needs, independent of which runtime produced it, plus
// the original provider response for auditability.
type RunRecord struct {
	Benchmark string `json:"benchmark"`
	Model     string `json:"model"`
	Runtime   string `json:"runtime"`

	Answer string `json:"answer"`

	// Options records the generation parameters used, for
	// reproducibility.
	Options Options `json:"options"`

	Metrics Metrics `json:"metrics"`

	// Raw is the unmodified response from the runtime.
	Raw json.RawMessage `json:"raw"`
}
