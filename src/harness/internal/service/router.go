package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Router picks the best model for a task category based on benchmark
// validation results: for each category it selects the model with the
// highest average score across that category's benchmarks, using each
// model's most recent validated run.
type Router struct {
	// best maps category -> model name.
	best map[string]string

	// scores maps category -> model -> average score, kept for
	// reporting the routing table.
	scores map[string]map[string]int
}

var routerDateDir = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// NewRouter builds a routing table from a benchmark suite root. Variant
// results (model@variant) are excluded: routing targets real models.
func NewRouter(benchmarksRoot string) (*Router, error) {

	categories, err := loadCategories(benchmarksRoot)
	if err != nil {
		return nil, err
	}

	latest, err := latestValidatedRuns(benchmarksRoot)
	if err != nil {
		return nil, err
	}

	r := &Router{
		best:   map[string]string{},
		scores: map[string]map[string]int{},
	}

	type sums struct{ total, count int }
	perCategory := map[string]map[string]*sums{}

	for model, dir := range latest {

		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			return nil, err
		}

		for _, file := range files {

			data, err := os.ReadFile(file)
			if err != nil {
				return nil, err
			}

			var result struct {
				Benchmark string `json:"benchmark"`
				Score     int    `json:"score"`
			}

			if err := json.Unmarshal(data, &result); err != nil {
				continue
			}

			category, ok := categories[result.Benchmark]
			if !ok {
				continue
			}

			if perCategory[category] == nil {
				perCategory[category] = map[string]*sums{}
			}
			if perCategory[category][model] == nil {
				perCategory[category][model] = &sums{}
			}

			s := perCategory[category][model]
			s.total += result.Score
			s.count++
		}
	}

	for category, models := range perCategory {

		r.scores[category] = map[string]int{}

		bestModel, bestScore := "", -1

		for _, model := range sortedRouterKeys(models) {

			s := models[model]
			avg := s.total / s.count
			r.scores[category][model] = avg

			if avg > bestScore {
				bestModel, bestScore = model, avg
			}
		}

		r.best[category] = bestModel
	}

	return r, nil
}

// Route returns the best model for a category.
func (r *Router) Route(category string) (string, bool) {
	model, ok := r.best[category]
	return model, ok
}

// Table returns the routing table for diagnostics: category -> model ->
// average score.
func (r *Router) Table() map[string]map[string]int {
	return r.scores
}

// loadCategories maps benchmark IDs to their category from the
// definitions directory.
func loadCategories(root string) (map[string]string, error) {

	files, err := filepath.Glob(
		filepath.Join(root, "definitions", "*.json"),
	)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf(
			"no benchmark definitions found in %s",
			filepath.Join(root, "definitions"),
		)
	}

	categories := make(map[string]string, len(files))

	for _, file := range files {

		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var d struct {
			ID       string `json:"id"`
			Category string `json:"category"`
		}

		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}

		if d.ID != "" && d.Category != "" {
			categories[d.ID] = d.Category
		}
	}

	return categories, nil
}

// gatePassed reports whether a passing gate verdict admits the model's
// run on that date. The verdict is written by `themis-bench gate` at
// gate/<date>/<model>/verdict.json; the field names and digest
// algorithm mirror the gate package's Verdict and RunDigest, which are
// internal to benchmarks and not importable here. The verdict must
// name the same model and date as its location — a verdict copied
// elsewhere admits nothing — and its scores digest must match the run
// directory, so score files rewritten after gating are no longer
// admitted. Missing, unreadable, mismatched, or failing verdicts do
// not admit the run: the gate fails closed.
func gatePassed(root, date, model string) bool {

	path := filepath.Join(root, "gate", date, model, "verdict.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var v struct {
		Model        string `json:"model"`
		Current      string `json:"current"`
		Pass         bool   `json:"pass"`
		ScoresDigest string `json:"scores_digest"`
	}

	if err := json.Unmarshal(data, &v); err != nil {
		return false
	}

	if !v.Pass || v.Model != model || v.Current != date {
		return false
	}

	digest, err := runDigest(filepath.Join(root, "validation", date, model))
	if err != nil || digest != v.ScoresDigest {
		return false
	}

	return true
}

// runDigest fingerprints a run's validation score files: sha256 over
// each *.json file's basename and contents, sorted by basename. The
// algorithm is a contract with the gate package's RunDigest.
func runDigest(dir string) (string, error) {

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return "", err
	}

	sort.Strings(files)

	h := sha256.New()

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		h.Write([]byte(filepath.Base(file)))
		h.Write([]byte{'\n'})
		h.Write(data)
		h.Write([]byte{'\n'})
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// latestValidatedRuns maps each model to its most recent validation
// directory that a passing gate verdict has admitted. Runs without a
// passing verdict are invisible: a newer ungated run never shadows an
// older admitted one. Model names may contain slashes; prompt-variant
// series (model@variant) are skipped.
func latestValidatedRuns(root string) (map[string]string, error) {

	validationDir := filepath.Join(root, "validation")

	type run struct {
		date string
		dir  string
	}

	latest := map[string]run{}

	err := filepath.WalkDir(
		validationDir,
		func(path string, d fs.DirEntry, err error) error {

			if err != nil {
				if os.IsNotExist(err) && path == validationDir {
					return filepath.SkipAll
				}
				return err
			}

			if d.IsDir() || !strings.HasSuffix(path, ".json") {
				return nil
			}

			rel, err := filepath.Rel(validationDir, filepath.Dir(path))
			if err != nil {
				return err
			}

			parts := strings.Split(filepath.ToSlash(rel), "/")

			if len(parts) < 2 || !routerDateDir.MatchString(parts[0]) {
				return nil
			}

			date := parts[0]
			model := strings.Join(parts[1:], "/")

			if strings.Contains(model, "@") {
				return nil
			}

			if !gatePassed(root, date, model) {
				return nil
			}

			if existing, ok := latest[model]; !ok || date > existing.date {
				latest[model] = run{date: date, dir: filepath.Dir(path)}
			}

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	dirs := make(map[string]string, len(latest))
	for model, r := range latest {
		dirs[model] = r.dir
	}

	return dirs, nil
}

func sortedRouterKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
