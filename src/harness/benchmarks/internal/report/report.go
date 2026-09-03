package report

import (
	"os"
	"path/filepath"
)

// Write generates and writes the Markdown report to
// reports/<date>/<model>.md. Model names may contain slashes
// (e.g. "org/model"), which become nested directories.
func Write(root, date string, report Report) error {

	outputFile := filepath.Join(
		root,
		"reports",
		date,
		report.Model+".md",
	)

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}

	return os.WriteFile(
		outputFile,
		[]byte(Markdown(report)),
		0644,
	)
}
