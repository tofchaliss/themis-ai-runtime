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
		return NewOllamaChat(endpoint), wireModel, options, nil

	case "openai":
		if entry.Endpoint == "" {
			return nil, "", options, fmt.Errorf(
				"model %s: openai runtime requires an endpoint", name,
			)
		}
		return NewOpenAIChat(entry.Endpoint, apiKey), wireModel, options, nil

	default:
		return nil, "", options, fmt.Errorf(
			"model %s: unsupported runtime %q", name, entry.Runtime,
		)
	}
}
