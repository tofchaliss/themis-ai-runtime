package instructions

import (
	"fmt"
	"regexp"
	"strings"
)

// secretPatterns mirror the secret-guard hook's family (DAY-0 §3): a
// pasted credential in an instruction body is a load-time hard error.
// The scan is a capability boundary internal to Layer 1 — no
// instruction at any scope can address its configuration.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`(ghp|gho|ghu|ghs)_[A-Za-z0-9]{36,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	// \b keeps hyphenated prose ("risk-assessment-…") from matching:
	// no word boundary exists inside "risk", so only a standalone
	// sk- token can match.
	regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`(?i)(api[_-]?key|apikey|auth[_-]?token|secret|password)["']?\s*[:=]\s*["'][A-Za-z0-9+/_-]{16,}["']`),
}

// validate checks one instruction against its declaring source. A nil
// error with ok=false means the instruction is rejected and recorded
// (untrusted namespace violation); an error is a trusted-configuration
// abort. Identity judgments never inspect bodies: a byte-identical
// body confers no ownership.
func validate(inst Instruction, src Source) (conflict *Conflict, err error) {
	owner, err := NamespaceOwner(inst.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", inst.SourceRef, err)
	}
	if inst.Scope != src.Kind {
		return nil, fmt.Errorf("%s: %w: instruction %q declares scope %s in a %s source",
			inst.SourceRef, ErrScopeMismatch, inst.ID, inst.Scope, src.Kind)
	}
	if owner != src.Kind {
		v := fmt.Errorf("%s: %w: %q is owned by %s, declared by %s",
			inst.SourceRef, ErrNamespaceViolation, inst.ID, owner, src.Kind)
		if trustedKinds[src.Kind] {
			return nil, v
		}
		return &Conflict{
			Class:    ConflictShadowed,
			ID:       inst.ID,
			Scope:    inst.Scope,
			Source:   inst.SourceRef,
			BodyHash: inst.BodyHash,
			Detail:   v.Error(),
		}, nil
	}
	// Protection is shadow-resistance — a durability attribute an
	// untrusted source must not grant its own content (no
	// caller-supplied field determines the treatment of its own
	// request). Structural invalidity of the payload: abort, never
	// silently strip.
	if inst.Protected && !trustedKinds[src.Kind] {
		return nil, fmt.Errorf("%s: %w: instruction %q", inst.SourceRef, ErrProtectedDeclaration, inst.ID)
	}
	if !validCategories[inst.Category] {
		return nil, fmt.Errorf("%s: %w: unknown category %q", inst.SourceRef, ErrMalformed, inst.Category)
	}
	if strings.TrimSpace(inst.Body) == "" {
		return nil, fmt.Errorf("%s: %w: instruction %q", inst.SourceRef, ErrEmptyBody, inst.ID)
	}
	if len(inst.Body) > MaxInstructionBytes {
		return nil, fmt.Errorf("%s: %w: instruction %q is %d bytes (cap %d)",
			inst.SourceRef, ErrTooLarge, inst.ID, len(inst.Body), MaxInstructionBytes)
	}
	for _, p := range secretPatterns {
		if p.MatchString(inst.Body) {
			return nil, fmt.Errorf("%s: %w: instruction %q matches pattern %q",
				inst.SourceRef, ErrSecretDetected, inst.ID, p.String())
		}
	}
	return nil, nil
}
