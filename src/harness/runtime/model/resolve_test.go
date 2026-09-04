package model

import (
	"strings"
	"testing"

	"github.com/tofchaliss/themis/internal/llm"
)

func TestResolve(t *testing.T) {

	t.Run("unregistered name defaults to local ollama", func(t *testing.T) {
		chat, wire, opts, err := Resolve(&llm.Registry{}, "some-model")
		if err != nil {
			t.Fatal(err)
		}
		oc, ok := chat.(*OllamaChat)
		if !ok || oc.Endpoint != llm.DefaultOllamaEndpoint {
			t.Errorf("runtime = %#v", chat)
		}
		if wire != "some-model" || opts.Temperature != 0 || opts.Seed != 42 {
			t.Errorf("wire/opts = %s / %+v", wire, opts)
		}
	})

	t.Run("openai entry with alias and env key", func(t *testing.T) {
		t.Setenv("TEST_MODEL_KEY", "k")
		reg := &llm.Registry{Models: map[string]llm.Entry{
			"gpt": {Runtime: "openai", Endpoint: "http://api", Model: "gpt-4o", APIKeyEnv: "TEST_MODEL_KEY"},
		}}
		chat, wire, _, err := Resolve(reg, "gpt")
		if err != nil {
			t.Fatal(err)
		}
		oc, ok := chat.(*OpenAIChat)
		if !ok || oc.APIKey != "k" || wire != "gpt-4o" {
			t.Errorf("runtime/wire = %#v / %s", chat, wire)
		}
	})

	t.Run("openai without endpoint fails", func(t *testing.T) {
		reg := &llm.Registry{Models: map[string]llm.Entry{
			"x": {Runtime: "openai"},
		}}
		if _, _, _, err := Resolve(reg, "x"); err == nil ||
			!strings.Contains(err.Error(), "requires an endpoint") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("missing env var fails naming it", func(t *testing.T) {
		reg := &llm.Registry{Models: map[string]llm.Entry{
			"x": {Runtime: "openai", Endpoint: "http://api", APIKeyEnv: "DEFINITELY_UNSET_VAR_42"},
		}}
		if _, _, _, err := Resolve(reg, "x"); err == nil ||
			!strings.Contains(err.Error(), "DEFINITELY_UNSET_VAR_42") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("unsupported runtime fails", func(t *testing.T) {
		reg := &llm.Registry{Models: map[string]llm.Entry{
			"x": {Runtime: "quantum"},
		}}
		if _, _, _, err := Resolve(reg, "x"); err == nil ||
			!strings.Contains(err.Error(), "unsupported runtime") {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("per-model option overrides apply", func(t *testing.T) {
		temp := 0.7
		seed := 7
		reg := &llm.Registry{Models: map[string]llm.Entry{
			"warm": {Temperature: &temp, Seed: &seed},
		}}
		_, _, opts, err := Resolve(reg, "warm")
		if err != nil {
			t.Fatal(err)
		}
		if opts.Temperature != 0.7 || opts.Seed != 7 {
			t.Errorf("opts = %+v", opts)
		}
	})
}
