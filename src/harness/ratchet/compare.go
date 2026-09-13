package ratchet

// The comparison core: an ACCEPTED invocation either produces a
// comparative-evidence package or a refusal fact for a named
// precondition — L11 has no outcome vocabulary (D-L11-16). Machinery
// failure returns an error and mints NOTHING (the L10 evaluator
// invariant). Once accepted, the outcome is durably recorded by the
// caller whatever it shows — there is no discard path (D-L11-14 §3).

import (
	"encoding/json"
	"fmt"
	"math"
)

// ReasonClass is the closed refusal vocabulary (D-L11-16 §3): each
// names a FAILED PRECONDITION, never quality. A refusal is
// epistemically neutral about the compared artifacts.
type ReasonClass string

const (
	ReasonUnregisteredArtifact   ReasonClass = "unregistered-artifact"
	ReasonWithdrawnArtifact      ReasonClass = "withdrawn-artifact"
	ReasonUnadmittedBaseline     ReasonClass = "unadmitted-baseline"
	ReasonEvidenceUnavailable    ReasonClass = "evidence-unavailable"
	ReasonComparabilityViolation ReasonClass = "comparability-violation"
	ReasonProvenanceViolation    ReasonClass = "provenance-violation"
	ReasonIntegrityFailure       ReasonClass = "integrity-failure"
)

var reasonClasses = map[ReasonClass]bool{
	ReasonUnregisteredArtifact:   true,
	ReasonWithdrawnArtifact:      true,
	ReasonUnadmittedBaseline:     true,
	ReasonEvidenceUnavailable:    true,
	ReasonComparabilityViolation: true,
	ReasonProvenanceViolation:    true,
	ReasonIntegrityFailure:       true,
}

// CheckReasonClass rejects reasons outside the closed vocabulary.
func CheckReasonClass(r ReasonClass) error {
	if !reasonClasses[r] {
		return fmt.Errorf("%w: unknown refusal reason class %q", ErrResolve, r)
	}
	return nil
}

// EvidenceRef is one resolved established fact: the selector it
// satisfies, the governed source plane, the immutable record
// identity, the SHA-256 of the exact value bytes, and the bytes. A
// proposition referencing evidence outside its enumerated record set
// is invalid (D-L11-4 §1) — grounding is checked, not trusted.
type EvidenceRef struct {
	Selector string          `json:"selector"`
	Source   string          `json:"source"`
	Ref      string          `json:"ref"` // ObjectID / record identity
	SHA256   string          `json:"sha256"`
	Value    json.RawMessage `json:"value"`
}

// AdmissionObservation records what L11 OBSERVED at the owning door
// at comparison time (D-L11-4 Amendment 2, D-L11-5 §2). It is an
// observation/reference, never an L11 admission fact: "L11 observed
// admission record X for B at door D against registry bytes H" —
// never "B is legitimately admitted". No bare boolean here may be
// read as authoritative admission.
type AdmissionObservation struct {
	Door             string `json:"door"`               // e.g. "l9-catalog", "l10-contract-registry"
	DoorRegistryHash string `json:"door_registry_hash"` // hash of the registry bytes observed
	Name             string `json:"name"`
	Version          int    `json:"version"`
	ArtifactSHA256   string `json:"artifact_sha256"`
	State            string `json:"state"`          // door-plane state string, quoted as observed
	CurrentActive    bool   `json:"current_active"` // as observed at the door
	ObservedAt       string `json:"observed_at"`    // caller-supplied instant (input, not derived)
}

// CompareInput is one accepted comparison invocation. Every identity
// is exact and externally supplied: L11 selects no criterion, no
// baseline, no version (D-L11-14 §1/§2). The caller resolves K from
// the registry and observes the door BEFORE invoking.
type CompareInput struct {
	CriterionRef string     // exact "name@version" as requested
	Criterion    *Criterion // resolved via Registry.ResolveCriterion
	// RegistrySHA256 binds the package to the criteria-registry
	// state in force at invocation (the seam authRegistrySHA256
	// precedent) — packages minted under a private registry are
	// distinguishable from governed ones.
	RegistrySHA256 string

	CandidateHash   string // content commitment of the candidate side
	ClaimedBaseline string // the candidate's CLAIM (D-L11-5 §6); may be ""

	Admission *AdmissionObservation // nil = admission not establishable

	CandidateFacts []EvidenceRef
	BaselineFacts  []EvidenceRef

	RunIdentities []string
	PlanRef       string // optional evaluation-plan reference (provenance only)
}

// RefusalFact records that an accepted invocation could not produce a
// package: a fact about an ATTEMPT, never a state of anything
// (D-L11-11). Terminal — it triggers nothing (D-L11-14).
type RefusalFact struct {
	Artifact     string      `json:"artifact"` // "l11-refusal"
	Schema       int         `json:"schema"`
	Reason       ReasonClass `json:"reason"`
	Detail       string      `json:"detail"`
	CriterionRef string      `json:"criterion_ref"`
	CriterionSHA string      `json:"criterion_sha256,omitempty"`
	Candidate    string      `json:"candidate,omitempty"`
	Claimed      string      `json:"claimed_baseline,omitempty"`
}

