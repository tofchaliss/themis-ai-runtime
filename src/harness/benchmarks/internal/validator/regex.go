package validator

import (
	"fmt"
	"regexp"
)

// ValidateRegex validates a model response against Go regular
// expressions: every `required` pattern must match the answer, and no
// `forbidden` pattern may match. Patterns opt into case-insensitivity
// with the (?i) flag.
//
// Use this instead of the keyword validator when a correct answer can
// be phrased several ways ("rotate|revoke") or when free text must
// reference specific identifiers without exact-sentence matching.
func ValidateRegex(expected *Expected, answer string) (Result, error) {

	result := Result{
		Benchmark:  expected.Benchmark,
		Missing:    []string{},
		Violations: []string{},
	}

	for _, pattern := range expected.Required {

		re, err := regexp.Compile(pattern)
		if err != nil {
			return result, fmt.Errorf(
				"%s: invalid required pattern %q: %w",
				expected.Benchmark,
				pattern,
				err,
			)
		}

		if re.MatchString(answer) {
			result.Passed++
		} else {
			result.Failed++
			result.Missing = append(result.Missing, pattern)
		}
	}

	for _, pattern := range expected.Forbidden {

		re, err := regexp.Compile(pattern)
		if err != nil {
			return result, fmt.Errorf(
				"%s: invalid forbidden pattern %q: %w",
				expected.Benchmark,
				pattern,
				err,
			)
		}

		if re.MatchString(answer) {
			result.Violations = append(result.Violations, pattern)
		}
	}

	result.Score = score(result.Passed, len(expected.Required), len(result.Violations))

	return result, nil
}
