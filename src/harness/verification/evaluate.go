package verification

// The L10 evaluator (D-L10-6 stage 4): bounded, deterministic,
// pure — it validates mechanical facts supplied by the governed
// execution path and applies the contract's declarative result
// mapping. It cannot invoke capabilities, re-run verification, retry
// execution, read or write any store, or initiate anything. Verifier
// output is untrusted verifier-domain data until validated and mapped
// (the hostile-verifier rule): a canonical result spelling "PASS" is
// just a domain string — only the mapping mints outcomes.
//
// Failure semantics are stage-indexed (D-L10-8). The evaluator
// produces exactly one outcome per evaluation instance, or — on a
// machinery defect in its own inputs' basic shape — an error and NO
// outcome: INVALID is a verdict BY a functioning evaluator, never
// about the evaluator itself.

import (
	"fmt"
)

// Reason is the closed diagnostic vocabulary for machinery-reserved
// outcomes (D-L10-8). Reasons are durable record content and never
// participate in workflow control — only the five outcomes cross to δ.
type Reason string

const (
	// UNAVAILABLE class: the required thing could not be obtained.
	ReasonEvidenceUnresolvable Reason = "evidence-unresolvable"
	ReasonNoCanonicalResult    Reason = "no-canonical-result"
	ReasonExecutionUnavailable Reason = "execution-unavailable"

	// INVALID class: a proposition about validity.
	ReasonEvidenceInadmissible   Reason = "evidence-inadmissible"
	ReasonResultOutOfMapping     Reason = "result-out-of-mapping"
	ReasonProvenanceIncomplete   Reason = "provenance-incomplete"
	ReasonConfigMismatch         Reason = "config-mismatch"
	ReasonNondeterminismDetected Reason = "nondeterminism-detected"
)

// unavailableReasons and invalidReasons close the per-class
// vocabulary: an outcome may carry only a reason of its own class.
var unavailableReasons = map[Reason]bool{
	ReasonEvidenceUnresolvable: true,
	ReasonNoCanonicalResult:    true,
	ReasonExecutionUnavailable: true,
}

var invalidReasons = map[Reason]bool{
	ReasonEvidenceInadmissible:   true,
	ReasonResultOutOfMapping:     true,
	ReasonProvenanceIncomplete:   true,
	ReasonConfigMismatch:         true,
	ReasonNondeterminismDetected: true,
}

// ProposedEvidence is one model-proposed evidence reference with the
// deterministic facts the governed path established about it: InTask
// (recorded within the evaluating task — D-L10-7 task binding) and
// Resolvable (the referenced object could actually be read). The
// evaluator trusts these FACTS from the deterministic caller, never
// from the model.
type ProposedEvidence struct {
	Slot       string
	ObjectID   string
	InTask     bool
	Resolvable bool
}

// ExecutionFacts carries what the governed execution path (L4→L5)
// established, by reference — the evaluation record restates nothing
// another layer owns (D-L10-10 #6).
type ExecutionFacts struct {
	// ExecutionRef references the L5/L7 execution record.
	ExecutionRef string
	// AppliedConfigSHA256 is the hash of the configuration L5 actually
	// applied; must equal the hash of the contract's pinned config.
	AppliedConfigSHA256 string
	// RawObjectID / CanonicalObjectID are the durable L6 identities of
	// the captured outputs; CanonicalResult is the canonical
	// verifier-domain value (empty when no canonical result exists).
	RawObjectID       string
	CanonicalObjectID string
	CanonicalResult   string
}

// Evaluation is the record content of one evaluation instance:
// references plus L10-computed facts (D-L10-10). Its durable identity
// is its committed L6 record — no independent evaluation ID exists
// (D-L10-9).
type Evaluation struct {
	ContractName    string `json:"contract_name"`
	ContractVersion int    `json:"contract_version"`
	ContractSHA256  string `json:"contract_sha256"`
	Capability      string `json:"capability"`
	RegistrySHA256  string `json:"registry_sha256"`

	Evidence []EvidenceRecord `json:"evidence"`

	ExecutionRef      string `json:"execution_ref"`
	RawObjectID       string `json:"raw_object_id,omitempty"`
	CanonicalObjectID string `json:"canonical_object_id,omitempty"`
	CanonicalResult   string `json:"canonical_result,omitempty"`

	Outcome Outcome `json:"outcome"`
	Reason  Reason  `json:"reason,omitempty"`

	// MatchedMapping is recorded convenience, derivable and
	// non-authoritative (D-L10-10 #7): the authoritative legitimacy
	// check is re-derivation from contract bytes + canonical result.
	MatchedMapping string `json:"matched_mapping,omitempty"`
}

// EvidenceRecord is one admitted (or refused) evidence reference as
// recorded: slot filled and L6 object identity.
type EvidenceRecord struct {
	Slot     string `json:"slot"`
	ObjectID string `json:"object_id"`
}

