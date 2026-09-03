package benchmark

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tofchaliss/themis/internal/llm"
)

// WriteRun writes the run envelope for one benchmark to
// runs/<date>/<model>/<benchmark>.json.
func WriteRun(
	baseDir string,
	date string,
	modelName string,
	record llm.RunRecord,
) error {

	dir := filepath.Join(
		baseDir,
		"runs",
		date,
		modelName,
	)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(dir, record.Benchmark+".json"),
		data,
		0644,
	)
}
