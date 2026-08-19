package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func WriteRawRun(
	baseDir string,
	date string,
	model string,
	id string,
	raw []byte,
) error {

	dir := filepath.Join(
		baseDir,
		"runs",
		date,
		model,
	)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var pretty map[string]any

	if json.Unmarshal(raw, &pretty) == nil {

		raw, _ = json.MarshalIndent(
			pretty,
			"",
			"  ",
		)
	}

	filename := filepath.Join(
		dir,
		id+".json",
	)

	return os.WriteFile(
		filename,
		raw,
		0644,
	)
}
