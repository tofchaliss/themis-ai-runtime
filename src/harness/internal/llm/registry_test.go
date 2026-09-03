package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRegistry(t *testing.T, content string) string {
	t.Helper()

	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, RegistryFile),
		[]byte(content),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestRegistry(t *testing.T) {

	t.Run("missing file resolves to local ollama", func(t *testing.T) {
		registry, err := LoadRegistry(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		rt, modelName, options, err := registry.Resolve("anymodel")
		if err != nil {
			t.Fatal(err)
		}

		ollama, ok := rt.(*Ollama)
		if !ok {
			t.Fatalf("runtime = %T, want *Ollama", rt)
		}
		if ollama.Endpoint != DefaultOllamaEndpoint {
			t.Errorf("endpoint = %s", ollama.Endpoint)
		}
		if modelName != "anymodel" {
			t.Errorf("model = %s, want anymodel", modelName)
		}
		if options.Temperature != 0 || options.Seed != 42 {
			t.Errorf("options = %+v, want deterministic defaults", options)
		}
	})

	t.Run("openai entry with alias and key", func(t *testing.T) {
		root := writeRegistry(t, `{
			"models": {
				"gpt": {
					"runtime": "openai",
					"endpoint": "https://api.openai.com/v1",
					"model": "gpt-4o",
					"api_key_env": "TEST_OPENAI_KEY"
				}
			}
		}`)

		t.Setenv("TEST_OPENAI_KEY", "sk-test")

		registry, err := LoadRegistry(root)
		if err != nil {
			t.Fatal(err)
		}

		rt, modelName, _, err := registry.Resolve("gpt")
		if err != nil {
			t.Fatal(err)
		}

		openai, ok := rt.(*OpenAI)
		if !ok {
			t.Fatalf("runtime = %T, want *OpenAI", rt)
		}
		if openai.APIKey != "sk-test" {
			t.Errorf("APIKey = %q", openai.APIKey)
		}
		if modelName != "gpt-4o" {
			t.Errorf("model = %s, want gpt-4o (alias)", modelName)
		}
	})

	t.Run("missing API key env is an error", func(t *testing.T) {
		root := writeRegistry(t, `{
			"models": {
				"gpt": {
					"runtime": "openai",
					"endpoint": "https://x/v1",
					"api_key_env": "DEFINITELY_NOT_SET_12345"
				}
			}
		}`)

		registry, err := LoadRegistry(root)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, _, err := registry.Resolve("gpt"); err == nil {
			t.Error("expected error for unset API key env")
		}
	})

	t.Run("options overrides", func(t *testing.T) {
		root := writeRegistry(t, `{
			"models": {
				"warm": {"temperature": 0.7, "seed": 7}
			}
		}`)

		registry, err := LoadRegistry(root)
		if err != nil {
			t.Fatal(err)
		}

		_, _, options, err := registry.Resolve("warm")
		if err != nil {
			t.Fatal(err)
		}
		if options.Temperature != 0.7 || options.Seed != 7 {
			t.Errorf("options = %+v, want 0.7/7", options)
		}
	})

	t.Run("unsupported runtime is an error", func(t *testing.T) {
		root := writeRegistry(t, `{
			"models": {"x": {"runtime": "quantum"}}
		}`)

		registry, err := LoadRegistry(root)
		if err != nil {
			t.Fatal(err)
		}

		if _, _, _, err := registry.Resolve("x"); err == nil {
			t.Error("expected error for unsupported runtime")
		}
	})

	t.Run("malformed registry is an error", func(t *testing.T) {
		root := writeRegistry(t, `{`)

		if _, err := LoadRegistry(root); err == nil {
			t.Error("expected error for malformed registry")
		}
	})
}
