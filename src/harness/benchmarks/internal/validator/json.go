package validator

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/tofchaliss/themis/internal/llm"
)

// ValidateJSON validates a model response by comparing the JSON object in
// the answer against the expected ground-truth object field by field.
//
// Each top-level field of the expected object counts as one check. A check
// passes when the answer contains the same field with a deep-equal value
// (string values are compared after trimming surrounding whitespace).
func ValidateJSON(expected *Expected, answer string) (Result, error) {

	result := Result{
		Benchmark:  expected.Benchmark,
		Missing:    []string{},
		Violations: []string{},
	}

	var want map[string]any

	if err := json.Unmarshal(expected.Expected, &want); err != nil {
		return result, fmt.Errorf(
			"%s: invalid expected JSON: %w",
			expected.Benchmark,
			err,
		)
	}

	if len(want) == 0 {
		return result, fmt.Errorf(
			"%s: expected JSON defines no fields",
			expected.Benchmark,
		)
	}

	got, err := llm.ExtractJSON(answer)
	if err != nil {
		// The model did not produce parseable JSON: every field fails.
		result.Failed = len(want)
		result.Missing = sortedKeys(want)
		result.Violations = append(
			result.Violations,
			"answer is not valid JSON",
		)
		return result, nil
	}

	for _, key := range sortedKeys(want) {

		value, ok := got[key]

		if ok && jsonEqual(want[key], value, expected.Options) {
			result.Passed++
		} else {
			result.Failed++
			result.Missing = append(result.Missing, key)
		}
	}

	result.Score = (result.Passed * 100) / len(want)

	return result, nil
}

// jsonEqual deep-compares two values decoded from JSON. Strings are
// compared after trimming surrounding whitespace; opts can further
// relax number and string comparison.
func jsonEqual(want, got any, opts JSONOptions) bool {

	switch w := want.(type) {

	case string:
		g, ok := got.(string)
		if !ok {
			return false
		}
		ws, gs := strings.TrimSpace(w), strings.TrimSpace(g)
		if opts.CaseInsensitive {
			return strings.EqualFold(ws, gs)
		}
		return ws == gs

	case float64:
		g, ok := got.(float64)
		if !ok && opts.CoerceNumbers {
			if s, isString := got.(string); isString {
				parsed, err := strconv.ParseFloat(
					strings.TrimSpace(s), 64,
				)
				if err != nil {
					return false
				}
				g, ok = parsed, true
			}
		}
		if !ok {
			return false
		}
		return math.Abs(w-g) <= opts.NumberTolerance

	case bool:
		g, ok := got.(bool)
		return ok && w == g

	case nil:
		return got == nil

	case []any:
		g, ok := got.([]any)
		if !ok || len(w) != len(g) {
			return false
		}
		for i := range w {
			if !jsonEqual(w[i], g[i], opts) {
				return false
			}
		}
		return true

	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok || len(w) != len(g) {
			return false
		}
		for k, v := range w {
			gv, ok := g[k]
			if !ok || !jsonEqual(v, gv, opts) {
				return false
			}
		}
		return true

	default:
		return false
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
