package instructions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// The directive-pattern policy is a versioned security-sensitive
// artifact (design §6, Q-L1-2): it lives under policies/security/,
// its content hash is recorded in the trace, changes are Class-3 with
// dual-corpus regression evidence, and pattern detection is defense
// in depth — never the sole enforcement mechanism for a security
// invariant. An unloadable or invalid policy aborts resolution (3.9):
// a configured control being absent is fail-open, and "the backstops
// exist" is never a reason to run without it.

var (
	ErrPolicyInvalid    = errors.New("invalid directive-pattern policy")
	ErrExemptionInvalid = errors.New("invalid pattern exemption")
)

// PatternTier separates hygiene heuristics from patterns anchored to
// Day-0 capability boundaries. The tier maps to the recorded conflict
// class (evidence quality); it does not gate exemptability — per the
// accepted design, Day-0 CONTROLS can never be exempted, but the
// patterns detecting attempts on them can be, because the underlying
// invariant is independently enforced (design §6, Q-L1-2).
type PatternTier string

const (
	TierHygiene  PatternTier = "hygiene"
	TierBoundary PatternTier = "boundary"
)

// Pattern is one directive-rejection rule. Patterns target
// instructions about instructions and controls, never bare verbs.
type Pattern struct {
	ID          string      `json:"id"`
	Tier        PatternTier `json:"tier"`
	Regex       string      `json:"regex"`
	Description string      `json:"description"`

	re *regexp.Regexp
}

func (p Pattern) conflictClass() ConflictClass {
	if p.Tier == TierBoundary {
		return ConflictRejectedProhibited
	}
	return ConflictRejectedPattern
}

// Policy is the loaded, validated pattern policy. The dual corpus is
// versioned with the list and re-verified at every load: a policy
// whose own corpus fails cannot activate.
type Policy struct {
	Version    int       `json:"version"`
	Patterns   []Pattern `json:"patterns"`
	MustReject []string  `json:"must_reject"`
	MustPass   []string  `json:"must_pass"`

	Hash string `json:"-"` // SHA-256 of the policy file bytes, hex
}

// Exemption is a narrowly scoped, recorded operator decision that
// suppresses ONE pattern for ONE instruction in ONE task. Exemption
// is never authorization: the admitted instruction still meets every
// other validation, and nothing downstream treats it as permitted.
// The granting channel (authenticated operator plane) is a deferred
// dependency; v1 consumes records handed in through task
// configuration and fails closed on any incomplete or stale record.
type Exemption struct {
	Operator        string `json:"operator"`
	TaskID          string `json:"task_id"`
	PatternID       string `json:"pattern_id"`
	PolicyHash      string `json:"policy_hash,omitempty"` // policy version granted against
	InstructionID   string `json:"instruction_id"`
	InstructionHash string `json:"instruction_hash,omitempty"` // pin to exact content
	Reason          string `json:"reason"`
	Timestamp       string `json:"timestamp"`
	Expiry          string `json:"expiry,omitempty"`
	Authorization   string `json:"authorization,omitempty"`
}

// Config carries the per-resolution security configuration. A nil
// Policy is a hard error: resolution without the configured detector
// does not happen. TaskID and Now are required whenever exemptions
// are present: Now is the orchestration-supplied clock reading
// (RFC3339) — time is an input, so resolution stays a pure function
// of its inputs.
type Config struct {
	Policy     *Policy
	Exemptions []Exemption
	TaskID     string
	Now        string
}

// LoadPolicy reads, validates, and self-checks the pattern policy.
// Fail closed: any deviation — unreadable file, unknown field, bad
// regex, duplicate id, bad tier, corpus violation — is an error.
func LoadPolicy(path string) (*Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrPolicyInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var p Policy
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrPolicyInvalid, path, err)
	}
	if p.Version < 1 {
		return nil, fmt.Errorf("%w: %s: version must be >= 1", ErrPolicyInvalid, path)
	}
	ids := map[string]bool{}
	for i := range p.Patterns {
		pat := &p.Patterns[i]
		if pat.ID == "" || pat.Regex == "" || pat.Description == "" {
			return nil, fmt.Errorf("%w: %s: pattern %d missing id, regex, or description", ErrPolicyInvalid, path, i)
		}
		if pat.Tier != TierHygiene && pat.Tier != TierBoundary {
			return nil, fmt.Errorf("%w: %s: pattern %q has unknown tier %q", ErrPolicyInvalid, path, pat.ID, pat.Tier)
		}
		if ids[pat.ID] {
			return nil, fmt.Errorf("%w: %s: duplicate pattern id %q", ErrPolicyInvalid, path, pat.ID)
		}
		ids[pat.ID] = true
		re, err := regexp.Compile(pat.Regex)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: pattern %q: %v", ErrPolicyInvalid, path, pat.ID, err)
		}
		pat.re = re
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content after policy object", ErrPolicyInvalid, path)
	}
	if len(p.Patterns) == 0 || len(p.MustReject) == 0 || len(p.MustPass) == 0 {
		return nil, fmt.Errorf("%w: %s: a hollow policy (no patterns or empty corpus side) cannot activate", ErrPolicyInvalid, path)
	}
	sum := sha256.Sum256(raw)
	p.Hash = hex.EncodeToString(sum[:])
	if err := p.checkCorpus(); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrPolicyInvalid, path, err)
	}
	return &p, nil
}

