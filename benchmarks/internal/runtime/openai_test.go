package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
)

func TestOpenAIRun(t *testing.T) {

	t.Run("successful completion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("path = %s, want /v1/chat/completions", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
					t.Errorf("Authorization = %q", got)
				}

				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				if req["model"] != "test-model" {
					t.Errorf("model = %v", req["model"])
				}
				if req["temperature"] != 0.0 {
					t.Errorf("temperature = %v, want 0", req["temperature"])
				}
				if req["seed"] != 42.0 {
					t.Errorf("seed = %v, want 42", req["seed"])
				}

				json.NewEncoder(w).Encode(map[string]any{
					"choices": []map[string]any{
						{"message": map[string]any{"content": "the answer"}},
					},
					"usage": map[string]any{
						"prompt_tokens":     10,
						"completion_tokens": 20,
						"total_tokens":      30,
					},
				})
			},
		))
		defer server.Close()

		resp, err := NewOpenAI(server.URL+"/v1", "test-key").Run(
			context.Background(),
			model.Request{
				Model:   "test-model",
				Prompt:  "question",
				Options: model.DefaultOptions(),
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Answer != "the answer" {
			t.Errorf("Answer = %q", resp.Answer)
		}
		if resp.Metrics.PromptTokens != 10 || resp.Metrics.CompletionTokens != 20 {
			t.Errorf("tokens = %d/%d, want 10/20",
				resp.Metrics.PromptTokens, resp.Metrics.CompletionTokens)
		}
		if resp.Runtime != "openai" {
			t.Errorf("Runtime = %q, want openai", resp.Runtime)
		}
	})

	t.Run("non-200 status is an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"error":{"message":"invalid key"}}`, http.StatusUnauthorized)
			},
		))
		defer server.Close()

		_, err := NewOpenAI(server.URL, "bad").Run(
			context.Background(),
			model.Request{Model: "m", Prompt: "q"},
		)
		if err == nil {
			t.Fatal("expected error for 401 response")
		}
	})

	t.Run("empty choices is an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
			},
		))
		defer server.Close()

		_, err := NewOpenAI(server.URL, "").Run(
			context.Background(),
			model.Request{Model: "m", Prompt: "q"},
		)
		if err == nil {
			t.Fatal("expected error for empty choices")
		}
	})
}
