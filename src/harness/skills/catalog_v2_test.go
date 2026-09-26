package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D-R-2 at the catalog door: a version-2 catalog loads only when every
// entry's decision record exists, hashes to the pinned value, and
// targets exactly that entry. The real governed tree is the positive
// twin; each tampering refuses, named.
func TestCatalogV2RequiresBoundDecisionRecords(t *testing.T) {
	repo := filepath.Join("..", "..", "..")
	real := filepath.Join(repo, "policies", "skills", "catalog.json")
	if c, err := LoadCatalog(real); err != nil || c.Version != 2 {
		t.Fatalf("the governed catalog must load at version 2: %v", err)
	}
	copyTree := func(t *testing.T) (string, string) {
		t.Helper()
		base := t.TempDir()
		for _, sub := range []string{"skills", "decisions"} {
			src := filepath.Join(repo, "policies", sub)
			ents, _ := os.ReadDir(src)
			for _, e := range ents {
				if e.IsDir() {
					continue
				}
				b, _ := os.ReadFile(filepath.Join(src, e.Name()))
				if err := os.MkdirAll(filepath.Join(base, sub), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(base, sub, e.Name()), b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
		return filepath.Join(base, "skills", "catalog.json"), filepath.Join(base, "decisions")
	}
	cat, ddir := copyTree(t)
	if _, err := LoadCatalog(cat); err != nil {
		t.Fatalf("copied tree: %v", err)
	}
	// Tamper the record's rationale: bytes change, pinned hash no longer matches.
	rec := filepath.Join(ddir, "reg-skill-remediate-dependency-4.json")
	b, _ := os.ReadFile(rec)
	_ = os.WriteFile(rec, []byte(strings.Replace(string(b), "Initial registration", "Initial registration (edited)", 1)), 0o644)
	if _, err := LoadCatalog(cat); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "record bytes hash") {
		t.Fatalf("tampered record: %v", err)
	}
	// Remove the record: refuses.
	_ = os.Remove(rec)
	if _, err := LoadCatalog(cat); !errors.Is(err, ErrCatalog) {
		t.Fatalf("missing record: %v", err)
	}
	// Retarget: a record for another entry cited here refuses by target.
	cat2, ddir2 := copyTree(t)
	cb, _ := os.ReadFile(cat2)
	swapped := strings.Replace(string(cb), `"decision_ref": "reg-skill-remediate-dependency-4"`, `"decision_ref": "reg-skill-remediate-dependency-3"`, 1)
	_ = os.WriteFile(cat2, []byte(swapped), 0o644)
	_ = ddir2
	if _, err := LoadCatalog(cat2); !errors.Is(err, ErrCatalog) {
		t.Fatalf("retargeted record: %v", err)
	}
	// A version-2 entry without provenance refuses; a version-1 catalog
	// carrying decision fields refuses (provenance is not optional
	// decoration on an old schema).
	cat3, _ := copyTree(t)
	cb3, _ := os.ReadFile(cat3)
	noRef := strings.Replace(string(cb3), `"decision_ref": "reg-skill-remediate-dependency-4",`, ``, 1)
	_ = os.WriteFile(cat3, []byte(noRef), 0o644)
	if _, err := LoadCatalog(cat3); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "required from catalog version 2") {
		t.Fatalf("missing decision_ref: %v", err)
	}
	cat4, _ := copyTree(t)
	cb4, _ := os.ReadFile(cat4)
	_ = os.WriteFile(cat4, []byte(strings.Replace(string(cb4), `"version": 2`, `"version": 1`, 1)), 0o644)
	if _, err := LoadCatalog(cat4); !errors.Is(err, ErrCatalog) || !strings.Contains(err.Error(), "version-1 catalog") {
		t.Fatalf("v1 with decision fields: %v", err)
	}
}
