package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func writeDefinition(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(
		filepath.Join(dir, name),
		[]byte(content),
		0644,
	); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDefinitions(t *testing.T) {

	t.Run("loads and sorts definitions", func(t *testing.T) {
		dir := t.TempDir()
		writeDefinition(t, dir, "B002.json", `{"id": "B002", "prompt": "B002.md"}`)
		writeDefinition(t, dir, "B001.json", `{"id": "B001", "prompt": "B001.md"}`)

		defs, err := LoadDefinitions(dir)
		if err != nil {
			t.Fatal(err)
		}

		if len(defs) != 2 {
			t.Fatalf("got %d definitions, want 2", len(defs))
		}
		if defs[0].ID != "B001" || defs[1].ID != "B002" {
			t.Errorf("definitions not sorted: %v, %v", defs[0].ID, defs[1].ID)
		}
	})

	t.Run("empty directory yields no definitions", func(t *testing.T) {
		defs, err := LoadDefinitions(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if len(defs) != 0 {
			t.Errorf("got %d definitions, want 0", len(defs))
		}
	})

	t.Run("missing id is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeDefinition(t, dir, "bad.json", `{"prompt": "x.md"}`)

		if _, err := LoadDefinitions(dir); err == nil {
			t.Error("expected error for missing id")
		}
	})

	t.Run("missing prompt is rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeDefinition(t, dir, "bad.json", `{"id": "B001"}`)

		if _, err := LoadDefinitions(dir); err == nil {
			t.Error("expected error for missing prompt")
		}
	})

	t.Run("duplicate ids are rejected", func(t *testing.T) {
		dir := t.TempDir()
		writeDefinition(t, dir, "a.json", `{"id": "B001", "prompt": "a.md"}`)
		writeDefinition(t, dir, "b.json", `{"id": "B001", "prompt": "b.md"}`)

		if _, err := LoadDefinitions(dir); err == nil {
			t.Error("expected error for duplicate id")
		}
	})

	t.Run("malformed JSON is rejected with filename", func(t *testing.T) {
		dir := t.TempDir()
		writeDefinition(t, dir, "bad.json", `{`)

		if _, err := LoadDefinitions(dir); err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}
