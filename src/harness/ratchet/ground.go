package ratchet

// Evidence grounding — D-G2-1 enforcement (H-1/D1 remediation,
// upgraded from byte-grounding to WITNESS-grounding). Before
// Compare, every supplied fact is verified against the mechanism
// that established it: L6-plane facts by resolving their claimed
// witness event (existence → class → naming → bytes); benchmark-
// plane facts by the verdict/digest/location predicate. The
// registered selector params are mechanically applied. A fact that
// cannot be witnessed refuses; L11 still establishes no meaning
// about the witnessed bytes.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tofchaliss/themis/state"
)

// l6Sources: fact kinds whose refs are L6 ObjectIDs and whose
// establishment is a committed witnessing event (witnessClasses).
var l6Sources = map[string]bool{
	"l6_execution_record":   true,
	"l10_evaluation_record": true,
}

// GroundFacts verifies every supplied fact against its establishing
// mechanism and applies the matching selector's registered params.
// Returns a refusal (never a weaker package) on the first
// unwitnessed fact plus, on success, the derived run identity per
// grounded fact keyed by selector name (D4: run enumeration is
// DERIVED from witnesses, never caller text). Facts for undeclared
// selectors are left to checkEvidence's comparability refusal.
func GroundFacts(root *state.Root, benchRoot string, selectors []Selector, facts []EvidenceRef, side string) (map[string]string, *RefusalFact) {
	declared := map[string]Selector{}
	for _, s := range selectors {
		declared[s.Name] = s
	}
	runs := map[string]string{}
	for _, f := range facts {
		sel, ok := declared[f.Selector]
		if !ok {
			continue // checkEvidence refuses undeclared selectors
		}
		refuse := func(reason ReasonClass, detail string) (map[string]string, *RefusalFact) {
			return nil, &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: reason,
				Detail: fmt.Sprintf("%s selector %q: %s", side, f.Selector, detail)}
		}
		switch {
		case l6Sources[f.Source]:
			if !strings2ObjectID(f.Ref) {
				return refuse(ReasonProvenanceViolation, fmt.Sprintf("L6-plane ref %q is not an object identity", f.Ref))
			}
			if f.TaskID == "" || f.EventSeq < 1 {
				return refuse(ReasonProvenanceViolation, "L6-plane fact carries no witness (task_id + event_seq) — storage proves bytes, events prove establishment")
			}
			if root == nil {
				return refuse(ReasonEvidenceUnavailable, "no record plane available to resolve the witness")
			}
			stored, err := root.Store().GetObject(f.Ref)
			if err != nil {
				return refuse(ReasonEvidenceUnavailable, fmt.Sprintf("ref %q did not resolve in the record plane", f.Ref))
			}
			if !bytes.Equal(stored, f.Value) {
				return refuse(ReasonIntegrityFailure, fmt.Sprintf("supplied value bytes differ from the bytes at %q", f.Ref))
			}
			// L11 output is terminal: even a (mis)witnessed l11-*
			// artifact never serves as a fact (belt to the witness
			// braces — no event class witnesses these anyway).
			if isL11Artifact(f.Value) {
				return refuse(ReasonComparabilityViolation, "referenced bytes are an L11 artifact — L11 output is terminal and never a fact (D-L11-15)")
			}
			toolPin, params, perr := splitToolPin(sel)
			if perr != nil {
				return refuse(ReasonComparabilityViolation, perr.Error())
			}
			if detail := verifyL6Witness(root, f.Source, f.Ref, f.TaskID, f.EventSeq, toolPin); detail != "" {
				return refuse(ReasonProvenanceViolation, detail)
			}
			if r := applyParamsMap(params, sel.Name, f, side); r != nil {
				return nil, r
			}
			runs[f.Selector] = f.TaskID
		case benchSources[f.Source]:
			runID, detail := verifyBenchWitness(benchRoot, f.Source, f.Ref, f.Value)
			if detail != "" {
				return refuse(ReasonEvidenceUnavailable, detail)
			}
			if r := applyParams(sel, f, side); r != nil {
				return nil, r
			}
			runs[f.Selector] = runID
		default:
			return refuse(ReasonComparabilityViolation, fmt.Sprintf("source %q has no establishing mechanism", f.Source))
		}
	}
	return runs, nil
}

// isL11Artifact reports whether bytes self-describe as an L11
// instance artifact.
func isL11Artifact(b []byte) bool {
	var probe struct {
		Artifact string `json:"artifact"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return false
	}
	return strings.HasPrefix(probe.Artifact, "l11-")
}

// splitToolPin extracts the reserved "witness_tool" param (matched
// against the witnessing event's minting capability, D-G2-1) from
// the selector's params; the remainder subset-match the fact bytes.
func splitToolPin(sel Selector) (string, map[string]any, error) {
	var params map[string]any
	if err := json.Unmarshal(sel.Params, &params); err != nil {
		return "", nil, fmt.Errorf("registered params unreadable")
	}
	pin := ""
	if v, ok := params["witness_tool"]; ok {
		s, ok := v.(string)
		if !ok {
			return "", nil, fmt.Errorf("witness_tool param must be a string")
		}
		pin = s
		delete(params, "witness_tool")
	}
	return pin, params, nil
}

// applyParamsMap is applyParams over an already-split param map.
func applyParamsMap(params map[string]any, selName string, f EvidenceRef, side string) *RefusalFact {
	if len(params) == 0 {
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal(f.Value, &doc); err != nil {
		return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonComparabilityViolation,
			Detail: fmt.Sprintf("%s selector %q declares params but the fact is not a JSON object", side, selName)}
	}
	for k, want := range params {
		got, ok := doc[k]
		if !ok || got != want {
			return &RefusalFact{Artifact: "l11-refusal", Schema: 1, Reason: ReasonComparabilityViolation,
				Detail: fmt.Sprintf("%s selector %q: fact does not satisfy registered param %q", side, selName, k)}
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
	return applyParamsMap(params, sel.Name, f, side)
}

// maxInputBytes bounds invocation-surface JSON inputs.
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
