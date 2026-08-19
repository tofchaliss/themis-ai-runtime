package evaluator

import (
	"path/filepath"
	"sort"
)

func FindRuns(root, date, model string) ([]string, error) {

	dir := filepath.Join(root, "runs", date, model)

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	return files, nil
}
