package runtime

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
