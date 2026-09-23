package model

import (
	"fmt"

	"github.com/tofchaliss/themis/internal/llm"
)

// Resolve returns the chat runtime, wire model identifier, and
// generation options for the named model, using the same registry and
// resolution rules as the single-prompt layer (llm.Registry.Lookup):
// aliasing, per-model option overrides, env-var-only API keys, and
// local Ollama as the default for unregistered names.
func Resolve(reg *llm.Registry, name string) (Interface, string, llm.Options, error) {

	entry, wireModel, apiKey, options, err := reg.Lookup(name)
	if err != nil {
		return nil, "", options, err
	}

	switch entry.Runtime {

	case "", "ollama":
		endpoint := entry.Endpoint
		if endpoint == "" {
			endpoint = llm.DefaultOllamaEndpoint
		}
		rt := NewOllamaChat(endpoint)
		if rt.MaxResponseBytes, err = ceilingFor(name, entry.MaxResponseBytes); err != nil {
			return nil, "", options, err
		}
		return rt, wireModel, options, nil

	case "openai":
		if entry.Endpoint == "" {
			return nil, "", options, fmt.Errorf(
				"model %s: openai runtime requires an endpoint", name,
			)
		}
		rt := NewOpenAIChat(entry.Endpoint, apiKey)
		if rt.MaxResponseBytes, err = ceilingFor(name, entry.MaxResponseBytes); err != nil {
			return nil, "", options, err
		}
		return rt, wireModel, options, nil

	default:
		return nil, "", options, fmt.Errorf(
			"model %s: unsupported runtime %q", name, entry.Runtime,
		)
	}
}

// ceilingFor applies a registry entry's response ceiling: absent (0)
// keeps the compiled default; a narrower value applies; a wider one is
// refused — a governed input narrows a hard bound, never lifts it.
func ceilingFor(name string, entryMax int64) (int64, error) {
	switch {
	case entryMax == 0:
		return DefaultMaxResponseBytes, nil
	case entryMax < 0 || entryMax > DefaultMaxResponseBytes:
		return 0, fmt.Errorf("model %s: max_response_bytes %d must be within [1, %d] — the registry narrows the response ceiling, never widens it", name, entryMax, DefaultMaxResponseBytes)
	default:
		return entryMax, nil
	}
}
