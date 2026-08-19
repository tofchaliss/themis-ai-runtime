package model

type Request struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type Metrics struct {
	LoadMS  int64   `json:"load_ms"`
	EvalMS  int64   `json:"eval_ms"`
	TotalMS int64   `json:"total_ms"`
	Tokens  int     `json:"tokens"`
	TPS     float64 `json:"tokens_per_second"`
}

type Response struct {
	Benchmark string `json:"benchmark,omitempty"`
	Model     string `json:"model"`
	Runtime   string `json:"runtime"`

	Answer string `json:"answer"`

	Metrics Metrics `json:"metrics"`

	// Raw contains the original response from the runtime.
	// It is intentionally excluded from JSON serialization.
	Raw []byte `json:"-"`
}
