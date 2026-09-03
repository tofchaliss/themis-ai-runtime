package service

import (
	"fmt"
	"regexp"
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
