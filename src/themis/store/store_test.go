package store

// Register A for the read door (D-T-9): the governed registries load,
// the pin is the exact bytes, Read serves exact record bytes for
// active records only, and every refusal names its gate. Positive
// first.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var governed = filepath.Join("..", "..", "..", "policies", "themis")

func TestGovernedStoreLoadsAndServes(t *testing.T) {
	s, err := Load(governed)
	if err != nil {
		t.Fatalf("the governed store must load: %v", err)
	}
	h, err := HashOf(governed)
	if err != nil || h != s.Hash {
		t.Fatalf("pin must be the exact bytes: %v %s %s", err, h, s.Hash)
	}
	b, err := s.Read("finding", "FIND-2026-0001")
	if err != nil {
		t.Fatalf("P0 finding: %v", err)
	}
	var f Finding
	if err := json.Unmarshal(b, &f); err != nil || f.Product != "PROD-demo-vuln-app" || f.Component != "vulnerable-dep" {
		t.Fatalf("exact bytes: %v %+v", err, f)
	}
	// Exact bytes: the served record is a substring of the registry
	// file, byte for byte.
	raw, _ := os.ReadFile(filepath.Join(governed, "findings.json"))
	if !strings.Contains(string(raw), string(b)) {
		t.Fatal("Read must serve the registered bytes, not a re-encoding")
	}
	if _, err := s.Read("product", "PROD-demo-vuln-app"); err != nil {
		t.Fatalf("P0 product: %v", err)
	}
	if st, ok := s.Exists("finding", "FIND-2026-0001"); !ok || st != "active" {
		t.Fatalf("exists: %v %s", ok, st)
	}
	// The served bytes carry no authority field of any name.
	for _, bad := range []string{"authority", "trust", "class", "disposition"} {
		if strings.Contains(string(b), `"`+bad+`"`) {
			t.Fatalf("a record must not carry %q — the L4 registration mints the class", bad)
		}
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goodProducts = `{"version":1,"entries":[{"id":"PROD-x","name":"x","version":"1","state":"active","steward":"s"},{"id":"PROD-old","name":"old","version":"0","state":"withdrawn","steward":"s"}]}`
const goodFindings = `{"version":1,"entries":[{"id":"FIND-1","product":"PROD-x","advisory":"A","component":"c","affected_version":"1","fixed_version":"2","severity":"high","summary":"s","state":"active","steward":"s"},{"id":"FIND-gone","product":"PROD-x","advisory":"A","component":"c","affected_version":"1","fixed_version":"2","severity":"low","summary":"s","state":"withdrawn","steward":"s"}]}`

func TestReadDoorRefusals(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "products.json", goodProducts)
	write(t, dir, "findings.json", goodFindings)
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read("finding", "FIND-1"); err != nil {
		t.Fatalf("positive: %v", err)
	}
	for name, c := range map[string]struct{ kind, id, want string }{
		"withdrawn finding": {"finding", "FIND-gone", "withdrawn"},
		"withdrawn product": {"product", "PROD-old", "withdrawn"},
		"unknown finding":   {"finding", "FIND-999", "no finding"},
		"unknown kind":      {"position", "FIND-1", "unknown record kind"},
		"kind mismatch":     {"product", "FIND-1", "no product"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.Read(c.kind, c.id)
			if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want unavailable %q: %v", c.want, err)
			}
		})
	}
	// Withdrawn still EXISTS (history) — the D-T-7 gate sees it.
	if st, ok := s.Exists("finding", "FIND-gone"); !ok || st != "withdrawn" {
		t.Fatalf("withdrawn must exist as history: %v %s", ok, st)
	}
}

func TestStoreLoaderRefusals(t *testing.T) {
	cases := []struct{ name, products, findings, want string }{
		{"finding references unregistered product", goodProducts, strings.Replace(goodFindings, `"product":"PROD-x","advisory":"A","component":"c","affected_version":"1","fixed_version":"2","severity":"high"`, `"product":"PROD-ghost","advisory":"A","component":"c","affected_version":"1","fixed_version":"2","severity":"high"`, 1), "unregistered product"},
		{"bad finding id", goodProducts, strings.Replace(goodFindings, `"id":"FIND-1"`, `"id":"F1"`, 1), "bad id"},
		{"unknown severity", goodProducts, strings.Replace(goodFindings, `"severity":"high"`, `"severity":"urgent"`, 1), "unknown severity"},
		{"unknown state", goodProducts, strings.Replace(goodFindings, `"state":"active","steward":"s"},{"id":"FIND-gone"`, `"state":"proposed","steward":"s"},{"id":"FIND-gone"`, 1), "unknown state"},
		{"duplicate finding", goodProducts, strings.Replace(goodFindings, `"id":"FIND-gone"`, `"id":"FIND-1"`, 1), "duplicate registration"},
		{"disposition smuggled onto a finding", goodProducts, strings.Replace(goodFindings, `"summary":"s","state":"active"`, `"summary":"s","disposition":"mitigated","state":"active"`, 1), "unknown field"},
		{"authority smuggled onto a product", strings.Replace(goodProducts, `"name":"x"`, `"name":"x","authority":"governed-record"`, 1), goodFindings, "unknown field"},
		{"case-variant key", goodProducts, strings.Replace(goodFindings, `"severity"`, `"Severity"`, 1), "exact lowercase key"},
		{"duplicate key", goodProducts, strings.Replace(goodFindings, `"summary":"s"`, `"summary":"s","summary":"t"`, 1), "duplicate key"},
		{"trailing content", goodProducts, goodFindings + " {}", "trailing content"},
		{"version missing", goodProducts, strings.Replace(goodFindings, `"version":1`, `"version":0`, 1), "version required"},
		{"bad product id", strings.Replace(goodProducts, `"id":"PROD-x"`, `"id":"x"`, 1), goodFindings, "bad id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, "products.json", c.products)
			write(t, dir, "findings.json", c.findings)
			_, err := Load(dir)
			if !errors.Is(err, ErrStore) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q: %v", c.want, err)
			}
		})
	}
	// Symlinked registry refuses (regular files only).
	dir := t.TempDir()
	write(t, dir, "products.json", goodProducts)
	outside := filepath.Join(t.TempDir(), "f.json")
	write(t, filepath.Dir(outside), "f.json", goodFindings)
	if err := os.Symlink(outside, filepath.Join(dir, "findings.json")); err == nil {
		if _, err := Load(dir); !errors.Is(err, ErrStore) || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("symlink: %v", err)
		}
	}
}

// The pin changes with any byte of either registry.
func TestPinCoversBothRegistries(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "products.json", goodProducts)
	write(t, dir, "findings.json", goodFindings)
	h0, _ := HashOf(dir)
	write(t, dir, "products.json", goodProducts+"\n")
	h1, _ := HashOf(dir)
	write(t, dir, "products.json", goodProducts)
	write(t, dir, "findings.json", goodFindings+"\n")
	h2, _ := HashOf(dir)
	if h0 == h1 || h0 == h2 || h1 == h2 {
		t.Fatalf("pin must cover both registries byte-exactly: %s %s %s", h0[:8], h1[:8], h2[:8])
	}
}
