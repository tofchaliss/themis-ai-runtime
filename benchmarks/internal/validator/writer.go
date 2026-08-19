package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write writes a validation result to:
//
// validation/<date>/<model>/<benchmark>.json
func Write(root, date, model string, result Result) error {

	dir := filepath.Join(
		root,
		"validation",
		date,
		model,
	)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create validation directory: %w", err)
	}

	path := filepath.Join(
		dir,
		result.Benchmark+".json",
	)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validation result: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write validation result: %w", err)
	}

	return nil
}