// checkCorpus re-verifies the dual corpus: every must-reject body
// matches at least one pattern; every must-pass body matches none.
// The corpus is a mechanical ratchet, not the authorization mechanism
// — security review separately judges pattern family membership.
func (p *Policy) checkCorpus() error {
	for _, body := range p.MustReject {
		if id, _ := p.match(body); id == "" {
			return fmt.Errorf("must-reject corpus entry matches no pattern: %q", body)
		}
	}
	for _, body := range p.MustPass {
		if id, _ := p.match(body); id != "" {
			return fmt.Errorf("must-pass corpus entry %q matches pattern %q", body, id)
		}
	}
	return nil
}

// match returns the first matching pattern in policy order (id and
// index), or "" if none match. Policy order makes rejection
// attribution deterministic.
func (p *Policy) match(body string) (string, int) {
	for i, pat := range p.Patterns {
		if pat.re.MatchString(body) {
			return pat.ID, i
		}
	}
	return "", -1
}

// validateExemptions fails closed on any exemption that is
// incomplete, unparseable, expired, bound to a different task,
// referencing an unknown pattern, or granted against a different
// policy version. A stale or invalid suppression record is a
// configuration discrepancy, never silently ignored and never
// silently applied. All pins are mandatory: an exemption names one
// pattern, one instruction content hash, one task, one policy
// version, and a bounded lifetime. (Authorization remains optional
// pending the authenticated operator plane — a deferred dependency.)
func (c Config) validateExemptions() error {
	if len(c.Exemptions) == 0 {
		return nil
	}
	if c.TaskID == "" || c.Now == "" {
		return fmt.Errorf("%w: exemptions require Config.TaskID and Config.Now", ErrExemptionInvalid)
	}
	now, err := time.Parse(time.RFC3339, c.Now)
	if err != nil {
		return fmt.Errorf("%w: Config.Now is not RFC3339: %v", ErrExemptionInvalid, err)
	}
	byID := map[string]bool{}
	for _, pat := range c.Policy.Patterns {
		byID[pat.ID] = true
	}
	for i, ex := range c.Exemptions {
		for name, v := range map[string]string{
			"operator": ex.Operator, "task_id": ex.TaskID, "pattern_id": ex.PatternID,
			"policy_hash": ex.PolicyHash, "instruction_id": ex.InstructionID,
			"instruction_hash": ex.InstructionHash, "reason": ex.Reason,
			"timestamp": ex.Timestamp, "expiry": ex.Expiry,
		} {
			if v == "" {
				return fmt.Errorf("%w: exemption %d missing %s", ErrExemptionInvalid, i, name)
			}
		}
		if !byID[ex.PatternID] {
			return fmt.Errorf("%w: exemption %d references unknown pattern %q", ErrExemptionInvalid, i, ex.PatternID)
		}
		if ex.PolicyHash != c.Policy.Hash {
			return fmt.Errorf("%w: exemption %d was granted against policy %s, current is %s", ErrExemptionInvalid, i, ex.PolicyHash, c.Policy.Hash)
		}
		if ex.TaskID != c.TaskID {
			return fmt.Errorf("%w: exemption %d is bound to task %q, this resolution is task %q", ErrExemptionInvalid, i, ex.TaskID, c.TaskID)
		}
		if _, err := time.Parse(time.RFC3339, ex.Timestamp); err != nil {
			return fmt.Errorf("%w: exemption %d timestamp is not RFC3339: %v", ErrExemptionInvalid, i, err)
		}
		expiry, err := time.Parse(time.RFC3339, ex.Expiry)
		if err != nil {
			return fmt.Errorf("%w: exemption %d expiry is not RFC3339: %v", ErrExemptionInvalid, i, err)
		}
		if !now.Before(expiry) {
			return fmt.Errorf("%w: exemption %d expired at %s (now %s)", ErrExemptionInvalid, i, ex.Expiry, c.Now)
		}
	}
	return nil
}

// checkPatterns runs the directive patterns over one untrusted
// instruction, boundary tier first so Day-0-anchored evidence is
// never downgraded by an earlier hygiene match. An exemption
// suppresses exactly the pattern decision it names and scanning
// CONTINUES: every match must be independently exempted or the
// instruction is rejected (rejection is the safe direction; the run
// continues). Applied exemptions are returned for the record even
// when a later pattern rejects.
func (c Config) checkPatterns(inst Instruction) (*Conflict, []Exemption) {
	var applied []Exemption
	for _, tier := range []PatternTier{TierBoundary, TierHygiene} {
		for _, pat := range c.Policy.Patterns {
			if pat.Tier != tier || !pat.re.MatchString(inst.Body) {
				continue
			}
			if ex := c.exemptionFor(pat, inst); ex != nil {
				applied = append(applied, *ex)
				continue
			}
			return &Conflict{
				Class:    pat.conflictClass(),
				ID:       inst.ID,
				Scope:    inst.Scope,
				Source:   inst.SourceRef,
				BodyHash: inst.BodyHash,
				Detail:   fmt.Sprintf("directive pattern %q (%s): %s", pat.ID, pat.Tier, pat.Description),
			}, applied
		}
	}
	return nil, applied
}

// exemptionFor finds a validated exemption for exactly this pattern
// and this instruction content. The content pin is mandatory: an
// exemption pinned to different bytes does not apply and the
// rejection stands recorded.
func (c Config) exemptionFor(pat Pattern, inst Instruction) *Exemption {
	for i, ex := range c.Exemptions {
		if ex.PatternID == pat.ID && ex.InstructionID == inst.ID && ex.InstructionHash == inst.BodyHash {
			return &c.Exemptions[i]
		}
	}
	return nil
}
