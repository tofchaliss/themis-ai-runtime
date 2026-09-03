package report

type Report struct {
	Model string
	Date  string

	Benchmarks int

	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int

	AverageTPS          float64
	AverageGenerationMS float64
	AverageTotalMS      float64

	// Validation summary. AverageScore covers validated benchmarks only.
	Validated    int
	AverageScore int
	TotalPassed  int
	TotalFailed  int

	Results []Benchmark
}

type Benchmark struct {
	Benchmark string

	// Validation. Validated is false when no validation result exists
	// for this benchmark (e.g. `validate` was not run).
	Validated  bool
	Score      int
	Passed     int
	Failed     int
	Violations int

	PromptTokens     int
	CompletionTokens int
	TotalTokens      int

	TPS float64

	LoadMS       float64
	PromptMS     float64
	GenerationMS float64
	TotalMS      float64
}
