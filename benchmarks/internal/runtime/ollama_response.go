package runtime

import "github.com/tofchaliss/themis/benchmarks/internal/model"

// OllamaResponse is the subset of the /api/generate response we consume.
type OllamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`

	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`

	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	EvalDuration       int64 `json:"eval_duration"`
	TotalDuration      int64 `json:"total_duration"`

	Done bool `json:"done"`

	// Error is set by Ollama when the request fails (e.g. unknown model).
	Error string `json:"error"`
}

// Metrics converts Ollama's nanosecond counters into normalized metrics.
func (r OllamaResponse) Metrics() model.Metrics {

	const nsToMS = 1_000_000.0

	m := model.Metrics{
		PromptTokens:     r.PromptEvalCount,
		CompletionTokens: r.EvalCount,
		TotalTokens:      r.PromptEvalCount + r.EvalCount,

		LoadTimeMS:       float64(r.LoadDuration) / nsToMS,
		PromptTimeMS:     float64(r.PromptEvalDuration) / nsToMS,
		GenerationTimeMS: float64(r.EvalDuration) / nsToMS,
		TotalTimeMS:      float64(r.TotalDuration) / nsToMS,
	}

	if m.GenerationTimeMS > 0 {
		m.TokensPerSecond =
			float64(m.CompletionTokens) / (m.GenerationTimeMS / 1000)
	}

	return m
}
