// Package gate compares validation scores between two dates and fails
// when a model has regressed, for use as a CI quality gate.
package gate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tofchaliss/themis/benchmarks/internal/validator"
)

// Delta is the score change of one benchmark between two dates.
type Delta struct {
	Benchmark string
	Baseline  int
	Current   int
}

// Drop returns how many points the score fell (negative = improved).
func (d Delta) Drop() int {
	return d.Baseline - d.Current
}

// Result of comparing a model's scores against a baseline date.
type Result struct {
	Model    string
	Baseline string
	Current  string

	Deltas []Delta

	// Missing lists benchmarks validated at the baseline but absent
	// from the current run. Always a failure: silently dropping a
	// benchmark must not pass the gate.
	Missing []string

	// Added lists benchmarks present now but not at the baseline.
	// Informational only.
	Added []string

	BaselineAverage int
	CurrentAverage  int
}

// AverageDrop returns how many points the average score fell.
func (r Result) AverageDrop() int {
	return r.BaselineAverage - r.CurrentAverage
}

// Pass reports whether the gate passes with the given tolerance on the
// average-score drop.
func (r Result) Pass(maxDrop int) bool {
	return len(r.Missing) == 0 && r.AverageDrop() <= maxDrop
}

// Compare loads validation scores for the model on both dates.
func Compare(root, model, baselineDate, currentDate string) (Result, error) {

	baseline, err := loadScores(root, baselineDate, model)
	if err != nil {
		return Result{}, fmt.Errorf("baseline: %w", err)
	}

	current, err := loadScores(root, currentDate, model)
	if err != nil {
		return Result{}, fmt.Errorf("current: %w", err)
	}

	r := Result{
		Model:    model,
		Baseline: baselineDate,
		Current:  currentDate,
	}

	for _, id := range sortedKeys(baseline) {

		score, ok := current[id]
		if !ok {
			r.Missing = append(r.Missing, id)
			continue
		}

		r.Deltas = append(r.Deltas, Delta{
			Benchmark: id,
			Baseline:  baseline[id],
			Current:   score,
		})
	}

	for _, id := range sortedKeys(current) {
		if _, ok := baseline[id]; !ok {
			r.Added = append(r.Added, id)
		}
	}

	r.BaselineAverage = average(baseline)
	r.CurrentAverage = average(current)

	return r, nil
}

// Report renders the comparison as text.
func Report(r Result, maxDrop int) string {

	var b strings.Builder

	fmt.Fprintf(&b, "Gate: %s\n", r.Model)
	fmt.Fprintf(&b, "Baseline %s: %d%%   Current %s: %d%%   (max allowed drop: %d)\n\n",
		r.Baseline, r.BaselineAverage,
		r.Current, r.CurrentAverage,
		maxDrop,
	)

	for _, d := range r.Deltas {

		marker := " "
		switch {
		case d.Drop() > 0:
			marker = "▼"
		case d.Drop() < 0:
			marker = "▲"
		}

		fmt.Fprintf(&b, "%s %s: %d%% -> %d%%\n",
			marker, d.Benchmark, d.Baseline, d.Current)
	}

	for _, id := range r.Missing {
		fmt.Fprintf(&b, "✗ %s: present in baseline, missing now\n", id)
	}

	for _, id := range r.Added {
		fmt.Fprintf(&b, "+ %s: new since baseline\n", id)
	}

	fmt.Fprintf(&b, "\n")

	if r.Pass(maxDrop) {
		fmt.Fprintf(&b, "PASS (average drop %d ≤ %d)\n", r.AverageDrop(), maxDrop)
	} else if len(r.Missing) > 0 {
		fmt.Fprintf(&b, "FAIL (%d benchmark(s) missing)\n", len(r.Missing))
	} else {
		fmt.Fprintf(&b, "FAIL (average drop %d > %d)\n", r.AverageDrop(), maxDrop)
	}

	return b.String()
}

func loadScores(root, date, model string) (map[string]int, error) {

	dir := filepath.Join(root, "validation", date, model)

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no validation results in %s", dir)
	}

	scores := make(map[string]int, len(files))

	for _, file := range files {

		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var result validator.Result

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		if result.Benchmark == "" {
			return nil, fmt.Errorf("%s: missing benchmark id", file)
		}

		scores[result.Benchmark] = result.Score
	}

	return scores, nil
}

func average(scores map[string]int) int {
	if len(scores) == 0 {
		return 0
	}
	total := 0
	for _, s := range scores {
		total += s
	}
	return total / len(scores)
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
