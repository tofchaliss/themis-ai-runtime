package benchmark_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/benchmarks/internal/benchmark"
	"github.com/tofchaliss/themis/benchmarks/internal/evaluator"
	"github.com/tofchaliss/themis/benchmarks/internal/report"
	"github.com/tofchaliss/themis/benchmarks/internal/validator"
	"github.com/tofchaliss/themis/internal/llm"
)

// TestPipelineEndToEnd drives the full pipeline — run, evaluate,
// validate, report — against a mock Ollama server in a temporary suite
// root.
func TestPipelineEndToEnd(t *testing.T) {

	root := t.TempDir()

	writeFile(t, root, "definitions/B001.json", `{
		"id": "B001",
		"name": "Test Benchmark",
		"category": "Test",
		"expected": "B001.json",
		"prompt": "B001.md",
		"weight": 10
	}`)

	writeFile(t, root, "prompts/B001.md", "Name the vulnerable product.")

	writeFile(t, root, "expected/B001.json", `{
		"benchmark": "B001",
		"validator": "keyword",
		"required": ["Log4j", "JNDI"],
		"forbidden": ["XSS"]
	}`)

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"model":             "test-model",
				"response":          "Log4j is vulnerable via JNDI lookups.",
				"done":              true,
				"prompt_eval_count": 10,
				"eval_count":        20,
				"eval_duration":     2_000_000_000,
				"total_duration":    2_500_000_000,
			})
		},
	))
	defer server.Close()

	const date = "2026-01-01"

	// Run.
	err := benchmark.RunAll(context.Background(), benchmark.RunConfig{
		Runtime: llm.NewOllama(server.URL),
		Name:    "test-model",
		Model:   "test-model",
		Options: llm.DefaultOptions(),
		Root:    root,
		Date:    date,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// The manifest exists and records the run.
	manifestData, err := os.ReadFile(
		filepath.Join(root, "runs", date, "test-model", "manifest.json"),
	)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}

	var manifest benchmark.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Runtime != "ollama" || manifest.Options.Seed != 42 {
		t.Errorf("manifest = %+v", manifest)
	}
	if len(manifest.Prompts) != 1 || manifest.Prompts["B001"] == "" {
		t.Errorf("manifest prompts = %v", manifest.Prompts)
	}

	// Evaluate. The manifest must not be treated as a run.
	if err := evaluator.EvaluateAll(root, date, "test-model"); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	// Validate.
	if err := validator.ValidateAll(root, date, "test-model"); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Report.
	benchmarks, err := report.LoadResponses(root, date, "test-model")
	if err != nil {
		t.Fatalf("load responses: %v", err)
	}

	r := report.Generate("test-model", date, benchmarks)

	if r.Benchmarks != 1 || r.Validated != 1 || r.AverageScore != 100 {
		t.Errorf("report = %+v", r)
	}

	if err := report.Write(root, date, r); err != nil {
		t.Fatalf("write report: %v", err)
	}

	md, err := os.ReadFile(
		filepath.Join(root, "reports", date, "test-model.md"),
	)
	if err != nil {
		t.Fatalf("report file: %v", err)
	}
	if !strings.Contains(string(md), "100%") {
		t.Errorf("report content:\n%s", md)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()

	path := filepath.Join(root, rel)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
