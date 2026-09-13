package ratchet

// Evidence grounding — the H-1 remediation. "Grounding is checked,
// not trusted" becomes true: before Compare, every supplied fact is
// resolved against its claimed plane — L6-plane refs are fetched
// from the store and byte-compared; file-plane refs are read through
// the caller's confined resolver and byte-compared; and the
// criterion's registered selector params are mechanically applied to
// the fact bytes (K's selectors choose evidence — the requester does
// not, D-L11-10 §2). A fact that cannot be grounded refuses; L11
// still establishes no meaning about the grounded bytes.

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/tofchaliss/themis/state"
)

// l6Sources: sources whose Ref is an L6 ObjectID.
var l6Sources = map[string]bool{
	"l6_execution_record":   true,
	"l10_evaluation_record": true,
}

// ExternalResolver reads external-plane evidence bytes (benchmark
// reports, gate verdicts) by ref. Implementations confine paths;
// this package never opens files for evidence itself.
type ExternalResolver func(ref string) ([]byte, error)

// GroundFacts verifies every supplied fact against its claimed
// plane and applies the matching selector's registered params.
// Returns a refusal (never a weaker package) on the first ungrounded
// fact; nil means all facts grounded. Facts for undeclared selectors
// are left to checkEvidence's comparability refusal.
func GroundFacts(store *state.ObjectStore, external ExternalResolver, selectors []Selector, facts []EvidenceRef, side string) *RefusalFact {
	declared := map[string]Selector{}
	for _, s := range selectors {
		declared[s.Name] = s
	}
	for _, f := range facts {
		sel, ok := declared[f.Selector]
		if !ok {
			continue // checkEvidence refuses undeclared selectors
		}
		var stored []byte
		var err error
		switch {
		case l6Sources[f.Source]:
			if !strings2ObjectID(f.Ref) {
				return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonProvenanceViolation,
					Detail: fmt.Sprintf("%s selector %q: L6-plane ref %q is not an object identity", side, f.Selector, f.Ref)}
			}
			if store == nil {
				return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonEvidenceUnavailable,
					Detail: fmt.Sprintf("%s selector %q: no record store available to ground the fact", side, f.Selector)}
			}
			stored, err = store.GetObject(f.Ref)
		default:
			if external == nil {
				return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonEvidenceUnavailable,
					Detail: fmt.Sprintf("%s selector %q: no resolver for source %q", side, f.Selector, f.Source)}
			}
			stored, err = external(f.Ref)
		}
		if err != nil {
			return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonEvidenceUnavailable,
				Detail: fmt.Sprintf("%s selector %q: ref %q did not resolve in its claimed plane", side, f.Selector, f.Ref)}
		}
		if !bytes.Equal(stored, f.Value) {
			return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonIntegrityFailure,
				Detail: fmt.Sprintf("%s selector %q: supplied value bytes differ from the bytes at %q", side, f.Selector, f.Ref)}
		}
		if r := applyParams(sel, f, side); r != nil {
			return r
		}
	}
	return nil
}

// applyParams mechanically applies the registered selector params to
// the fact bytes: every scalar param (k, v) must appear as a
// top-level field k == v of the fact object. Declared data with
// fixed closed semantics — parameters, never programs (D-L11-6).
func applyParams(sel Selector, f EvidenceRef, side string) *RefusalFact {
	var params map[string]any
	if err := json.Unmarshal(sel.Params, &params); err != nil {
		return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonComparabilityViolation,
			Detail: fmt.Sprintf("%s selector %q: registered params unreadable", side, sel.Name)}
	}
	if len(params) == 0 {
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal(f.Value, &doc); err != nil {
		return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonComparabilityViolation,
			Detail: fmt.Sprintf("%s selector %q declares params but the fact is not a JSON object", side, sel.Name)}
	}
	for k, want := range params {
		got, ok := doc[k]
		if !ok || got != want {
			return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonComparabilityViolation,
				Detail: fmt.Sprintf("%s selector %q: fact does not satisfy registered param %q", side, sel.Name, k)}
		}
	}
	return nil
}

// maxInputBytes bounds invocation-surface JSON inputs (admission
// files no longer exist; evidence files and authored artifacts do).
const maxInputBytes = 4 << 20 // 4 MiB

// ParseEvidenceRefs parses an evidence-refs JSON array with the same
// strictness as every governed artifact: bounded, duplicate-key
// refused, unknown fields refused, trailing content refused (the
// L10 M-1 lesson applied to the invocation surface).
func ParseEvidenceRefs(raw []byte, origin string) ([]EvidenceRef, error) {
	if len(raw) > maxInputBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", ErrResolve, origin, maxInputBytes)
	}
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrResolve, origin, err)
	}
	dec := jsonDecoder(raw)
	var refs []EvidenceRef
	if err := dec.Decode(&refs); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrResolve, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrResolve, origin)
	}
	return refs, nil
}
