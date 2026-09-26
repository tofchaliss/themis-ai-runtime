package model

// P-L8-1 / P-L8-2 (openspec/changes/archive/2026-09-23-layer-08-subagents, M0): the provider
// response ceiling and the provider-reported identity, at both
// adapters. Over-limit → typed termination, nothing parsed; at-limit →
// an ordinary turn. Mutation: remove the limiter → the over-limit
// cases pass through as content and these tests fail.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/llm"
)

func serve(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func ollamaBody(content string) string {
	return `{"model":"qwen2.5:7b","done":true,"message":{"role":"assistant","content":"` + content + `"}}`
}

func openaiBody(content string) string {
	return `{"model":"gpt-x","choices":[{"message":{"role":"assistant","content":"` + content + `"},"finish_reason":"stop"}],"usage":{}}`
}

func TestResponseCeiling(t *testing.T) {
	req := ExecutionRequest{Model: "requested", Messages: []Message{{Role: RoleUser, Content: "hi"}}}
	for _, tc := range []struct {
		name string
		body func(string) string
		mk   func(url string, max int64) Interface
	}{
		{"ollama", ollamaBody, func(url string, max int64) Interface {
			rt := NewOllamaChat(url)
			rt.MaxResponseBytes = max
			return rt
		}},
		{"openai", openaiBody, func(url string, max int64) Interface {
			rt := NewOpenAIChat(url, "")
			rt.MaxResponseBytes = max
			return rt
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pad := strings.Repeat("x", 100)
			exact := tc.body(pad)
			// At the ceiling: an ordinary turn, identity reported.
			srv := serve(t, exact)
			resp, err := tc.mk(srv.URL, int64(len(exact))).Execute(context.Background(), req)
			if err != nil || resp.Content != pad {
				t.Fatalf("at-limit body must be an ordinary turn: %v %+v", err, resp)
			}
			if resp.Identity.WireModel != "requested" || resp.Identity.Reported == "" || resp.Identity.Reported == resp.Identity.WireModel {
				t.Fatalf("reported ≠ requested must be observable: %+v", resp.Identity)
			}
			// One byte over: typed termination, nothing parsed, no turn.
			resp, err = tc.mk(srv.URL, int64(len(exact))-1).Execute(context.Background(), req)
			if !errors.Is(err, ErrResponseOverCeiling) || resp != nil {
				t.Fatalf("over-limit body must terminate typed with no response: %v %+v", err, resp)
			}
			// Default ceiling from the constructor is the compiled bound.
			if o, ok := tc.mk(srv.URL, DefaultMaxResponseBytes).(interface{ Name() string }); !ok || o == nil {
				t.Fatal("adapter")
			}
			// An adapter with no ceiling fails closed rather than reading
			// unbounded — a zero value cannot mean "unlimited".
			if _, err := tc.mk(srv.URL, 0).Execute(context.Background(), req); !errors.Is(err, ErrResponseOverCeiling) {
				t.Fatalf("zero ceiling must refuse, not read unbounded: %v", err)
			}
			if _, err := tc.mk(srv.URL, DefaultMaxResponseBytes+1).Execute(context.Background(), req); !errors.Is(err, ErrResponseOverCeiling) {
				t.Fatalf("a ceiling above the compiled bound must refuse: %v", err)
			}
		})
	}
	if NewOllamaChat("x").MaxResponseBytes != DefaultMaxResponseBytes || NewOpenAIChat("x", "").MaxResponseBytes != DefaultMaxResponseBytes {
		t.Fatal("constructors must set the compiled ceiling")
	}
}

// The registry narrows the ceiling; it never widens it.
func TestRegistryNarrowsResponseCeiling(t *testing.T) {
	reg := &llm.Registry{Models: map[string]llm.Entry{
		"narrow": {Runtime: "ollama", MaxResponseBytes: 1024},
		"wide":   {Runtime: "ollama", MaxResponseBytes: DefaultMaxResponseBytes + 1},
		"plain":  {Runtime: "ollama"},
		"neg":    {Runtime: "openai", Endpoint: "http://x", MaxResponseBytes: -1},
	}}
	rt, _, _, err := Resolve(reg, "narrow")
	if err != nil || rt.(*OllamaChat).MaxResponseBytes != 1024 {
		t.Fatalf("narrowing: %v %+v", err, rt)
	}
	rt, _, _, err = Resolve(reg, "plain")
	if err != nil || rt.(*OllamaChat).MaxResponseBytes != DefaultMaxResponseBytes {
		t.Fatalf("absent = compiled default: %v", err)
	}
	if _, _, _, err := Resolve(reg, "wide"); err == nil || !strings.Contains(err.Error(), "never widens") {
		t.Fatalf("widening must refuse: %v", err)
	}
	if _, _, _, err := Resolve(reg, "neg"); err == nil || !strings.Contains(err.Error(), "never widens") {
		t.Fatalf("negative must refuse: %v", err)
	}
}

// Security review LOW-4: endpoints persist as scheme://host only.
func TestRedactEndpoint(t *testing.T) {
	for in, want := range map[string]string{
		"http://localhost:11434":                           "http://localhost:11434",
		"https://user:secret@api.example.com/v1?key=abc#x": "https://api.example.com",
		"not a url": "",
		"":          "",
	} {
		if got := RedactEndpoint(in); got != want {
			t.Fatalf("%q → %q, want %q", in, got, want)
		}
	}
}
