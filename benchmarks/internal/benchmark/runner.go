package benchmark

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tofchaliss/themis/benchmarks/internal/model"
	"github.com/tofchaliss/themis/benchmarks/internal/runtime"
)

// RunConfig describes one benchmark run of a model.
type RunConfig struct {
	Runtime runtime.Runtime

	// Name is what the user calls the model; it names the output
	// directories and is shared by the later pipeline stages.
	Name string

	// Model is the identifier sent to the runtime (usually equal to
	// Name unless the registry aliases it).
	Model string

	// Variant selects a prompt variant (prompts/variants/<variant>/)
	// for A/B testing. Empty means the base prompts.
	Variant string

	Options model.Options

	Root string

	// Date defaults to today when empty.
	Date string
}

// RunAll executes every benchmark definition and writes run envelopes
// under runs/<date>/<name>/.
func RunAll(ctx context.Context, cfg RunConfig) error {

	defs, err := LoadDefinitions(
		filepath.Join(cfg.Root, "definitions"),
	)
	if err != nil {
		return err
	}

	if len(defs) == 0 {
		return fmt.Errorf("no benchmark definitions found")
	}

	date := cfg.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	manifest := Manifest{
		Name:        cfg.Name,
		Model:       cfg.Model,
		Runtime:     cfg.Runtime.Name(),
		Variant:     cfg.Variant,
		Date:        date,
		StartedAt:   time.Now().UTC(),
		Options:     cfg.Options,
		Prompts:     map[string]string{},
		Definitions: map[string]string{},
	}

	loader, err := NewPrompts(cfg.Root, cfg.Variant)
	if err != nil {
		return err
	}

	// Preload every prompt so missing files or broken templates fail
	// before any model call, and so the manifest records the exact
	// (rendered) inputs of the whole run.
	prompts := make(map[string][]byte, len(defs))

	for _, d := range defs {

		prompt, err := loader.Load(d.Prompt)
		if err != nil {
			return fmt.Errorf("%s: load prompt: %w", d.ID, err)
		}

		prompts[d.ID] = prompt
		manifest.Prompts[d.ID] = hashBytes(prompt)

		definitionFile := filepath.Join(
			cfg.Root,
			"definitions",
			d.ID+".json",
		)

		if data, err := os.ReadFile(definitionFile); err == nil {
			manifest.Definitions[d.ID] = hashBytes(data)
		}
	}

	if err := WriteManifest(cfg.Root, date, cfg.Name, manifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	for _, d := range defs {

		if err := ctx.Err(); err != nil {
			return err
		}

		fmt.Printf("Running %s ...\n", d.ID)

		resp, err := cfg.Runtime.Run(
			ctx,
			model.Request{
				Model:   cfg.Model,
				Prompt:  string(prompts[d.ID]),
				Options: cfg.Options,
			},
		)
		if err != nil {
			return fmt.Errorf("%s: %w", d.ID, err)
		}

		record := model.RunRecord{
			Benchmark: d.ID,
			Model:     cfg.Name,
			Runtime:   resp.Runtime,
			Answer:    resp.Answer,
			Options:   cfg.Options,
			Metrics:   resp.Metrics,
			Raw:       resp.Raw,
		}

		if err := WriteRun(cfg.Root, date, cfg.Name, record); err != nil {
			return fmt.Errorf("%s: write run: %w", d.ID, err)
		}

		fmt.Printf("✓ %s completed\n", d.ID)
	}

	return nil
}
