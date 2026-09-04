package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tofchaliss/themis/internal/llm"
)

// OpenAIChat speaks the OpenAI-compatible /chat/completions protocol
// with native tool calls — the same wire contract vLLM, llama.cpp
// server, LM Studio, and hosted providers implement.
type OpenAIChat struct {
	Endpoint string
	APIKey   string `json:"-"` // secret: never serialized
	Client   *http.Client
}

// NewOpenAIChat returns an OpenAI-compatible chat runtime.
func NewOpenAIChat(endpoint, apiKey string) *OpenAIChat {
	return &OpenAIChat{
		Endpoint: endpoint,
		APIKey:   apiKey,
		Client:   &http.Client{Timeout: llm.DefaultTimeout},
	}
}

func (o *OpenAIChat) Name() string { return "openai" }

// Wire shapes. OpenAI takes generation options at the top level and
// transports tool-call arguments as a JSON-encoded string.

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolDeclared `json:"function"`
}

type openAIToolDeclared struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAIChat) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResponse, error) {

	messages := make([]openAIMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		wire := openAIMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			wire.ToolCalls = append(wire.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openAIToolFunction{
					Name:      tc.Name,
					Arguments: string(tc.Arguments),
				},
			})
		}
		messages = append(messages, wire)
	}

	body := map[string]any{
		"model":       req.Model,
		"messages":    messages,
		"temperature": req.Options.Temperature,
		"seed":        req.Options.Seed,
		"stream":      false,
	}
	if len(req.Tools) > 0 {
		tools := make([]openAITool, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, openAITool{
				Type: "function",
				Function: openAIToolDeclared{
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
		ctx, http.MethodPost, o.Endpoint+"/chat/completions", bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}

	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: llm.DefaultTimeout}
	}

	start := time.Now()

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
		return nil, fmt.Errorf("openai chat: status %d: %s", resp.StatusCode, truncate(raw, 512))
	}

	var r openAIChatResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("openai chat: parse response: %w", err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("openai chat: %s", truncate([]byte(r.Error.Message), 512))
	}
	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("openai chat: response contains no choices")
	}

	choice := r.Choices[0]
	wallMS := float64(time.Since(start).Milliseconds())

	out := &ExecutionResponse{
		Content: choice.Message.Content,
		Usage: llm.Metrics{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
			GenerationTimeMS: wallMS,
			TotalTimeMS:      wallMS,
		},
		Identity: Identity{WireModel: req.Model, Runtime: o.Name()},
		Provenance: Provenance{
			Endpoint: o.Endpoint,
			Options:  req.Options,
			Raw:      raw,
		},
		Termination: TerminationStop,
	}
	if wallMS > 0 {
		out.Usage.TokensPerSecond = float64(out.Usage.CompletionTokens) / (wallMS / 1000)
	}

	for _, tc := range choice.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: rawOrQuoted(tc.Function.Arguments),
		})
	}

	// Fail-closed mapping: only recognized clean stops stay clean.
	switch {
	case len(out.ToolCalls) > 0 || choice.FinishReason == "tool_calls":
		out.Termination = TerminationToolCalls
	case choice.FinishReason == "stop":
		out.Termination = TerminationStop
	case choice.FinishReason == "length":
		out.Termination = TerminationLength
	default:
		out.Termination = TerminationUnknown
	}

	return out, nil
}

// rawOrQuoted transports a provider's argument string verbatim when it
// is valid JSON, and re-encodes it as a JSON string otherwise — so a
// provider cannot make the response unserializable and break the audit
// record for exactly the suspicious invocation.
func rawOrQuoted(s string) json.RawMessage {
	if len(s) > 0 && json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	quoted, _ := json.Marshal(s)
	return quoted
}
