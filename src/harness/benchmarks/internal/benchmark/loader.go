package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LoadDefinitions loads every *.json benchmark definition in dir and
// validates that each one is usable by the run pipeline.
func LoadDefinitions(dir string) ([]Definition, error) {

	pattern := filepath.Join(dir, "*.json")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	definitions := make([]Definition, 0, len(files))
	seen := make(map[string]string, len(files))

	for _, file := range files {

		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var d Definition

		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}

		if d.ID == "" {
			return nil, fmt.Errorf("%s: missing required field: id", file)
		}

		if d.Prompt == "" {
			return nil, fmt.Errorf("%s: missing required field: prompt", file)
		}

		if previous, ok := seen[d.ID]; ok {
			return nil, fmt.Errorf(
				"%s: duplicate benchmark id %q (also defined in %s)",
				file,
				d.ID,
				previous,
			)
		}
		seen[d.ID] = file

		definitions = append(definitions, d)
	}

	return definitions, nil
}
