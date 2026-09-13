package ratchet

// Cold reconstruction (D-L11-17): re-derive the same PROPOSITION —
// not merely the same numerical Δ — from the package bytes plus the
// registered criterion bytes. Three results, precisely separated:
// CONFIRMED; UNREPRODUCIBLE-FOR-MISSING-INPUTS (availability fact,
// epistemically neutral); DISCREPANCY (re-derivation differs, or
// bytes fail integrity). Reconstruction NEVER repairs, amends, or
// replaces the original package — a discrepancy is a subsequent fact
// about it (D-L11-16 owner refinement). This is licensed re-read (i)
// of D-L11-15 Class 4: identity-preserving, never epistemically
// additive — it can produce NO new comparative proposition.

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ReconstructionResult is a structural fact class about a
// reconstruction ATTEMPT — not an outcome vocabulary (D-L11-16).
type ReconstructionResult string

const (
	ReconConfirmed     ReconstructionResult = "confirmed"
	ReconMissingInputs ReconstructionResult = "unreproducible-for-missing-inputs"
	ReconDiscrepancy   ReconstructionResult = "discrepancy"
)

// Reconstruction reports one cold re-derivation. Discrepancies list
// WHAT disagreed; L11 never decides which computation is right —
// that is Governance's question (D-L11-4 §4).
type Reconstruction struct {
	Artifact string               `json:"artifact"` // "l11-reconstruction"
	Result   ReconstructionResult `json:"result"`

	PackageID     string   `json:"package_id"`
	MissingInputs []string `json:"missing_inputs,omitempty"`
	Discrepancies []string `json:"discrepancies,omitempty"`
}

// Reconstruct re-derives a comparison package cold: package bytes +
// the registered criterion bytes + the observed door-registry bytes
// are the ONLY inputs (plus this binary's registered comparator
// table). criterionBytes / doorRegistryBytes may be nil when they
// could not be obtained — missing-inputs facts, not discrepancies.
// A machinery error mints nothing. Reconstruction re-applies the
// SAME checks Compare applied (minus plane availability), so
// "confirmed" means "reproducible as Compare's proposition" — not
// merely internally consistent (close-review M-3).
func Reconstruct(packageBytes, criterionBytes, doorRegistryBytes []byte) (*Reconstruction, error) {
	rec := &Reconstruction{Artifact: "l11-reconstruction", PackageID: "sha256:" + hashBytes(packageBytes)}

	dec := jsonDecoder(packageBytes)
	var p ComparisonPackage
	if err := dec.Decode(&p); err != nil || dec.More() {
		rec.Result = ReconDiscrepancy
		rec.Discrepancies = append(rec.Discrepancies, "package bytes do not parse as an l11-comparison artifact")
		return rec, nil
	}
	if p.Artifact != "l11-comparison" || p.Schema != 1 {
		rec.Result = ReconDiscrepancy
		rec.Discrepancies = append(rec.Discrepancies, "package artifact/schema declaration is not l11-comparison@1")
		return rec, nil
	}

	if criterionBytes == nil {
		rec.Result = ReconMissingInputs
		rec.MissingInputs = append(rec.MissingInputs, fmt.Sprintf("criterion bytes for %s (sha256 %s)", p.CriterionRef, p.CriterionSHA256))
		return rec, nil
	}
	if hashBytes(criterionBytes) != p.CriterionSHA256 {
		// The supplied bytes are not the conditioning criterion — the
		// true input is (still) missing; nothing has disagreed.
		rec.Result = ReconMissingInputs
		rec.MissingInputs = append(rec.MissingInputs, fmt.Sprintf("supplied criterion bytes do not hash to the conditioning tuple's %s", p.CriterionSHA256))
		return rec, nil
	}
	c, err := ParseCriterion(criterionBytes, p.CriterionRef)
	if err != nil {
		rec.Result = ReconDiscrepancy
		rec.Discrepancies = append(rec.Discrepancies, fmt.Sprintf("conditioning criterion bytes no longer parse: %v", err))
		return rec, nil
	}

	// Re-verify the conditioning chain, clause by clause.
	if c.Comparator.Name != p.ComparatorName || c.Comparator.Version != p.ComparatorVer {
		rec.Discrepancies = append(rec.Discrepancies, "package comparator identity disagrees with the criterion binding")
	}
	if hashBytes(c.Config) != p.ConfigSHA256 {
		rec.Discrepancies = append(rec.Discrepancies, "package config hash disagrees with the criterion's pinned config")
	}
	if p.Admission.ArtifactSHA256 != p.BaselineHash {
		rec.Discrepancies = append(rec.Discrepancies, "package baseline hash disagrees with its own admission observation")
	}
	if !shaSyntax.MatchString(p.CandidateHash) {
		rec.Discrepancies = append(rec.Discrepancies, "package candidate hash is malformed")
	}
	if !shaSyntax.MatchString(p.RegistrySHA256) {
		rec.Discrepancies = append(rec.Discrepancies, "package lacks a well-formed criteria-registry binding")
	}
	if len(p.RunIdentities) == 0 {
		rec.Discrepancies = append(rec.Discrepancies, "package carries no run identities — Compare could not have produced it")
	}
	// D-L11-5 §2 re-verification: the admission observation against
	// the door registry bytes it claims to be grounded in.
	if doorRegistryBytes == nil {
		rec.MissingInputs = append(rec.MissingInputs, fmt.Sprintf("door-registry bytes for %s (sha256 %s)", p.Admission.Door, p.Admission.DoorRegistryHash))
	} else if hashBytes(doorRegistryBytes) != p.Admission.DoorRegistryHash {
		rec.MissingInputs = append(rec.MissingInputs, "supplied door-registry bytes do not hash to the observed registry state")
	} else {
		rec.Discrepancies = append(rec.Discrepancies, ReverifyAdmission(p.Admission, doorRegistryBytes)...)
	}
	candidate, missing, disc := reverifyFacts(c.CandidateSelectors, p.CandidateEvidence, "candidate")
	rec.MissingInputs = append(rec.MissingInputs, missing...)
	rec.Discrepancies = append(rec.Discrepancies, disc...)
	baseline, missing, disc := reverifyFacts(c.BaselineSelectors, p.BaselineEvidence, "baseline")
	rec.MissingInputs = append(rec.MissingInputs, missing...)
	rec.Discrepancies = append(rec.Discrepancies, disc...)

	if len(rec.Discrepancies) == 0 && len(rec.MissingInputs) == 0 {
		comp, err := LookupComparator(c.Comparator)
		if err != nil {
			rec.MissingInputs = append(rec.MissingInputs, fmt.Sprintf("registered comparator %s@%d is not present in this build", c.Comparator.Name, c.Comparator.Version))
		} else {
			delta, err := comp.Compute(c.Config, candidate, baseline)
			if err != nil {
				rec.Discrepancies = append(rec.Discrepancies, fmt.Sprintf("comparator re-execution failed: %v", err))
			} else if !bytes.Equal(CanonicalDelta(delta), CanonicalDelta(p.Delta)) {
				rec.Discrepancies = append(rec.Discrepancies, fmt.Sprintf("re-derived delta %s disagrees with recorded delta %s", CanonicalDelta(delta), CanonicalDelta(p.Delta)))
			}
		}
	}

	switch {
	case len(rec.Discrepancies) > 0:
		rec.Result = ReconDiscrepancy
	case len(rec.MissingInputs) > 0:
		rec.Result = ReconMissingInputs
	default:
		rec.Result = ReconConfirmed
	}
	return rec, nil
}

