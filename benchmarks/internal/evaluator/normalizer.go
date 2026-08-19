package evaluator

import (
	"encoding/json"
	"fmt"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
	"github.com/tofchaliss/themis/benchmarks/internal/runtime"
)

// Normalize converts a raw runtime response into a normalized Result.
func Normalize(
	benchmark string,
	resp *model.Response,
) (*Result, error) {

	var raw runtime.OllamaResponse

	if err := json.Unmarshal(resp.Raw, &raw); err != nil {
		return nil, fmt.Errorf("%s: parse raw run: %w", benchmark, err)
	}

	if raw.Error != "" {
		return nil, fmt.Errorf(
			"%s: run contains a runtime error: %s",
			benchmark,
			raw.Error,
		)
	}

	if !raw.Done {
		return nil, fmt.Errorf(
			"%s: run is incomplete (done=false); re-run the benchmark",
			benchmark,
		)
	}

	const nsToMS = 1_000_000.0

	r := &Result{
		Benchmark: benchmark,
		Model:     resp.Model,
		Runtime:   resp.Runtime,
		Answer:    resp.Answer,
	}

	r.Metrics.PromptTokens = raw.PromptEvalCount
	r.Metrics.CompletionTokens = raw.EvalCount
	r.Metrics.TotalTokens = raw.PromptEvalCount + raw.EvalCount

	r.Metrics.LoadTimeMS = float64(raw.LoadDuration) / nsToMS
	r.Metrics.PromptTimeMS = float64(raw.PromptEvalDuration) / nsToMS
	r.Metrics.GenerationTimeMS = float64(raw.EvalDuration) / nsToMS
	r.Metrics.TotalTimeMS = float64(raw.TotalDuration) / nsToMS

	if r.Metrics.GenerationTimeMS > 0 {
		r.Metrics.TokensPerSecond =
			float64(r.Metrics.CompletionTokens) /
				(r.Metrics.GenerationTimeMS / 1000)
	}

	return r, nil
}
