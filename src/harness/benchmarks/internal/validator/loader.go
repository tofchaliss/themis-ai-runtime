package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadExpected loads an expected validation definition from
// benchmarks/expected/<filename>.
func LoadExpected(root, filename string) (*Expected, error) {

	path := filepath.Join(
		root,
		"expected",
		filename,
	)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read expected file %s: %w", path, err)
	}

	var expected Expected

	if err := json.Unmarshal(data, &expected); err != nil {
		return nil, fmt.Errorf("parse expected file %s: %w", path, err)
	}

	return &expected, nil
}
