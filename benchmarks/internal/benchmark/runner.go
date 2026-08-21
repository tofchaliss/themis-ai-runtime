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

	for _, d := range defs {

		if err := ctx.Err(); err != nil {
			return err
		}

		fmt.Printf("Running %s ...\n", d.ID)

		promptFile := filepath.Join(
			cfg.Root,
			"prompts",
			d.Prompt,
		)

		prompt, err := os.ReadFile(promptFile)
		if err != nil {
			return fmt.Errorf("%s: read prompt: %w", d.ID, err)
		}

		resp, err := cfg.Runtime.Run(
			ctx,
			model.Request{
				Model:   cfg.Model,
				Prompt:  string(prompt),
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
