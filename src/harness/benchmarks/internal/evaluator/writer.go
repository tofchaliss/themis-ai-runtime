package evaluator

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func WriteResult(
	root string,
	date string,
	model string,
	benchmark string,
	result *Result,
) error {

	dir := filepath.Join(
		root,
		"responses",
		date,
		model,
	)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(result, "", "  ")

	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(dir, benchmark+".json"),
		data,
		0644,
	)
}
