package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tofchaliss/themis/benchmarks/internal/evaluator"
	"github.com/tofchaliss/themis/benchmarks/internal/validator"
)

// LoadResponses loads evaluated responses together with their validation
// results. A missing validation result leaves the benchmark unvalidated;
// a malformed one is an error.
func LoadResponses(root, date, model string) ([]Benchmark, error) {

	responseDir := filepath.Join(
		root,
		"responses",
		date,
		model,
	)

	files, err := filepath.Glob(filepath.Join(responseDir, "*.json"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no responses found in %s", responseDir)
	}

	sort.Strings(files)

	benchmarks := make([]Benchmark, 0, len(files))

	for _, file := range files {

		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var response evaluator.Result

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("parse response file %s: %w", file, err)
		}

		benchmark := Benchmark{
			Benchmark: response.Benchmark,

			PromptTokens:     response.Metrics.PromptTokens,
			CompletionTokens: response.Metrics.CompletionTokens,
			TotalTokens:      response.Metrics.TotalTokens,

			TPS: response.Metrics.TokensPerSecond,

			LoadMS:       response.Metrics.LoadTimeMS,
			PromptMS:     response.Metrics.PromptTimeMS,
			GenerationMS: response.Metrics.GenerationTimeMS,
			TotalMS:      response.Metrics.TotalTimeMS,
		}

		validationFile := filepath.Join(
			root,
			"validation",
			date,
			model,
			response.Benchmark+".json",
		)

		validationData, err := os.ReadFile(validationFile)

		switch {

		case err == nil:
			var validation validator.Result

			if err := json.Unmarshal(validationData, &validation); err != nil {
				return nil, fmt.Errorf(
					"parse validation file %s: %w",
					validationFile,
					err,
				)
			}

			benchmark.Validated = true
			benchmark.Score = validation.Score
			benchmark.Passed = validation.Passed
			benchmark.Failed = validation.Failed
			benchmark.Violations = len(validation.Violations)

		case errors.Is(err, os.ErrNotExist):
			// Not validated yet; reported as such.

		default:
			return nil, err
		}

		benchmarks = append(benchmarks, benchmark)
	}

	return benchmarks, nil
}
