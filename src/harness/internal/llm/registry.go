package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RegistryFile is the optional per-suite model registry.
const RegistryFile = "models.json"

const DefaultOllamaEndpoint = "http://localhost:11434"

// Entry describes how to reach one model.
type Entry struct {
	// Runtime selects the implementation: "ollama" (default) or "openai".
	Runtime string `json:"runtime"`

	// Endpoint overrides the runtime's API base URL.
	Endpoint string `json:"endpoint"`

	// Model is the identifier sent to the runtime when it differs from
	// the registry name (e.g. an alias like "gpt" -> "gpt-4o").
	Model string `json:"model"`

	// APIKeyEnv names the environment variable holding the API key.
	APIKeyEnv string `json:"api_key_env"`

	// Temperature and Seed override the deterministic defaults when set.
	Temperature *float64 `json:"temperature"`
	Seed        *int     `json:"seed"`
}

// Registry maps model names to runtime configuration.
type Registry struct {
	Models map[string]Entry `json:"models"`

	// Defaults applies to models not listed in Models.
	Defaults Entry `json:"defaults"`
}

// LoadRegistry reads models.json from root. A missing file yields an
// empty registry, which resolves every model to local Ollama.
func LoadRegistry(root string) (*Registry, error) {

	path := filepath.Join(root, RegistryFile)

	data, err := os.ReadFile(path)

	if errors.Is(err, os.ErrNotExist) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, err
	}

	var r Registry

	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &r, nil
}

// Resolve returns the runtime, the model identifier to send to it, and
// the generation options for the named model.
func (r *Registry) Resolve(name string) (Runtime, string, Options, error) {

	entry, ok := r.Models[name]
	if !ok {
		entry = r.Defaults
	}

	modelName := name
	if entry.Model != "" {
		modelName = entry.Model
	}

	options := DefaultOptions()
	if entry.Temperature != nil {
		options.Temperature = *entry.Temperature
	}
	if entry.Seed != nil {
		options.Seed = *entry.Seed
	}

	apiKey := ""
	if entry.APIKeyEnv != "" {
		apiKey = os.Getenv(entry.APIKeyEnv)
		if apiKey == "" {
			return nil, "", options, fmt.Errorf(
				"model %s: environment variable %s is not set",
				name,
				entry.APIKeyEnv,
			)
		}
	}

	switch entry.Runtime {

	case "", "ollama":
		endpoint := entry.Endpoint
		if endpoint == "" {
			endpoint = DefaultOllamaEndpoint
		}
		return NewOllama(endpoint), modelName, options, nil

	case "openai":
		if entry.Endpoint == "" {
			return nil, "", options, fmt.Errorf(
				"model %s: openai runtime requires an endpoint",
				name,
			)
		}
		return NewOpenAI(entry.Endpoint, apiKey), modelName, options, nil

	default:
		return nil, "", options, fmt.Errorf(
			"model %s: unsupported runtime %q",
			name,
			entry.Runtime,
		)
	}
}
