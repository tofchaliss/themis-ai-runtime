package validator

import (
	"path/filepath"
	"sort"
)

func FindResponses(root, date, model string) ([]string, error) {

	dir := filepath.Join(root, "responses", date, model)

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	return files, nil
}
