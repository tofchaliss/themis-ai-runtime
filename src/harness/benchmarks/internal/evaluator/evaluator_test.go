package evaluator

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRunFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "B001.json")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestEvaluateRun(t *testing.T) {

	t.Run("envelope run file", func(t *testing.T) {
		path := writeRunFile(t, `{
			"benchmark": "B001",
			"model": "test-model",
			"runtime": "openai",
			"answer": "hello",
			"options": {"temperature": 0, "seed": 42},
			"metrics": {
				"prompt_tokens": 10,
				"completion_tokens": 20,
				"total_tokens": 30,
				"generation_time_ms": 4000,
				"tokens_per_second": 5
			},
			"raw": {"anything": true}
		}`)

		r, err := EvaluateRun("B001", path)
		if err != nil {
			t.Fatal(err)
		}

		if r.Runtime != "openai" || r.Model != "test-model" || r.Answer != "hello" {
			t.Errorf("result = %+v", r)
		}
		if r.Metrics.CompletionTokens != 20 || r.Metrics.TokensPerSecond != 5 {
			t.Errorf("metrics = %+v", r.Metrics)
		}
	})

	t.Run("legacy ollama run file", func(t *testing.T) {
		path := writeRunFile(t, `{
			"model": "test-model",
			"response": "hello",
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 20,
			"eval_duration": 4000000000,
			"total_duration": 5000000000
		}`)

		r, err := EvaluateRun("B001", path)
		if err != nil {
			t.Fatal(err)
		}

		if r.Runtime != "ollama" || r.Answer != "hello" {
			t.Errorf("result = %+v", r)
		}
		if r.Metrics.GenerationTimeMS != 4000 {
			t.Errorf("GenerationTimeMS = %f, want 4000", r.Metrics.GenerationTimeMS)
		}
		// 20 tokens over 4 seconds.
		if r.Metrics.TokensPerSecond != 5 {
			t.Errorf("TokensPerSecond = %f, want 5", r.Metrics.TokensPerSecond)
		}
	})

	t.Run("legacy error payload is rejected", func(t *testing.T) {
		path := writeRunFile(t, `{"error": "model not found"}`)

		if _, err := EvaluateRun("B001", path); err == nil {
			t.Error("expected error for runtime-error payload")
		}
	})

	t.Run("legacy incomplete run is rejected", func(t *testing.T) {
		path := writeRunFile(t, `{"model": "m", "response": "partial", "done": false}`)

		if _, err := EvaluateRun("B001", path); err == nil {
			t.Error("expected error for incomplete run")
		}
	})

	t.Run("malformed JSON is rejected", func(t *testing.T) {
		path := writeRunFile(t, `not json`)

		if _, err := EvaluateRun("B001", path); err == nil {
			t.Error("expected error for malformed run file")
		}
	})
}