// ComparisonPackage is THE comparison record — one artifact, not two
// (D-L11-4 §3): the complete conditioning tuple plus Δ. Δ has no
// standalone meaning outside this package; no field here carries an
// evaluative or security proposition; relations and regions are
// DERIVED on demand, never stored (D-L11-4 claim 2).
type ComparisonPackage struct {
	Artifact string `json:"artifact"` // "l11-comparison"
	Schema   int    `json:"schema"`

	CriterionRef    string `json:"criterion_ref"`
	CriterionSHA256 string `json:"criterion_sha256"`
	RegistrySHA256  string `json:"criteria_registry_sha256"`
	ComparatorName  string `json:"comparator_name"`
	ComparatorVer   int    `json:"comparator_version"`
	ConfigSHA256    string `json:"config_sha256"`

	CandidateHash   string `json:"candidate_hash"`
	BaselineHash    string `json:"baseline_hash"` // FROM the observation, never the claim
	ClaimedBaseline string `json:"claimed_baseline,omitempty"`
	ClaimMismatch   bool   `json:"claim_mismatch"` // recorded visibly, never silently resolved

	Admission AdmissionObservation `json:"admission_observation"`

	CandidateEvidence []EvidenceRef `json:"candidate_evidence"`
	BaselineEvidence  []EvidenceRef `json:"baseline_evidence"`
	RunIdentities     []string      `json:"run_identities"`
	PlanRef           string        `json:"plan_ref,omitempty"`

	// Delta in K's declared shape, canonically serialized. The full
	// multi-metric Δ is always retained — scalarization is a
	// projection, never a replacement (D-L11-7 Amendment 2).
	Delta map[string]float64 `json:"delta"`
}

// Compare executes one accepted comparison invocation. Returns
// exactly one of (package, refusal) on success paths; an error means
// machinery failure and mints nothing.
func Compare(in CompareInput) (*ComparisonPackage, *RefusalFact, error) {
	if in.Criterion == nil {
		return nil, nil, fmt.Errorf("%w: no resolved criterion supplied", ErrResolve)
	}
	c := in.Criterion
	if !shaSyntax.MatchString(in.RegistrySHA256) {
		return nil, nil, fmt.Errorf("%w: no criteria-registry hash supplied — packages must bind to the registry in force", ErrResolve)
	}
	refuse := func(reason ReasonClass, detail string) (*ComparisonPackage, *RefusalFact, error) {
		return nil, &RefusalFact{
			Artifact:     "l11-refusal",
			Schema:       1,
			Reason:       reason,
			Detail:       detail,
			CriterionRef: in.CriterionRef,
			CriterionSHA: c.SHA256,
			Candidate:    in.CandidateHash,
			Claimed:      in.ClaimedBaseline,
		}, nil
	}
	if in.CandidateHash == "" || !shaSyntax.MatchString(in.CandidateHash) {
		return refuse(ReasonComparabilityViolation, "candidate content hash missing or malformed — no identity without bytes")
	}
	// Admission is a production precondition (D-L11-5 §3): cannot
	// establish baseline admission → the comparison DOES NOT EXIST.
	if in.Admission == nil {
		return refuse(ReasonUnadmittedBaseline, "no admission observation for the baseline at its owning door")
	}
	adm := *in.Admission
	if adm.Door == "" || !shaSyntax.MatchString(adm.DoorRegistryHash) || !shaSyntax.MatchString(adm.ArtifactSHA256) {
		return refuse(ReasonUnadmittedBaseline, "admission observation is not grounded in door-registry bytes")
	}
	if adm.Name == "" || adm.Version < 1 || adm.ObservedAt == "" {
		return refuse(ReasonUnadmittedBaseline, "admission observation lacks entry identity or observation instant")
	}
	if adm.State == "withdrawn" {
		return refuse(ReasonWithdrawnArtifact, "baseline is withdrawn at its owning door — new comparison refused; history stands")
	}
	if adm.State != "active" {
		return refuse(ReasonUnadmittedBaseline, fmt.Sprintf("observed baseline state %q is not an admitted state", adm.State))
	}
	// K's declared baseline constraints, mechanically applied — L11
	// evaluates the observed fact against the registered declaration;
	// it judges nothing (D-L11-5 §4 owner precision).
	for _, bc := range c.BaselineConstraints {
		if bc == "requires-current-active" && !adm.CurrentActive {
			return refuse(ReasonComparabilityViolation, "criterion declares requires-current-active and the observed baseline is not current-active")
		}
	}
	baselineHash := adm.ArtifactSHA256
	claimMismatch := in.ClaimedBaseline != "" && in.ClaimedBaseline != baselineHash

	// Evidence completeness + grounding, both sides. Missing →
	// evidence-unavailable; tampered value bytes →
	// integrity-failure; facts outside the enumerated selector set →
	// comparability-violation (no fact without grounding).
	candFacts, ref, err := checkEvidence(c.CandidateSelectors, in.CandidateFacts, "candidate")
	if ref != nil || err != nil {
		if ref != nil {
			return refuse(ref.Reason, ref.Detail)
		}
		return nil, nil, err
	}
	baseFacts, ref, err := checkEvidence(c.BaselineSelectors, in.BaselineFacts, "baseline")
	if ref != nil || err != nil {
		if ref != nil {
			return refuse(ref.Reason, ref.Detail)
		}
		return nil, nil, err
	}
	if len(in.RunIdentities) == 0 {
		return refuse(ReasonProvenanceViolation, "criterion requires run_identities and none were supplied")
	}

	comp, err := LookupComparator(c.Comparator)
	if err != nil {
		return nil, nil, err
	}
	delta, err := comp.Compute(c.Config, candFacts, baseFacts)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: comparator: %v", ErrResolve, err)
	}
	if err := checkDeltaShape(c, delta); err != nil {
		return nil, nil, err
	}

	return &ComparisonPackage{
		Artifact:          "l11-comparison",
		Schema:            1,
		CriterionRef:      in.CriterionRef,
		CriterionSHA256:   c.SHA256,
		RegistrySHA256:    in.RegistrySHA256,
		ComparatorName:    c.Comparator.Name,
		ComparatorVer:     c.Comparator.Version,
		ConfigSHA256:      hashBytes(c.Config),
		CandidateHash:     in.CandidateHash,
		BaselineHash:      baselineHash,
		ClaimedBaseline:   in.ClaimedBaseline,
		ClaimMismatch:     claimMismatch,
		Admission:         adm,
		CandidateEvidence: in.CandidateFacts,
		BaselineEvidence:  in.BaselineFacts,
		RunIdentities:     in.RunIdentities,
		PlanRef:           in.PlanRef,
		Delta:             delta,
	}, nil, nil
}

