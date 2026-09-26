package deployment

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D-R-2 at the anchor door: the governed anchors registry (version 2)
// loads only with every reliance decision bound; tampering refuses.
// The observed copy (parseRegistry, bytes only) is compared by
// identity and is not bound here.
func TestAnchorsRegistryV2RequiresBoundDecisionRecords(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	for _, name := range []string{"anchors.json", "anchors.proposed.json"} {
		r, err := LoadRegistry(filepath.Join(repo, "policies", "deployment", name))
		if err != nil || r.Version != 2 {
			t.Fatalf("%s must load at version 2: %v", name, err)
		}
	}
	base := t.TempDir()
	for _, sub := range []string{"deployment", "decisions"} {
		_ = os.MkdirAll(filepath.Join(base, sub), 0o755)
		ents, _ := os.ReadDir(filepath.Join(repo, "policies", sub))
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			b, _ := os.ReadFile(filepath.Join(repo, "policies", sub, e.Name()))
			_ = os.WriteFile(filepath.Join(base, sub, e.Name()), b, 0o644)
		}
	}
	reg := filepath.Join(base, "deployment", "anchors.json")
	if _, err := LoadRegistry(reg); err != nil {
		t.Fatal(err)
	}
	rec := filepath.Join(base, "decisions", "rel-anchor-rsys-5.json")
	b, _ := os.ReadFile(rec)
	_ = os.WriteFile(rec, []byte(strings.Replace(string(b), `"version": 5`, `"version": 6`, 1)), 0o644)
	if _, err := LoadRegistry(reg); !errors.Is(err, ErrAdmission) || !strings.Contains(err.Error(), "rsys@5") {
		t.Fatalf("retargeted reliance record: %v", err)
	}
	// The bytes-only parser still admits a version-1 observed copy.
	if _, err := parseRegistry([]byte(`{"version":1,"kind":"deployment-anchors","entries":[{"name":"rsys","version":5,"artifact_sha256":"` + strings.Repeat("a", 64) + `","state":"active","steward":"t"}]}`)); err != nil {
		t.Fatalf("observed v1 copy: %v", err)
	}
}
