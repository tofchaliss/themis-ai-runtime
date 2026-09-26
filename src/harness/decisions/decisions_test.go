package decisions

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const rec = `{"id":"reg-skill-x-1","kind":"registration","target":{"name":"x","version":1,"hash":"` + h64a + `"},"evidence":[],"rationale":"initial registration; no L11 comparison","decided_at":"2026-09-26","actor":{"kind":"commit","id":"tofchaliss"}}`
const h64a = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestDecisionRecordBindsTwoWay(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reg-skill-x-1.json"), []byte(rec), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(dir, "reg-skill-x-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Bind(dir, "reg-skill-x-1", r.Hash, "x", 1, h64a); err != nil {
		t.Fatalf("valid binding refused: %v", err)
	}
	// Wrong pinned hash, wrong target, wrong version: each refuses, named.
	if _, err := Bind(dir, "reg-skill-x-1", strings.Repeat("b", 64), "x", 1, h64a); !errors.Is(err, ErrDecision) || !strings.Contains(err.Error(), "record bytes hash") {
		t.Errorf("hash: %v", err)
	}
	if _, err := Bind(dir, "reg-skill-x-1", r.Hash, "y", 1, h64a); !errors.Is(err, ErrDecision) || !strings.Contains(err.Error(), "record targets") {
		t.Errorf("target: %v", err)
	}
	if _, err := Bind(dir, "reg-skill-x-1", r.Hash, "x", 2, h64a); !errors.Is(err, ErrDecision) {
		t.Errorf("version: %v", err)
	}
	if _, err := Bind(dir, "missing", r.Hash, "x", 1, h64a); !errors.Is(err, ErrDecision) {
		t.Errorf("missing: %v", err)
	}
}

func TestDecisionRecordRefusals(t *testing.T) {
	cases := map[string]string{
		"kind":          strings.Replace(rec, `"registration"`, `"promotion"`, 1),
		"no evidence":   strings.Replace(rec, `"evidence":[],`, ``, 1),
		"bad evidence":  strings.Replace(rec, `"evidence":[]`, `"evidence":[{"criterion":"k@1","package_object_id":"abc","record_root":"r"}]`, 1),
		"actor kind":    strings.Replace(rec, `"kind":"commit"`, `"kind":"human"`, 1),
		"rationale":     strings.Replace(rec, `"rationale":"initial registration; no L11 comparison"`, `"rationale":" "`, 1),
		"unknown field": strings.Replace(rec, `"id":`, `"outcome":"promoted","id":`, 1),
		"dup key":       strings.Replace(rec, `"id":`, `"kind":"reliance","id":`, 1),
		"target hash":   strings.Replace(rec, h64a, "abc", 1),
	}
	for name, raw := range cases {
		if _, err := Parse([]byte(raw), name); !errors.Is(err, ErrDecision) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A well-formed evidence citation is accepted and never interpreted.
	withEv := strings.Replace(rec, `"evidence":[]`, `"evidence":[{"criterion":"walk-report-score-delta@1","package_object_id":"sha256:`+h64a+`","record_root":"/srv/themis/rsys/ratchet"}]`, 1)
	if _, err := Parse([]byte(withEv), "ev"); err != nil {
		t.Fatal(err)
	}
	// The file's name must be the record's id.
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "other.json"), []byte(rec), 0o644)
	if _, err := Load(dir, "other"); !errors.Is(err, ErrDecision) {
		t.Errorf("id/file mismatch: %v", err)
	}
	if Dir("/x/policies/skills/catalog.json") != "/x/policies/decisions" {
		t.Fatal("Dir")
	}
}
