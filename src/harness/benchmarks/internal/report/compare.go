package report

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// RunRef identifies one evaluated run: a model on a date.
type RunRef struct {
	Date  string
	Model string
}

// ModelComparison holds every run of one model, ordered by date.
type ModelComparison struct {
	Model string
	Runs  []Report
}

// Latest returns the most recent run.
func (m ModelComparison) Latest() Report {
	return m.Runs[len(m.Runs)-1]
}

// Comparison is a cross-model, cross-date view of all results.
type Comparison struct {
	Models []ModelComparison

	// Skipped lists runs that could not be loaded, with the reason.
	Skipped []string
}

var dateDir = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// DiscoverRuns finds every (date, model) pair with evaluated responses
// under responses/. Model names may contain slashes, so a model is the
// full path between the date directory and the response files.
func DiscoverRuns(root string) ([]RunRef, error) {

	responsesDir := filepath.Join(root, "responses")

	seen := map[RunRef]bool{}

	err := filepath.WalkDir(
		responsesDir,
		func(path string, d fs.DirEntry, err error) error {

			if err != nil {
				if os.IsNotExist(err) && path == responsesDir {
					return filepath.SkipAll
				}
				return err
			}

			if d.IsDir() || !strings.HasSuffix(path, ".json") {
				return nil
			}

			rel, err := filepath.Rel(responsesDir, filepath.Dir(path))
			if err != nil {
				return err
			}

			parts := strings.Split(filepath.ToSlash(rel), "/")

			if len(parts) < 2 || !dateDir.MatchString(parts[0]) {
				return nil
			}

			seen[RunRef{
				Date:  parts[0],
				Model: strings.Join(parts[1:], "/"),
			}] = true

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	refs := make([]RunRef, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Model != refs[j].Model {
			return refs[i].Model < refs[j].Model
		}
		return refs[i].Date < refs[j].Date
	})

	return refs, nil
}

// Compare loads every discovered run and groups the results by model.
// Runs that fail to load are skipped and reported in Comparison.Skipped.
func Compare(root string) (Comparison, error) {

	refs, err := DiscoverRuns(root)
	if err != nil {
		return Comparison{}, err
	}

	if len(refs) == 0 {
		return Comparison{}, fmt.Errorf(
			"no evaluated runs found under %s",
			filepath.Join(root, "responses"),
		)
	}

	var c Comparison

	byModel := map[string]*ModelComparison{}
	var order []string

	for _, ref := range refs {

		benchmarks, err := LoadResponses(root, ref.Date, ref.Model)
		if err != nil {
			c.Skipped = append(c.Skipped, fmt.Sprintf(
				"%s @ %s: %v", ref.Model, ref.Date, err,
			))
			continue
		}

		report := Generate(ref.Model, ref.Date, benchmarks)

		mc, ok := byModel[ref.Model]
		if !ok {
			mc = &ModelComparison{Model: ref.Model}
			byModel[ref.Model] = mc
			order = append(order, ref.Model)
		}

		mc.Runs = append(mc.Runs, report)
	}

	for _, model := range order {
		c.Models = append(c.Models, *byModel[model])
	}

	if len(c.Models) == 0 {
		return c, fmt.Errorf(
			"no runs could be loaded (%d skipped)",
			len(c.Skipped),
		)
	}

	return c, nil
}

// ComparisonMarkdown renders the comparison as Markdown: a summary of
// each model's latest run, a per-benchmark score matrix, and score
// history for models with more than one run.
func ComparisonMarkdown(c Comparison) string {

	var b strings.Builder

	fmt.Fprintf(&b, "# Themis Model Comparison\n\n")

	// Latest-run summary.
	fmt.Fprintf(&b, "## Models (latest run)\n\n")
	fmt.Fprintf(&b, "| Model | Date | Benchmarks | Validated | Avg Score | Avg TPS | Avg Generation (ms) | Total Tokens |\n")
	fmt.Fprintf(&b, "|-------|------|-----------:|----------:|----------:|--------:|--------------------:|-------------:|\n")

	for _, m := range c.Models {
		latest := m.Latest()
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d%% | %.2f | %.2f | %d |\n",
			m.Model,
			latest.Date,
			latest.Benchmarks,
			latest.Validated,
			latest.AverageScore,
			latest.AverageTPS,
			latest.AverageGenerationMS,
			latest.TotalTokens,
		)
	}

	// Per-benchmark score matrix over each model's latest run.
	fmt.Fprintf(&b, "\n## Benchmark Scores (latest run)\n\n")

	ids := benchmarkIDs(c)

	fmt.Fprintf(&b, "| Benchmark |")
	for _, m := range c.Models {
		fmt.Fprintf(&b, " %s |", m.Model)
	}
	fmt.Fprintf(&b, "\n|-----------|")
	for range c.Models {
		fmt.Fprintf(&b, "----:|")
	}
	fmt.Fprintf(&b, "\n")

	for _, id := range ids {
		fmt.Fprintf(&b, "| %s |", id)

		for _, m := range c.Models {
			fmt.Fprintf(&b, " %s |", scoreCell(m.Latest(), id))
		}
		fmt.Fprintf(&b, "\n")
	}

	// History for models benchmarked on multiple dates.
	history := false
	for _, m := range c.Models {
		if len(m.Runs) > 1 {
			history = true
			break
		}
	}

	if history {
		fmt.Fprintf(&b, "\n## Score History\n\n")

		for _, m := range c.Models {
			if len(m.Runs) < 2 {
				continue
			}

			fmt.Fprintf(&b, "### %s\n\n", m.Model)
			fmt.Fprintf(&b, "| Date | Benchmarks | Validated | Avg Score | Avg TPS |\n")
			fmt.Fprintf(&b, "|------|-----------:|----------:|----------:|--------:|\n")

			for _, run := range m.Runs {
				fmt.Fprintf(&b, "| %s | %d | %d | %d%% | %.2f |\n",
					run.Date,
					run.Benchmarks,
					run.Validated,
					run.AverageScore,
					run.AverageTPS,
				)
			}
			fmt.Fprintf(&b, "\n")
		}
	}

	if len(c.Skipped) > 0 {
		fmt.Fprintf(&b, "\n## Skipped Runs\n\n")
		for _, s := range c.Skipped {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}

	return b.String()
}

// WriteComparison writes the comparison to reports/comparison.md.
func WriteComparison(root string, c Comparison) (string, error) {

	outputFile := filepath.Join(root, "reports", "comparison.md")

	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(
		outputFile,
		[]byte(ComparisonMarkdown(c)),
		0644,
	); err != nil {
		return "", err
	}

	return outputFile, nil
}

// benchmarkIDs returns the sorted union of benchmark IDs across every
// model's latest run.
func benchmarkIDs(c Comparison) []string {

	seen := map[string]bool{}

	for _, m := range c.Models {
		for _, r := range m.Latest().Results {
			seen[r.Benchmark] = true
		}
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	return ids
}

// scoreCell renders one matrix cell: the score, "-" for unvalidated,
// or "·" when the model has no result for the benchmark.
func scoreCell(r Report, benchmark string) string {

	for _, result := range r.Results {
		if result.Benchmark != benchmark {
			continue
		}
		if !result.Validated {
			return "-"
		}
		return fmt.Sprintf("%d%%", result.Score)
	}

	return "·"
}
