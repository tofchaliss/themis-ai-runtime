package llm

// Options are the generation parameters sent to the runtime. Benchmarks
// pin these for reproducibility.
type Options struct {
	Temperature float64 `json:"temperature"`
	Seed        int     `json:"seed"`
}

// DefaultOptions are the deterministic defaults used for benchmarking.
func DefaultOptions() Options {
	return Options{
		Temperature: 0,
		Seed:        42,
	}
}

type Request struct {
	Model   string  `json:"model"`
	Prompt  string  `json:"prompt"`
	Options Options `json:"options"`
}

// Metrics are the normalized execution metrics a runtime reports.
// Runtimes fill in what they can measure; fields they cannot observe
// stay zero.
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
