package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tofchaliss/themis/benchmarks/internal/evaluator"
)

// ValidateAll validates all evaluated benchmark responses. It keeps going
// when a single benchmark fails and reports all failures at the end.
func ValidateAll(root, date, model string) error {

	files, err := FindResponses(root, date, model)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf(
			"no responses found in %s",
			filepath.Join(root, "responses", date, model),
		)
	}

	failed := 0

	for _, file := range files {

		if err := validateFile(root, date, model, file); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", filepath.Base(file), err)
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf(
			"%d of %d responses failed to validate",
			failed,
			len(files),
		)
	}

	return nil
}

func validateFile(root, date, model, file string) error {

	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	var response evaluator.Result

	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("parse response file %s: %w", file, err)
	}

	if response.Benchmark == "" {
		return fmt.Errorf("response file %s: missing benchmark id", file)
	}

	expected, err := LoadExpected(
		root,
		response.Benchmark+".json",
	)
	if err != nil {
		return err
	}

	var result Result

	switch expected.Validator {

	case "keyword":
		result = ValidateKeyword(
			expected,
			response.Answer,
		)

	case "json":
		result, err = ValidateJSON(
			expected,
			response.Answer,
		)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf(
			"%s: unsupported validator: %q",
			response.Benchmark,
			expected.Validator,
		)
	}

	if err := Write(
		root,
		date,
		model,
		result,
	); err != nil {
		return err
	}

	fmt.Printf(
		"✓ %s validated (score %d%%)\n",
		response.Benchmark,
		result.Score,
	)

	return nil
}
