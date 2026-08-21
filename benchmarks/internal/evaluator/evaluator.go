package evaluator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
	"github.com/tofchaliss/themis/benchmarks/internal/runtime"
)

// EvaluateRun normalizes a single run file. Run files are runtime-
// agnostic envelopes (model.RunRecord); raw Ollama payloads written by
// older versions of the tool are still accepted.
func EvaluateRun(
	benchmark string,
	filename string,
) (*Result, error) {

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var record model.RunRecord

	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse run file %s: %w", filename, err)
	}

	if record.Runtime != "" && record.Answer != "" {
		return &Result{
			Benchmark: benchmark,
			Model:     record.Model,
			Runtime:   record.Runtime,
			Answer:    record.Answer,
			Metrics:   Metrics(record.Metrics),
		}, nil
	}

	return evaluateLegacyRun(benchmark, filename, data)
}

// evaluateLegacyRun handles pre-envelope run files, which are raw
// Ollama /api/generate payloads.
func evaluateLegacyRun(
	benchmark string,
	filename string,
	data []byte,
) (*Result, error) {

	var raw runtime.OllamaResponse

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse run file %s: %w", filename, err)
	}

	if raw.Error != "" {
		return nil, fmt.Errorf(
			"run file %s contains a runtime error: %s",
			filename,
			raw.Error,
		)
	}

	if raw.Model == "" {
		return nil, fmt.Errorf("invalid run file %s: missing model", filename)
	}

	if !raw.Done {
		return nil, fmt.Errorf(
			"%s: run is incomplete (done=false); re-run the benchmark",
			benchmark,
		)
	}

	return &Result{
		Benchmark: benchmark,
		Model:     raw.Model,
		Runtime:   "ollama",
		Answer:    raw.Response,
		Metrics:   Metrics(raw.Metrics()),
	}, nil
}

// EvaluateAll normalizes every run for the given date and model. It
// keeps going when a single run fails and reports all failures at the
// end.
func EvaluateAll(
	root string,
	date string,
	model string,
) error {

	files, err := FindRuns(root, date, model)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf(
			"no runs found in %s",
			filepath.Join(root, "runs", date, model),
		)
	}

	failed := 0

	for _, file := range files {

		benchmark := strings.TrimSuffix(
			filepath.Base(file),
			filepath.Ext(file),
		)

		result, err := EvaluateRun(benchmark, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", benchmark, err)
			failed++
			continue
		}

		if err := WriteResult(
			root,
			date,
			model,
			benchmark,
			result,
		); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", benchmark, err)
			failed++
			continue
		}

		fmt.Printf("✓ %s evaluated\n", benchmark)
	}

	if failed > 0 {
		return fmt.Errorf(
			"%d of %d runs failed to evaluate",
			failed,
			len(files),
		)
	}

	return nil
}
