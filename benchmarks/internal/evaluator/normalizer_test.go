package evaluator

import (
	"testing"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
)

func TestNormalize(t *testing.T) {

	t.Run("valid run", func(t *testing.T) {
		resp := &model.Response{
			Model:   "test-model",
			Runtime: "ollama",
			Answer:  "hello",
			Raw: []byte(`{
				"model": "test-model",
				"response": "hello",
				"done": true,
				"prompt_eval_count": 10,
				"eval_count": 20,
				"load_duration": 1000000,
				"prompt_eval_duration": 2000000,
				"eval_duration": 4000000000,
				"total_duration": 5000000000
			}`),
		}

		r, err := Normalize("B001", resp)
		if err != nil {
			t.Fatal(err)
		}

		if r.Metrics.PromptTokens != 10 || r.Metrics.CompletionTokens != 20 {
			t.Errorf("tokens = %d/%d, want 10/20",
				r.Metrics.PromptTokens, r.Metrics.CompletionTokens)
		}
		if r.Metrics.TotalTokens != 30 {
			t.Errorf("TotalTokens = %d, want 30", r.Metrics.TotalTokens)
		}
		if r.Metrics.LoadTimeMS != 1 {
			t.Errorf("LoadTimeMS = %f, want 1", r.Metrics.LoadTimeMS)
		}
		if r.Metrics.GenerationTimeMS != 4000 {
			t.Errorf("GenerationTimeMS = %f, want 4000", r.Metrics.GenerationTimeMS)
		}
		// 20 tokens over 4 seconds.
		if r.Metrics.TokensPerSecond != 5 {
			t.Errorf("TokensPerSecond = %f, want 5", r.Metrics.TokensPerSecond)
		}
	})

	t.Run("missing metric fields does not panic", func(t *testing.T) {
		resp := &model.Response{
			Model: "test-model",
			Raw:   []byte(`{"model": "test-model", "response": "x", "done": true}`),
		}

		r, err := Normalize("B001", resp)
		if err != nil {
			t.Fatal(err)
		}
		if r.Metrics.TokensPerSecond != 0 {
			t.Errorf("TokensPerSecond = %f, want 0", r.Metrics.TokensPerSecond)
		}
	})

	t.Run("runtime error payload is rejected", func(t *testing.T) {
		resp := &model.Response{
			Raw: []byte(`{"error": "model not found"}`),
		}

		if _, err := Normalize("B001", resp); err == nil {
			t.Error("expected error for run containing runtime error")
		}
	})

	t.Run("incomplete run is rejected", func(t *testing.T) {
		resp := &model.Response{
			Raw: []byte(`{"model": "m", "response": "partial", "done": false}`),
		}

		if _, err := Normalize("B001", resp); err == nil {
			t.Error("expected error for incomplete run")
		}
	})

	t.Run("malformed JSON is rejected", func(t *testing.T) {
		resp := &model.Response{Raw: []byte(`not json`)}

		if _, err := Normalize("B001", resp); err == nil {
			t.Error("expected error for malformed raw run")
		}
	})
}
