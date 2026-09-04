package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tofchaliss/themis/internal/llm"
)

// OllamaChat speaks Ollama's /api/chat protocol with native tool calls.
type OllamaChat struct {
	Endpoint string
	Client   *http.Client
}

// NewOllamaChat returns an Ollama chat runtime for the given endpoint.
func NewOllamaChat(endpoint string) *OllamaChat {
	return &OllamaChat{
		Endpoint: endpoint,
		Client:   &http.Client{Timeout: llm.DefaultTimeout},
	}
}

func (o *OllamaChat) Name() string { return "ollama" }

// Wire shapes. Ollama nests generation options and returns tool-call
// arguments as a JSON object.

type ollamaChatMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaToolCall struct {
	Function ollamaToolFunction `json:"function"`
}

type ollamaToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ollamaTool struct {
	Type     string             `json:"type"`
	Function ollamaToolDeclared `json:"function"`
}

type ollamaToolDeclared struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ollamaChatResponse struct {
	Model      string            `json:"model"`
	Message    ollamaChatMessage `json:"message"`
	Done       bool              `json:"done"`
	DoneReason string            `json:"done_reason"`

	PromptEvalCount    int   `json:"prompt_eval_count"`
	EvalCount          int   `json:"eval_count"`
	LoadDuration       int64 `json:"load_duration"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	EvalDuration       int64 `json:"eval_duration"`
	TotalDuration      int64 `json:"total_duration"`

	Error string `json:"error"`
}

const nsToMS = 1e6

func (o *OllamaChat) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResponse, error) {

	messages := make([]ollamaChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		wire := ollamaChatMessage{Role: string(m.Role), Content: m.Content}
		for _, tc := range m.ToolCalls {
			wire.ToolCalls = append(wire.ToolCalls, ollamaToolCall{
				Function: ollamaToolFunction{Name: tc.Name, Arguments: tc.Arguments},
			})
		}
		messages = append(messages, wire)
	}

	body := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   false,
		"options": map[string]any{
			"temperature": req.Options.Temperature,
			"seed":        req.Options.Seed,
		},
	}
	if len(req.Tools) > 0 {
		tools := make([]ollamaTool, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, ollamaTool{
				Type: "function",
				Function: ollamaToolDeclared{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			})
		}
		body["tools"] = tools
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost, o.Endpoint+"/api/chat", bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: llm.DefaultTimeout}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama chat: status %d: %s", resp.StatusCode, truncate(raw, 512))
	}

	var r ollamaChatResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("ollama chat: parse response: %w", err)
	}
	// Ollama can report errors inside a 200 body.
	if r.Error != "" {
		return nil, fmt.Errorf("ollama chat: %s", truncate([]byte(r.Error), 512))
	}
	// A response that is not done is a partial: fail fast rather than
	// report it upward as a clean completion.
	if !r.Done {
		return nil, fmt.Errorf("ollama chat: response is incomplete (done=false)")
	}

	out := &ExecutionResponse{
		Content: r.Message.Content,
		Usage: llm.Metrics{
			PromptTokens:     r.PromptEvalCount,
			CompletionTokens: r.EvalCount,
			TotalTokens:      r.PromptEvalCount + r.EvalCount,
			LoadTimeMS:       float64(r.LoadDuration) / nsToMS,
			PromptTimeMS:     float64(r.PromptEvalDuration) / nsToMS,
			GenerationTimeMS: float64(r.EvalDuration) / nsToMS,
			TotalTimeMS:      float64(r.TotalDuration) / nsToMS,
		},
		Identity: Identity{WireModel: req.Model, Runtime: o.Name()},
		Provenance: Provenance{
			Endpoint: o.Endpoint,
			Options:  req.Options,
			Raw:      raw,
		},
		Termination: TerminationStop,
	}
	if out.Usage.GenerationTimeMS > 0 {
		out.Usage.TokensPerSecond = float64(out.Usage.CompletionTokens) /
			(out.Usage.GenerationTimeMS / 1000)
	}

	for i, tc := range r.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        fmt.Sprintf("call_%d", i),
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	// Fail-closed mapping: only recognized clean stops stay clean.
	switch {
	case len(out.ToolCalls) > 0:
		out.Termination = TerminationToolCalls
	case r.DoneReason == "stop" || r.DoneReason == "":
		out.Termination = TerminationStop
	case r.DoneReason == "length":
		out.Termination = TerminationLength
	default:
		out.Termination = TerminationUnknown
	}

	return out, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
