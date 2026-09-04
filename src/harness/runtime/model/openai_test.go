package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tofchaliss/themis/internal/llm"
)

func TestOpenAIChatToolConversation(t *testing.T) {

	var gotAuth string
	var requests []map[string]any
	bodies := []string{
		`{"choices": [{"finish_reason": "tool_calls",
		   "message": {"role": "assistant", "content": "",
		     "tool_calls": [{"id": "call_1", "type": "function",
		       "function": {"name": "read_file",
		         "arguments": "{\"path\": \"go.mod\"}"}}]}}],
		  "usage": {"prompt_tokens": 12, "completion_tokens": 8, "total_tokens": 20}}`,
		`{"choices": [{"finish_reason": "stop",
		   "message": {"role": "assistant", "content": "module themis"}}],
		  "usage": {"prompt_tokens": 25, "completion_tokens": 4, "total_tokens": 29}}`,
	}
	call := 0

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)
			requests = append(requests, req)
			w.Write([]byte(bodies[call]))
			call++
		}))
	t.Cleanup(srv.Close)

	chat := NewOpenAIChat(srv.URL, "sk-test-key-value")
	params := json.RawMessage(`{"type":"object"}`)

	req := ExecutionRequest{
		Model: "gpt-x",
		Messages: []Message{
			{Role: RoleUser, Content: "what module is this?"},
		},
		Tools:   []ToolDef{{Name: "read_file", Description: "read", Parameters: params}},
		Options: llm.DefaultOptions(),
	}

	resp, err := chat.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if gotAuth != "Bearer sk-test-key-value" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if resp.Termination != TerminationToolCalls {
		t.Errorf("termination = %s", resp.Termination)
	}
	if resp.Usage.PromptTokens != 12 || resp.Usage.CompletionTokens != 8 ||
		resp.Usage.TotalTokens != 20 {
		t.Errorf("usage = %+v", resp.Usage)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].ID != "call_1" {
		t.Fatalf("tool calls = %+v", resp.ToolCalls)
	}
	// The string-encoded arguments become raw JSON of the object.
	var args map[string]any
	if err := json.Unmarshal(resp.ToolCalls[0].Arguments, &args); err != nil || args["path"] != "go.mod" {
		t.Errorf("arguments = %s (%v)", resp.ToolCalls[0].Arguments, err)
	}

	// Wire shape: top-level options, tools declared.
	wire := requests[0]
	if wire["temperature"] != 0.0 || wire["seed"] != 42.0 {
		t.Errorf("top-level options = %v/%v", wire["temperature"], wire["seed"])
	}
	if _, ok := wire["tools"].([]any); !ok {
		t.Error("tools not sent")
	}

	// Turn 2 with the tool result; tool_call_id must round-trip.
	req.Messages = append(req.Messages,
		Message{Role: RoleAssistant, ToolCalls: resp.ToolCalls},
		Message{Role: RoleTool, Content: "module themis", ToolCallID: "call_1"},
	)

	resp2, err := chat.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Termination != TerminationStop || resp2.Content != "module themis" {
		t.Errorf("final = %q / %s", resp2.Content, resp2.Termination)
	}

	msgs := requests[1]["messages"].([]any)
	toolMsg := msgs[2].(map[string]any)
	if toolMsg["tool_call_id"] != "call_1" {
		t.Errorf("tool_call_id lost: %v", toolMsg)
	}
	assistant := msgs[1].(map[string]any)
	tcs := assistant["tool_calls"].([]any)
	fn := tcs[0].(map[string]any)["function"].(map[string]any)
	if _, isString := fn["arguments"].(string); !isString {
		t.Error("assistant tool-call arguments must re-encode as a string")
	}
}

func TestOpenAIChatFailures(t *testing.T) {

	serve := func(body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("error field", func(t *testing.T) {
		srv := serve(`{"error": {"message": "quota exceeded"}}`)
		_, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty choices", func(t *testing.T) {
		srv := serve(`{"choices": []}`)
		_, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "upstream down", http.StatusBadGateway)
			}))
		t.Cleanup(srv.Close)

		_, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("malformed body", func(t *testing.T) {
		srv := serve(`not json at all`)
		_, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("length termination", func(t *testing.T) {
		srv := serve(`{"choices": [{"finish_reason": "length",
			"message": {"role": "assistant", "content": "trunc"}}]}`)
		resp, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Termination != TerminationLength {
			t.Errorf("termination = %s, want length", resp.Termination)
		}
	})

	t.Run("tool_calls finish reason with empty array", func(t *testing.T) {
		srv := serve(`{"choices": [{"finish_reason": "tool_calls",
			"message": {"role": "assistant", "content": ""}}]}`)
		resp, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Termination != TerminationToolCalls {
			t.Errorf("termination = %s, want tool_calls", resp.Termination)
		}
	})

	t.Run("malformed tool arguments transported verbatim", func(t *testing.T) {
		// Contract pin: arguments are untrusted transport, validated by
		// the Tool Interface — never here.
		srv := serve(`{"choices": [{"finish_reason": "tool_calls",
			"message": {"role": "assistant", "content": "",
			  "tool_calls": [{"id": "c1", "type": "function",
			    "function": {"name": "t", "arguments": "not{json"}}]}}]}`)
		resp, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		// Invalid JSON is re-encoded as a JSON string so the response
		// stays serializable for the audit record; content is preserved.
		var s string
		if err := json.Unmarshal(resp.ToolCalls[0].Arguments, &s); err != nil || s != "not{json" {
			t.Errorf("arguments = %s (%v), want quoted verbatim content", resp.ToolCalls[0].Arguments, err)
		}
		if _, err := json.Marshal(resp); err != nil {
			t.Errorf("response must stay serializable: %v", err)
		}
	})

	t.Run("unknown finish reason maps to unknown", func(t *testing.T) {
		srv := serve(`{"choices": [{"finish_reason": "content_filter",
			"message": {"role": "assistant", "content": "partial"}}]}`)
		resp, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Termination != TerminationUnknown {
			t.Errorf("termination = %s, want unknown (fail-closed)", resp.Termination)
		}
	})

	t.Run("no auth header without key", func(t *testing.T) {
		var gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.Write([]byte(`{"choices": [{"finish_reason": "stop",
					"message": {"role": "assistant", "content": "ok"}}]}`))
			}))
		t.Cleanup(srv.Close)

		if _, err := NewOpenAIChat(srv.URL, "").Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}}); err != nil {
			t.Fatal(err)
		}
		if gotAuth != "" {
			t.Errorf("auth header sent without a key: %q", gotAuth)
		}
	})
}