// Evaluate is the whole bounded evaluator: it validates the evaluation
// instance's mechanical facts in the fixed stage order of D-L10-8 and
// applies the contract mapping. The first failing stage determines the
// machinery-reserved outcome; a fully valid instance receives the
// mapped contract outcome. It returns an error — and NO Evaluation —
// only when its own inputs are malformed in a way that indicates a
// harness defect (nil contract, class-inconsistent facts): that is the
// evaluator-invariant-failure path, which mints nothing (D-L10-8).
func Evaluate(c *Contract, refs []ProposedEvidence, facts ExecutionFacts) (*Evaluation, error) {
	if c == nil || c.SHA256 == "" {
		return nil, fmt.Errorf("evaluator invariant: contract absent or unloaded — no outcome may be minted")
	}

	ev := &Evaluation{
		ContractName:    c.Name,
		ContractVersion: c.Contract,
		ContractSHA256:  c.SHA256,
		Capability:      c.Verifier.Capability,
		RegistrySHA256:  c.Verifier.RegistrySHA256,
		ExecutionRef:    facts.ExecutionRef,
	}

	// Stage: evidence admissibility (INVALID; execution disregarded).
	// Every declared slot must be filled exactly once by an in-task
	// reference with a well-formed object identity; no undeclared
	// slots (closed world).
	declared := map[string]bool{}
	for _, s := range c.Evidence {
		declared[s.Name] = true
	}
	filled := map[string]bool{}
	for _, r := range refs {
		if !declared[r.Slot] || filled[r.Slot] || !shaSyntax.MatchString(r.ObjectID) || !r.InTask {
			ev.Outcome, ev.Reason = OutcomeInvalid, ReasonEvidenceInadmissible
			return ev, nil
		}
		filled[r.Slot] = true
		ev.Evidence = append(ev.Evidence, EvidenceRecord{Slot: r.Slot, ObjectID: r.ObjectID})
	}
	for _, s := range c.Evidence {
		if !filled[s.Name] {
			ev.Outcome, ev.Reason = OutcomeInvalid, ReasonEvidenceInadmissible
			return ev, nil
		}
	}

	// Stage: evidence resolvability (UNAVAILABLE — structurally valid
	// reference whose object cannot be obtained; owner amendment to
	// D-L10-8).
	for _, r := range refs {
		if !r.Resolvable {
			ev.Outcome, ev.Reason = OutcomeUnavailable, ReasonEvidenceUnresolvable
			return ev, nil
		}
	}

	// Stage: execution produced a canonical result (UNAVAILABLE
	// otherwise — crash, timeout, provisioning failure).
	if facts.ExecutionRef == "" {
		ev.Outcome, ev.Reason = OutcomeUnavailable, ReasonExecutionUnavailable
		return ev, nil
	}
	if facts.CanonicalResult == "" || facts.CanonicalObjectID == "" {
		ev.Outcome, ev.Reason = OutcomeUnavailable, ReasonNoCanonicalResult
		return ev, nil
	}
	ev.RawObjectID = facts.RawObjectID
	ev.CanonicalObjectID = facts.CanonicalObjectID
	ev.CanonicalResult = facts.CanonicalResult

	// Stage: provenance completeness (INVALID — D-L10-10; a record
	// missing required provenance can never ground derived authority).
	if !shaSyntax.MatchString(facts.RawObjectID) || !shaSyntax.MatchString(facts.CanonicalObjectID) {
		ev.Outcome, ev.Reason = OutcomeInvalid, ReasonProvenanceIncomplete
		return ev, nil
	}

	// Stage: configuration identity (INVALID on mismatch — the applied
	// config must be the contract's pinned config).
	if facts.AppliedConfigSHA256 != hashBytes(c.Config) {
		ev.Outcome, ev.Reason = OutcomeInvalid, ReasonConfigMismatch
		return ev, nil
	}

	// Stage: contract mapping. The mapping's keys are the declared
	// domain the contract evaluates; an unmapped canonical result is
	// mechanically INVALID — no guessing, no nearest match, no
	// semantic interpretation (D-L10-8). Only this mapping mints
	// PASS/FAIL/INCONCLUSIVE: a canonical result that happens to spell
	// an outcome name is inert domain data.
	out, ok := c.ResultMapping[facts.CanonicalResult]
	if !ok {
		ev.Outcome, ev.Reason = OutcomeInvalid, ReasonResultOutOfMapping
		return ev, nil
	}
	if !contractMappable[out] {
		// Unreachable if the loader holds; treated as an evaluator
		// invariant rather than graded, since a contract in this state
		// was never loadable.
		return nil, fmt.Errorf("evaluator invariant: loaded contract maps to non-contract outcome %q", out)
	}
	ev.Outcome = out
	ev.MatchedMapping = facts.CanonicalResult
	return ev, nil
}

// CheckReasonClass verifies an evaluation's reason belongs to its
// outcome's class — contract outcomes carry no reason; machinery
// outcomes carry exactly one of their own class. Consumers (recording,
// reconstruction) use this as a structural integrity check.
func CheckReasonClass(ev *Evaluation) error {
	switch ev.Outcome {
	case OutcomePass, OutcomeFail, OutcomeInconclusive:
		if ev.Reason != "" {
			return fmt.Errorf("contract outcome %s must not carry a reason", ev.Outcome)
		}
	case OutcomeUnavailable:
		if !unavailableReasons[ev.Reason] {
			return fmt.Errorf("UNAVAILABLE requires a reason of its class, got %q", ev.Reason)
		}
	case OutcomeInvalid:
		if !invalidReasons[ev.Reason] {
			return fmt.Errorf("INVALID requires a reason of its class, got %q", ev.Reason)
		}
	default:
		return fmt.Errorf("unknown outcome %q", ev.Outcome)
	}
	return nil
}
