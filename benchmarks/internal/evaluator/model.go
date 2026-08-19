package evaluator

// Metrics contains the normalized execution metrics.
type Metrics struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	LoadTimeMS       float64 `json:"load_time_ms"`
	PromptTimeMS     float64 `json:"prompt_time_ms"`
	GenerationTimeMS float64 `json:"generation_time_ms"`
	TotalTimeMS      float64 `json:"total_time_ms"`

	TokensPerSecond float64 `json:"tokens_per_second"`
}

// Result is the normalized benchmark result.
type Result struct {
	Benchmark string `json:"benchmark"`
	Model     string `json:"model"`
	Runtime   string `json:"runtime"`

	Answer string `json:"answer"`

	Metrics Metrics `json:"metrics"`
}
