package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tofchaliss/themis/internal/llm"
)

// scriptedServer answers each successive POST with the next canned
// body, capturing every request payload for wire-shape assertions.
func scriptedServer(t *testing.T, bodies ...string) (*httptest.Server, *[]map[string]any) {
	t.Helper()

	var requests []map[string]any
	call := 0

	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode request: %v", err)
			}
			requests = append(requests, req)

			if call >= len(bodies) {
				t.Error("more requests than scripted responses")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write([]byte(bodies[call]))
			call++
		},
	))
	t.Cleanup(srv.Close)

	return srv, &requests
}

func TestOllamaChatToolConversation(t *testing.T) {

	srv, requests := scriptedServer(t,
		// Turn 1: the model requests a tool.
		`{"model": "m", "done": true, "done_reason": "stop",
		  "message": {"role": "assistant", "content": "",
		    "tool_calls": [{"function": {"name": "read_file",
		      "arguments": {"path": "go.mod"}}}]},
		  "prompt_eval_count": 12, "eval_count": 8,
		  "eval_duration": 2000000000, "total_duration": 3000000000}`,
		// Turn 2: final answer.
		`{"model": "m", "done": true, "done_reason": "stop",
		  "message": {"role": "assistant", "content": "module themis"},
		  "prompt_eval_count": 20, "eval_count": 5}`,
	)

	chat := NewOllamaChat(srv.URL)
	params := json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}}}`)

	req := ExecutionRequest{
		Model: "m",
		Messages: []Message{
			{Role: RoleSystem, Content: "analysis assistant"},
			{Role: RoleUser, Content: "what module is this?"},
		},
		Tools:   []ToolDef{{Name: "read_file", Description: "read a file", Parameters: params}},
		Options: llm.DefaultOptions(),
	}

	resp, err := chat.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Termination != TerminationToolCalls {
		t.Errorf("termination = %s, want tool_calls", resp.Termination)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "read_file" {
		t.Fatalf("tool calls = %+v", resp.ToolCalls)
	}
	var args map[string]any
	if err := json.Unmarshal(resp.ToolCalls[0].Arguments, &args); err != nil || args["path"] != "go.mod" {
		t.Errorf("arguments = %s (%v)", resp.ToolCalls[0].Arguments, err)
	}
	if resp.Usage.TotalTokens != 20 || resp.Usage.GenerationTimeMS != 2000 {
		t.Errorf("usage = %+v", resp.Usage)
	}
	if resp.Identity.Runtime != "ollama" || resp.Provenance.Endpoint != srv.URL {
		t.Errorf("identity/provenance = %+v / %+v", resp.Identity, resp.Provenance)
	}
	if len(resp.Provenance.Raw) == 0 {
		t.Error("provenance raw payload not preserved")
	}

	// Wire shape of turn 1.
	wire := (*requests)[0]
	if wire["stream"] != false {
		t.Error("stream not disabled")
	}
	opts := wire["options"].(map[string]any)
	if opts["temperature"] != 0.0 || opts["seed"] != 42.0 {
		t.Errorf("options = %v (deterministic defaults must nest)", opts)
	}
	if _, ok := wire["tools"].([]any); !ok {
		t.Error("tools not sent")
	}

	// Turn 2: replay with the assistant tool call and the tool result.
	req.Messages = append(req.Messages,
		Message{Role: RoleAssistant, ToolCalls: resp.ToolCalls},
		Message{Role: RoleTool, Content: "module themis", ToolCallID: resp.ToolCalls[0].ID},
	)

	resp2, err := chat.Execute(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Termination != TerminationStop || resp2.Content != "module themis" {
		t.Errorf("final = %q / %s", resp2.Content, resp2.Termination)
	}

	msgs := (*requests)[1]["messages"].([]any)
	if len(msgs) != 4 {
		t.Fatalf("replayed %d messages, want 4", len(msgs))
	}
	toolMsg := msgs[3].(map[string]any)
	if toolMsg["role"] != "tool" || toolMsg["content"] != "module themis" {
		t.Errorf("tool message = %v", toolMsg)
	}
	// The mirror-image wire invariant: on the Ollama wire, replayed
	// assistant tool-call arguments stay a JSON object (OpenAI's are a
	// string).
	assistant := msgs[2].(map[string]any)
	tcs := assistant["tool_calls"].([]any)
	fn := tcs[0].(map[string]any)["function"].(map[string]any)
	if _, isObject := fn["arguments"].(map[string]any); !isObject {
		t.Errorf("replayed arguments = %T, want JSON object", fn["arguments"])
	}
}

func TestOllamaChatFailures(t *testing.T) {

	t.Run("error inside a 200 body", func(t *testing.T) {
		srv, _ := scriptedServer(t, `{"error": "model 'x' not found"}`)
		_, err := NewOllamaChat(srv.URL).Execute(context.Background(),
			ExecutionRequest{Model: "x", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			}))
		t.Cleanup(srv.Close)

		_, err := NewOllamaChat(srv.URL).Execute(context.Background(),
			ExecutionRequest{Model: "x", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("done=false is an error", func(t *testing.T) {
		srv, _ := scriptedServer(t,
			`{"done": false, "message": {"role": "assistant", "content": "part"}}`)
		_, err := NewOllamaChat(srv.URL).Execute(context.Background(),
			ExecutionRequest{Model: "x", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error for incomplete response")
		}
	})

	t.Run("unknown done_reason maps to unknown", func(t *testing.T) {
		srv, _ := scriptedServer(t,
			`{"done": true, "done_reason": "load",
			  "message": {"role": "assistant", "content": ""}}`)
		resp, err := NewOllamaChat(srv.URL).Execute(context.Background(),
			ExecutionRequest{Model: "x", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Termination != TerminationUnknown {
			t.Errorf("termination = %s, want unknown (fail-closed)", resp.Termination)
		}
	})

	t.Run("malformed body", func(t *testing.T) {
		srv, _ := scriptedServer(t, `not json at all`)
		_, err := NewOllamaChat(srv.URL).Execute(context.Background(),
			ExecutionRequest{Model: "x", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("length termination", func(t *testing.T) {
		srv, _ := scriptedServer(t,
			`{"done": true, "done_reason": "length",
			  "message": {"role": "assistant", "content": "truncat"}}`)
		resp, err := NewOllamaChat(srv.URL).Execute(context.Background(),
			ExecutionRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Termination != TerminationLength {
			t.Errorf("termination = %s, want length", resp.Termination)
		}
	})
}
