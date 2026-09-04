package benchmark

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tofchaliss/themis/internal/llm"
)

// ManifestFile is written into each run directory. Its name is reserved:
// the evaluate stage must skip it when globbing run files.
const ManifestFile = "manifest.json"

// Manifest records everything needed to attribute and reproduce a run:
// which model on which runtime, with which generation options, against
// which exact prompts and definitions.
type Manifest struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	Runtime string `json:"runtime"`
	Variant string `json:"variant,omitempty"`
	Date    string `json:"date"`

	StartedAt time.Time `json:"started_at"`

	Options llm.Options `json:"options"`

	// Prompts, Definitions, and Expected map benchmark IDs to the
	// SHA-256 of the file contents used for this run. Expected covers
	// the grading criteria: a benchmark's identity is prompt +
	// definition + expected spec + the options above, so a change to
	// how answers are scored is as visible as a change to what was
	// asked.
	Prompts     map[string]string `json:"prompts"`
	Definitions map[string]string `json:"definitions"`
	Expected    map[string]string `json:"expected"`
}

// WriteManifest writes the manifest into runs/<date>/<name>/.
func WriteManifest(baseDir, date, name string, m Manifest) error {

	dir := filepath.Join(baseDir, "runs", date, name)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(
		filepath.Join(dir, ManifestFile),
		data,
		0644,
	)
}

func hashBytes(b []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(b))
}
