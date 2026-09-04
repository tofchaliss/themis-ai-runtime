package model

import "github.com/tofchaliss/themis/internal/llm"

// The model layer's shared types are re-exported here so an external
// consumer needs only this package: the implementation lives in
// internal/llm pending its recorded relocation into this package
// (docs/architecture/repository-structure.md), and internal types must
// never appear in the seam's API surface. Discovered by the external
// consumer proof: without these aliases, a separate Go module cannot
// construct an ExecutionRequest or call Resolve at all.

// Options are the generation parameters (deterministic defaults:
// temperature 0, seed 42).
type Options = llm.Options

// Metrics are the normalized execution metrics a provider reports.
type Metrics = llm.Metrics

// Registry maps model names to provider configuration (models.json;
// API keys referenced only by environment-variable name).
type Registry = llm.Registry

// Entry describes how to reach one model.
type Entry = llm.Entry

// DefaultOptions returns the pinned deterministic generation options.
func DefaultOptions() Options { return llm.DefaultOptions() }

// LoadRegistry loads models.json from root; a missing file yields an
// empty registry (every name resolves to local Ollama).
func LoadRegistry(root string) (*Registry, error) { return llm.LoadRegistry(root) }
