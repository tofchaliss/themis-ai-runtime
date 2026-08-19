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

// EvaluateRun normalizes a single raw run file.
func EvaluateRun(
	benchmark string,
	filename string,
) (*Result, error) {

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

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

	resp := &model.Response{
		Runtime: "ollama",
		Model:   raw.Model,
		Answer:  raw.Response,
		Raw:     data,
	}

	return Normalize(benchmark, resp)
}

// EvaluateAll normalizes every raw run for the given date and model.
// It keeps going when a single run fails and reports all failures at
// the end.
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
