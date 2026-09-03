package report

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeResponse(t *testing.T, root, date, model, benchmark, answer string) {
	t.Helper()

	dir := filepath.Join(root, "responses", date, model)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `{
		"benchmark": "` + benchmark + `",
		"model": "` + model + `",
		"runtime": "ollama",
		"answer": "` + answer + `",
		"metrics": {"tokens_per_second": 10}
	}`

	if err := os.WriteFile(
		filepath.Join(dir, benchmark+".json"),
		[]byte(content),
		0644,
	); err != nil {
		t.Fatal(err)
	}
}

func writeValidation(t *testing.T, root, date, model, benchmark string, score int) {
	t.Helper()

	dir := filepath.Join(root, "validation", date, model)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	content := fmt.Sprintf(
		`{"benchmark": %q, "score": %d, "passed": 1, "failed": 1, "missing": [], "violations": []}`,
		benchmark,
		score,
	)

	if err := os.WriteFile(
		filepath.Join(dir, benchmark+".json"),
		[]byte(content),
		0644,
	); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverRuns(t *testing.T) {

	t.Run("finds runs including slashed model names", func(t *testing.T) {
		root := t.TempDir()

		writeResponse(t, root, "2026-08-19", "modelA", "B001", "x")
		writeResponse(t, root, "2026-08-21", "modelA", "B001", "x")
		writeResponse(t, root, "2026-08-21", "org/model-b", "B001", "x")

		refs, err := DiscoverRuns(root)
		if err != nil {
			t.Fatal(err)
		}

		want := []RunRef{
			{Date: "2026-08-19", Model: "modelA"},
			{Date: "2026-08-21", Model: "modelA"},
			{Date: "2026-08-21", Model: "org/model-b"},
		}

		if !reflect.DeepEqual(refs, want) {
			t.Errorf("refs = %v, want %v", refs, want)
		}
	})

	t.Run("ignores non-date directories", func(t *testing.T) {
		root := t.TempDir()
		writeResponse(t, root, "not-a-date", "modelA", "B001", "x")

		refs, err := DiscoverRuns(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 0 {
			t.Errorf("refs = %v, want none", refs)
		}
	})

	t.Run("missing responses directory yields no runs", func(t *testing.T) {
		refs, err := DiscoverRuns(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 0 {
			t.Errorf("refs = %v, want none", refs)
		}
	})
}

func TestCompare(t *testing.T) {

	root := t.TempDir()

	writeResponse(t, root, "2026-08-19", "modelA", "B001", "old")
	writeValidation(t, root, "2026-08-19", "modelA", "B001", 50)

	writeResponse(t, root, "2026-08-21", "modelA", "B001", "new")
	writeValidation(t, root, "2026-08-21", "modelA", "B001", 50)

	writeResponse(t, root, "2026-08-21", "org/model-b", "B002", "x")

	c, err := Compare(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(c.Models) != 2 {
		t.Fatalf("models = %d, want 2", len(c.Models))
	}

	modelA := c.Models[0]
	if modelA.Model != "modelA" || len(modelA.Runs) != 2 {
		t.Errorf("modelA = %s with %d runs, want modelA with 2", modelA.Model, len(modelA.Runs))
	}
	if modelA.Latest().Date != "2026-08-21" {
		t.Errorf("latest date = %s, want 2026-08-21", modelA.Latest().Date)
	}

	md := ComparisonMarkdown(c)

	for _, want := range []string{
		"# Themis Model Comparison",
		"| modelA | 2026-08-21 |",
		"## Benchmark Scores",
		"## Score History",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q", want)
		}
	}

	// modelA has no B002 result -> "·"; org/model-b B002 is unvalidated -> "-".
	if !strings.Contains(md, "| B002 | · | - |") {
		t.Errorf("matrix cells wrong:\n%s", md)
	}
}

func TestCompareEmpty(t *testing.T) {
	if _, err := Compare(t.TempDir()); err == nil {
		t.Error("expected error when no runs exist")
	}
}