// checkEvidence validates facts against the enumerated selectors.
// Returns (facts-by-selector, refusal, error).
func checkEvidence(selectors []Selector, facts []EvidenceRef, side string) (map[string]json.RawMessage, *RefusalFact, error) {
	declared := map[string]Selector{}
	for _, s := range selectors {
		declared[s.Name] = s
	}
	bySel := map[string]json.RawMessage{}
	seen := map[string]bool{}
	for _, f := range facts {
		sel, ok := declared[f.Selector]
		if !ok {
			return nil, &RefusalFact{Reason: ReasonComparabilityViolation, Detail: fmt.Sprintf("%s fact for undeclared selector %q — evidence outside the enumerated set is invalid", side, f.Selector)}, nil
		}
		if seen[f.Selector] {
			return nil, &RefusalFact{Reason: ReasonComparabilityViolation, Detail: fmt.Sprintf("%s selector %q resolved more than one fact", side, f.Selector)}, nil
		}
		seen[f.Selector] = true
		if f.Source != sel.Source {
			return nil, &RefusalFact{Reason: ReasonComparabilityViolation, Detail: fmt.Sprintf("%s selector %q: fact source %q does not match declared source %q", side, f.Selector, f.Source, sel.Source)}, nil
		}
		if f.Ref == "" {
			return nil, &RefusalFact{Reason: ReasonProvenanceViolation, Detail: fmt.Sprintf("%s selector %q: fact has no record identity", side, f.Selector)}, nil
		}
		if !shaSyntax.MatchString(f.SHA256) || hashBytes(f.Value) != f.SHA256 {
			return nil, &RefusalFact{Reason: ReasonIntegrityFailure, Detail: fmt.Sprintf("%s selector %q: value bytes do not match their declared hash", side, f.Selector)}, nil
		}
		// Registered selector params applied here too (not only at
		// grounding), so no package can exist whose facts violate
		// them — reconstruction parity by construction.
		if r := applyParams(sel, f, side); r != nil {
			return nil, r, nil
		}
		bySel[f.Selector] = f.Value
	}
	for name := range declared {
		if !seen[name] {
			return nil, &RefusalFact{Reason: ReasonEvidenceUnavailable, Detail: fmt.Sprintf("%s selector %q resolved no fact", side, name)}, nil
		}
	}
	return bySel, nil, nil
}

// checkDeltaShape enforces the declared Δ shape exactly: every
// declared field present, no extras, integer fields integral. A
// mismatch is a comparator/config defect — machinery, not refusal.
func checkDeltaShape(c *Criterion, delta map[string]float64) error {
	declared := map[string]string{}
	for _, d := range c.DeltaShape {
		declared[d.Name] = d.Type
	}
	for name, v := range delta {
		typ, ok := declared[name]
		if !ok {
			return fmt.Errorf("%w: comparator produced undeclared delta field %q", ErrResolve, name)
		}
		if typ == "integer" && v != math.Trunc(v) {
			return fmt.Errorf("%w: delta field %q declared integer but is %v", ErrResolve, name, v)
		}
	}
	for name := range declared {
		if _, ok := delta[name]; !ok {
			return fmt.Errorf("%w: comparator omitted declared delta field %q", ErrResolve, name)
		}
	}
	return nil
}
