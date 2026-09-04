// Package model is the Harness Model Interface: the single seam through
// which the Harness reaches a reasoning model. Everything above this
// package is provider-neutral — nothing may know which provider is
// configured (DEC-05), and the model is never granted authority framing
// (DEC-06). Model output crossing this seam is untrusted; tool-call
// arguments are transported verbatim and validated by the Tool
// Interface, never here.
//
// The package coexists with the single-prompt internal/llm layer, which
// the benchmark suite keeps using unchanged; new Harness layers build
// against this interface only.
package model

import (
	"context"
	"encoding/json"

	"github.com/tofchaliss/themis/internal/llm"
)

// Role identifies a conversation participant.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolCall is a model's request to invoke a tool. Arguments are the
// raw JSON the model produced — a proposal, not a command: the Tool
// Interface parses them strictly and the policy layer authorizes them.
type ToolCall struct {
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Message is one turn of a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`

	// ToolCallID links a RoleTool message to the assistant tool call
	// it answers.
	ToolCallID string `json:"tool_call_id,omitempty"`

	// ToolCalls carries an assistant turn's tool requests when the
	// conversation is replayed back to the model.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDef declares a tool the model may request. Parameters is a JSON
// Schema object.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ExecutionRequest is one model invocation.
type ExecutionRequest struct {
	Model    string      `json:"model"`
	Messages []Message   `json:"messages"`
	Tools    []ToolDef   `json:"tools,omitempty"`
	Options  llm.Options `json:"options"`
}

// Termination states why the model stopped. It is a provider-controlled
// signal and maps fail-closed: anything a provider reports that is not
// an explicitly recognized clean stop becomes TerminationUnknown, which
// upper layers must treat as not-clean.
type Termination string

const (
	TerminationStop      Termination = "stop"
	TerminationToolCalls Termination = "tool_calls"
	TerminationLength    Termination = "length"
	TerminationUnknown   Termination = "unknown"
)

// Identity records which model actually answered. Providers fill
// WireModel and Runtime; Name is the registry name the caller resolved,
// filled by the caller of Resolve (providers cannot know it).
type Identity struct {
	Name      string `json:"name,omitempty"`
	WireModel string `json:"wire_model"` // identifier sent to the provider

	// Runtime names the provider implementation — audit metadata only;
	// per DEC-05 no layer above this seam may branch on it.
	Runtime string `json:"runtime"`
}

// Provenance records what produced the response, following the
// RunRecord pattern: the raw provider payload is preserved for audit.
// This deliberately diverges from llm.Response (whose Raw is excluded
// from serialization) — at this seam the raw payload is the audit
// record. It is untrusted, injection-bearing content: stores and
// displays must treat it as hostile data, and it never contains the
// API key (which travels only in the Authorization header).
type Provenance struct {
	Endpoint string          `json:"endpoint"`
	Options  llm.Options     `json:"options"`
	Raw      json.RawMessage `json:"raw,omitempty"`
}

// ExecutionResponse is the provider-neutral result of one invocation.
type ExecutionResponse struct {
	Content     string      `json:"content"`
	ToolCalls   []ToolCall  `json:"tool_calls,omitempty"`
	Usage       llm.Metrics `json:"usage"`
	Identity    Identity    `json:"identity"`
	Provenance  Provenance  `json:"provenance"`
	Termination Termination `json:"termination"`
}

// Interface is the Model Interface. Implementations translate the
// neutral request/response into one provider's wire protocol and
// nothing more: no retries, no policy, no interpretation.
type Interface interface {
	Name() string
	Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResponse, error)
}
