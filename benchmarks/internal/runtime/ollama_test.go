package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
)

func TestOllamaRun(t *testing.T) {

	t.Run("successful generation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				if r.URL.Path != "/api/generate" {
					t.Errorf("path = %s, want /api/generate", r.URL.Path)
				}

				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				if req["model"] != "test-model" {
					t.Errorf("model = %v, want test-model", req["model"])
				}
				if req["stream"] != false {
					t.Errorf("stream = %v, want false", req["stream"])
				}

				options, ok := req["options"].(map[string]any)
				if !ok {
					t.Fatalf("options missing from request: %v", req)
				}
				if options["temperature"] != 0.0 {
					t.Errorf("temperature = %v, want 0", options["temperature"])
				}
				if options["seed"] != 42.0 {
					t.Errorf("seed = %v, want 42", options["seed"])
				}

				json.NewEncoder(w).Encode(map[string]any{
					"model":         "test-model",
					"response":      "the answer",
					"done":          true,
					"eval_count":    30,
					"eval_duration": 3_000_000_000,
					"load_duration": 1_000_000,
				})
			},
		))
		defer server.Close()

		resp, err := NewOllama(server.URL).Run(
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
		if resp.Metrics.CompletionTokens != 30 {
			t.Errorf("CompletionTokens = %d, want 30", resp.Metrics.CompletionTokens)
		}
		// 30 tokens over 3 seconds.
		if resp.Metrics.TokensPerSecond != 10 {
			t.Errorf("TokensPerSecond = %f, want 10", resp.Metrics.TokensPerSecond)
		}
		if len(resp.Raw) == 0 {
			t.Error("Raw response not preserved")
		}
	})

	t.Run("non-200 status is an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
			},
		))
		defer server.Close()

		_, err := NewOllama(server.URL).Run(
			context.Background(),
			model.Request{Model: "missing", Prompt: "question"},
		)
		if err == nil {
			t.Fatal("expected error for 404 response")
		}
	})

	t.Run("error payload with 200 status is an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{
					"error": "something failed",
				})
			},
		))
		defer server.Close()

		_, err := NewOllama(server.URL).Run(
			context.Background(),
			model.Request{Model: "m", Prompt: "q"},
		)
		if err == nil {
			t.Fatal("expected error for error payload")
		}
	})

	t.Run("cancelled context aborts the request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
			},
		))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := NewOllama(server.URL).Run(
			ctx,
			model.Request{Model: "m", Prompt: "q"},
		)
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
	})
}
