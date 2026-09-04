package service

import (
	"fmt"
	"regexp"
	"sort"
)

// injectionPatterns are deterministic markers of prompt-injection
// attempts inside supplied evidence. Detection flags the request in the
// response (injection_suspected) — evidence is still analyzed, mirroring
// benchmark B010 where the model must analyze around the injection, and
// the caller decides how to treat flagged results.
var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+instructions`),
	regexp.MustCompile(`(?i)disregard\s+(all\s+)?(previous|prior|above)`),
	regexp.MustCompile(`(?i)you\s+are\s+no\s+longer\s+a`),
	regexp.MustCompile(`(?i)new\s+instructions\s*:`),
	regexp.MustCompile(`(?i)reveal\s+(your\s+)?(system\s+prompt|instructions)`),
	regexp.MustCompile(`(?i)do\s+not\s+analyze`),
}

// SuspectInjection reports whether the evidence contains prompt-
// injection markers.
func SuspectInjection(evidence string) bool {
	for _, p := range injectionPatterns {
		if p.MatchString(evidence) {
			return true
		}
	}
	return false
}

// validStances are the only Enterprise Position stances the service
// will relay. Anything else from a model is a contract violation.
var validStances = map[string]bool{
	"affected":     true,
	"not_affected": true,
	"open":         true,
}

// CheckStance enforces the recommendation output contract.
func CheckStance(stance string) error {
	if !validStances[stance] {
		return fmt.Errorf(
			"model returned invalid stance %q (must be affected, not_affected, or open)",
			stance,
		)
	}
	return nil
}

// RequireFields enforces that the model's JSON answer contains every
// field of the operation's output contract.
func RequireFields(obj map[string]any, fields ...string) error {
	for _, f := range fields {
		if _, ok := obj[f]; !ok {
			return fmt.Errorf("model answer is missing required field %q", f)
		}
	}
	return nil
}

// validConfidence pins the only confidence levels prompts/recommend.md
// permits. Anything else from a model is a contract violation.
var validConfidence = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

// CheckConfidence enforces the recommendation confidence contract.
func CheckConfidence(confidence string) error {
	if !validConfidence[confidence] {
		return fmt.Errorf(
			"model returned invalid confidence %q (must be low, medium, or high)",
			confidence,
		)
	}
	return nil
}

// recommendContract pins the kinds of prompts/recommend.md's output
// contract. Unlike extraction, no recommendation field may be null:
// an uncertain model must recommend the "open" stance, not omit its
// reasoning.
var recommendContract = map[string]string{
	"finding_id":         "string",
	"recommended_stance": "string",
	"confidence":         "string",
	"rationale":          "string",
	"evidence_basis":     "array",
}

// ValidateRecommendation enforces presence, non-null, kind, and enum
// constraints on the model's recommendation. Violations are upstream
// failures and never relayed.
func ValidateRecommendation(rec map[string]any) error {

	fields := make([]string, 0, len(recommendContract))
	for f := range recommendContract {
		fields = append(fields, f)
	}
	sort.Strings(fields)

	for _, f := range fields {
		v, ok := rec[f]
		if !ok {
			return fmt.Errorf("model answer is missing required field %q", f)
		}
		if v == nil {
			return fmt.Errorf("model answer field %q must not be null", f)
		}
		if err := checkKind(f, recommendContract[f], v); err != nil {
			return err
		}
	}

	stance, _ := rec["recommended_stance"].(string)
	if err := CheckStance(stance); err != nil {
		return err
	}
	confidence, _ := rec["confidence"].(string)
	return CheckConfidence(confidence)
}

// extractContract pins the output contract of prompts/extract.md: every
// field must be present, and a non-null value must be of the pinned
// kind. The prompt requires null (plus a listing in unknown_fields) for
// anything the evidence does not establish, so null satisfies any kind.
var extractContract = map[string]string{
	"cve":                "string",
	"cwe":                "string",
	"description":        "string",
	"affected_component": "string",
	"affected_versions":  "array",
	"fixed_versions":     "array",
	"cvss":               "object",
	"exploitability":     "string",
	"references":         "array",
	"unknown_fields":     "array",
}

// ValidateExtractFacts enforces the extraction output contract on the
// model's parsed answer. Model output violating the contract is treated
// as an upstream failure and never relayed.
func ValidateExtractFacts(facts map[string]any) error {

	fields := make([]string, 0, len(extractContract))
	for f := range extractContract {
		fields = append(fields, f)
	}
	sort.Strings(fields)

	for _, f := range fields {
		v, ok := facts[f]
		if !ok {
			return fmt.Errorf("model answer is missing required field %q", f)
		}
		if err := checkKind(f, extractContract[f], v); err != nil {
			return err
		}
	}
	return nil
}

// checkKind verifies a non-null contract value against its pinned kind.
func checkKind(field, kind string, v any) error {
	if v == nil {
		return nil
	}
	var ok bool
	switch kind {
	case "string":
		_, ok = v.(string)
	case "array":
		_, ok = v.([]any)
	case "object":
		_, ok = v.(map[string]any)
	}
	if !ok {
		return fmt.Errorf(
			"model answer field %q must be a %s or null, got %T",
			field, kind, v,
		)
	}
	return nil
}
