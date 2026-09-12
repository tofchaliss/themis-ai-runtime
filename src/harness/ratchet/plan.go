package ratchet

// The Evaluation Plan (D-L11-10 §4): inert, declarative,
// content-addressed data naming what needs to run. It is a REQUEST
// representation — never an authority-bearing prerequisite for
// execution (owner invariant): a task initiated pursuant to a plan
// is valid because ordinary initiation/execution mechanisms
// authorize it, never because the plan does. The schema is closed
// and contains no imperative or continuation vocabulary — "retry
// until PASS" is unrepresentable, not merely forbidden.

import (
	"fmt"
)

// RequiredRun names one governed run the plan asks for: a registered
// skill identity and a content commitment for its input. Declarative
// only — no ordering, no conditionals, no fallbacks.
type RequiredRun struct {
	Skill       string `json:"skill"`        // exact "name@version"
	InputSHA256 string `json:"input_sha256"` // input commitment
}

// requiredProvenance is the closed vocabulary of provenance the plan
// may demand of runs (recorded facts, never selections — D-L11-10:
// a plan may require that model identity be RECORDED; it never
// selects a model).
var requiredProvenance = map[string]bool{
	"model_identity": true,
	"plan_reference": true,
}

// EvaluationPlan is the closed schema.
type EvaluationPlan struct {
	Artifact string `json:"artifact"` // "l11-evaluation-plan"
	Schema   int    `json:"schema"`

	CandidateHash   string       `json:"candidate_hash"`
	ClaimedBaseline *BaselineRef `json:"claimed_baseline,omitempty"`

	Criteria []string `json:"criteria,omitempty"` // exact K pins
	Sets     []string `json:"sets,omitempty"`     // exact S pins

	RequiredRuns       []RequiredRun `json:"required_runs"`
	RequiredProvenance []string      `json:"required_provenance"`
}

// ParsePlan validates plan bytes fail-closed.
func ParsePlan(raw []byte, origin string) (*EvaluationPlan, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrResolve, origin, err)
	}
	dec := jsonDecoder(raw)
	var p EvaluationPlan
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrResolve, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrResolve, origin)
	}
	if p.Artifact != "l11-evaluation-plan" || p.Schema != 1 {
		return nil, fmt.Errorf("%w: %s: not an l11-evaluation-plan@1 artifact", ErrResolve, origin)
	}
	if !shaSyntax.MatchString(p.CandidateHash) {
		return nil, fmt.Errorf("%w: %s: candidate_hash must be a sha256 hex digest", ErrResolve, origin)
	}
	if len(p.Criteria) == 0 && len(p.Sets) == 0 {
		return nil, fmt.Errorf("%w: %s: a plan pins at least one criterion or set — exactly (D-L11-9 Am. 2)", ErrResolve, origin)
	}
	seen := map[string]bool{}
	for _, ref := range append(append([]string{}, p.Criteria...), p.Sets...) {
		if _, _, err := ParseRef(ref); err != nil {
			return nil, fmt.Errorf("%w: %s: pin %q: %v", ErrResolve, origin, ref, err)
		}
		if seen[ref] {
			return nil, fmt.Errorf("%w: %s: duplicate pin %q", ErrResolve, origin, ref)
		}
		seen[ref] = true
	}
	if len(p.RequiredRuns) == 0 {
		return nil, fmt.Errorf("%w: %s: required_runs must name at least one governed run", ErrResolve, origin)
	}
	for _, r := range p.RequiredRuns {
		if _, _, err := ParseRef(r.Skill); err != nil {
			return nil, fmt.Errorf("%w: %s: run skill %q: %v", ErrResolve, origin, r.Skill, err)
		}
		if !shaSyntax.MatchString(r.InputSHA256) {
			return nil, fmt.Errorf("%w: %s: run input commitment must be a sha256 hex digest", ErrResolve, origin)
		}
	}
	if len(p.RequiredProvenance) == 0 {
		return nil, fmt.Errorf("%w: %s: required_provenance must be explicit", ErrResolve, origin)
	}
	seenProv := map[string]bool{}
	for _, rp := range p.RequiredProvenance {
		if !requiredProvenance[rp] {
			return nil, fmt.Errorf("%w: %s: unknown required provenance %q", ErrResolve, origin, rp)
		}
		if seenProv[rp] {
			return nil, fmt.Errorf("%w: %s: duplicate required provenance %q", ErrResolve, origin, rp)
		}
		seenProv[rp] = true
	}
	return &p, nil
}

// CheckPlanConformance is mechanical record-to-plan matching — NOT a
// verification or evaluation (D-L11-10 owner amendment: the
// five-role division). It reports, as enumerated facts, whether a
// produced comparison package corresponds to the identities the plan
// enumerated. Empty result = conforms. It judges nothing and
// triggers nothing.
func CheckPlanConformance(plan *EvaluationPlan, planID string, pkg *ComparisonPackage) []string {
	var mismatches []string
	if pkg.PlanRef != planID {
		mismatches = append(mismatches, fmt.Sprintf("package plan_ref %q is not this plan (%s)", pkg.PlanRef, planID))
	}
	if pkg.CandidateHash != plan.CandidateHash {
		mismatches = append(mismatches, "package candidate is not the plan's candidate")
	}
	pinned := false
	for _, k := range plan.Criteria {
		if pkg.CriterionRef == k {
			pinned = true
			break
		}
	}
	if !pinned && len(plan.Criteria) > 0 {
		mismatches = append(mismatches, fmt.Sprintf("package criterion %s is not pinned by the plan", pkg.CriterionRef))
	}
	return mismatches
}
