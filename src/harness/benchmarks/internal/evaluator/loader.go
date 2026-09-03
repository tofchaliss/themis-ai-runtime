package evaluator

import (
	"path/filepath"
	"sort"
)

// FindRuns lists the run files for one date and model, excluding the
// run manifest.
func FindRuns(root, date, model string) ([]string, error) {

	dir := filepath.Join(root, "runs", date, model)

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	runs := files[:0]
	for _, f := range files {
		if filepath.Base(f) != "manifest.json" {
			runs = append(runs, f)
		}
	}

	sort.Strings(runs)

	return runs, nil
}
