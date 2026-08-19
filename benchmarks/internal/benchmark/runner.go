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

// RunAll executes every benchmark definition against the model and writes
// the raw runtime responses under runs/<date>/<model>/.
func RunAll(
	ctx context.Context,
	rt runtime.Runtime,
	modelName string,
	root string,
	date string,
) error {

	defs, err := LoadDefinitions(
		filepath.Join(root, "definitions"),
	)
	if err != nil {
		return err
	}

	if len(defs) == 0 {
		return fmt.Errorf("no benchmark definitions found")
	}

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	for _, d := range defs {

		if err := ctx.Err(); err != nil {
			return err
		}

		fmt.Printf("Running %s ...\n", d.ID)

		promptFile := filepath.Join(
			root,
			"prompts",
			d.Prompt,
		)

		prompt, err := os.ReadFile(promptFile)
		if err != nil {
			return fmt.Errorf("%s: read prompt: %w", d.ID, err)
		}

		resp, err := rt.Run(
			ctx,
			model.Request{
				Model:  modelName,
				Prompt: string(prompt),
			},
		)
		if err != nil {
			return fmt.Errorf("%s: %w", d.ID, err)
		}

		if err := WriteRawRun(
			root,
			date,
			modelName,
			d.ID,
			resp.Raw,
		); err != nil {
			return fmt.Errorf("%s: write run: %w", d.ID, err)
		}

		fmt.Printf("✓ %s completed\n", d.ID)
	}

	return nil
}