// reverifyFacts checks the package's embedded evidence against its
// declared hashes and the criterion's selector enumeration.
func reverifyFacts(selectors []Selector, facts []EvidenceRef, side string) (map[string]json.RawMessage, []string, []string) {
	var missing, disc []string
	declared := map[string]Selector{}
	for _, s := range selectors {
		declared[s.Name] = s
	}
	bySel := map[string]json.RawMessage{}
	seen := map[string]bool{}
	for _, f := range facts {
		sel, ok := declared[f.Selector]
		if !ok {
			disc = append(disc, fmt.Sprintf("%s evidence names undeclared selector %q", side, f.Selector))
			continue
		}
		if seen[f.Selector] {
			disc = append(disc, fmt.Sprintf("%s selector %q carries more than one fact — Compare refuses this shape", side, f.Selector))
			continue
		}
		seen[f.Selector] = true
		if f.Source != sel.Source {
			disc = append(disc, fmt.Sprintf("%s selector %q: fact source %q disagrees with the declared source %q", side, f.Selector, f.Source, sel.Source))
			continue
		}
		if hashBytes(f.Value) != f.SHA256 {
			disc = append(disc, fmt.Sprintf("%s selector %q: embedded value bytes fail their declared hash", side, f.Selector))
			continue
		}
		if r := applyParams(sel, f, side); r != nil {
			disc = append(disc, fmt.Sprintf("%s selector %q: fact no longer satisfies the registered params", side, f.Selector))
			continue
		}
		bySel[f.Selector] = f.Value
	}
	for name := range declared {
		if _, ok := bySel[name]; !ok {
			found := false
			for _, f := range facts {
				if f.Selector == name {
					found = true // present but discrepant — already recorded
					break
				}
			}
			if !found {
				missing = append(missing, fmt.Sprintf("%s selector %q has no embedded fact", side, name))
			}
		}
	}
	return bySel, missing, disc
}
