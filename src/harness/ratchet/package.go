package ratchet

// Instance-artifact identity and the set-level regression package
// (D-L11-9/11). Instance artifacts carry CONTENT-HASH identity only —
// naming is the registration plane's privilege; a "candidate registry
// by name" is structurally impossible here.

import (
	"encoding/json"
	"fmt"
)

// CanonicalBytes serializes an instance artifact deterministically:
// encoding/json fixes struct field order and sorts map keys, so the
// same value yields the same bytes — the identity precondition.
func CanonicalBytes(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("%w: canonical serialization: %v", ErrResolve, err)
	}
	return b, nil
}

// InstanceID returns the content-hash identity of canonical bytes.
func InstanceID(canonical []byte) string { return hashBytes(canonical) }

// SetConstituent references one member comparison by exact member pin
// and the constituent package's content identity.
type SetConstituent struct {
	Member    string `json:"member"`     // exact "name@version" pin from S
	PackageID string `json:"package_id"` // content hash of the constituent package
}

// RegressionPackage is the set-level regression-evidence package:
// aggregation by enumeration under a registered set identity, and
// NOTHING stronger (D-L11-9 §1). It exists only complete-under-S —
// partial coverage is representable solely as free-standing
// constituent packages that cannot claim the set identity (§3). It
// stores no "resistant" flag: resistant-under-S is derived on demand.
type RegressionPackage struct {
	Artifact string `json:"artifact"` // "l11-regression"
	Schema   int    `json:"schema"`

	SetRef    string `json:"set_ref"` // exact "name@version"
	SetSHA256 string `json:"set_sha256"`

	CandidateHash string `json:"candidate_hash"`
	BaselineHash  string `json:"baseline_hash"`

	Constituents []SetConstituent `json:"constituents"` // exactly one per member, member order
}

// BuildRegressionPackage assembles the set-level package from the
// constituent comparisons. Any missing member, identity drift, or
// cross-constituent inconsistency means the set-level package DOES
// NOT EXIST (D-L11-9 case A) — the error names why; nothing partial
// is produced, and there is no coverage field to misread.
func BuildRegressionPackage(setRef string, set *RegressionSet, constituents map[string]*ComparisonPackage) (*RegressionPackage, error) {
	if set == nil {
		return nil, fmt.Errorf("%w: no resolved regression set", ErrResolve)
	}
	var candidate, baseline string
	out := &RegressionPackage{
		Artifact:  "l11-regression",
		Schema:    1,
		SetRef:    setRef,
		SetSHA256: set.SHA256,
	}
	for _, member := range set.Members {
		pkg, ok := constituents[member]
		if !ok || pkg == nil {
			return nil, fmt.Errorf("%w: member %s has no constituent comparison — no set-level package exists under %s (completeness is mechanical, D-L11-9)", ErrResolve, member, setRef)
		}
		if pkg.CriterionRef != member {
			return nil, fmt.Errorf("%w: constituent for %s was produced under %s — member identity must match exactly", ErrResolve, member, pkg.CriterionRef)
		}
		if candidate == "" {
			candidate, baseline = pkg.CandidateHash, pkg.BaselineHash
		} else if pkg.CandidateHash != candidate || pkg.BaselineHash != baseline {
			return nil, fmt.Errorf("%w: constituent %s compares different artifacts — a set package is one candidate against one baseline", ErrResolve, member)
		}
		canonical, err := CanonicalBytes(pkg)
		if err != nil {
			return nil, err
		}
		out.Constituents = append(out.Constituents, SetConstituent{
			Member:    member,
			PackageID: InstanceID(canonical),
		})
	}
	out.CandidateHash, out.BaselineHash = candidate, baseline
	return out, nil
}

// DeriveResistantUnderSet is the bounded derivation of D-L11-9
// Amendment 1: every EXPLICITLY ENUMERATED constituent comparison in
// the registered finite set satisfies its declared non-regression
// region. It never establishes unrestricted absence of regression.
// Stateless: recomputed from constituent packages + their criteria;
// no stored flag exists anywhere. An incomparable or out-of-region
// constituent yields false — and false is the same class of result
// as true (completeness is coverage, not favorability).
func DeriveResistantUnderSet(set *RegressionSet, criteria map[string]*Criterion, constituents map[string]*ComparisonPackage) (bool, error) {
	if set == nil {
		return false, fmt.Errorf("%w: no resolved regression set", ErrResolve)
	}
	for _, member := range set.Members {
		pkg, ok := constituents[member]
		if !ok || pkg == nil {
			return false, fmt.Errorf("%w: member %s has no constituent — resistant-under-S is derivable only from complete coverage", ErrResolve, member)
		}
		c, ok := criteria[member]
		if !ok || c == nil {
			return false, fmt.Errorf("%w: member %s has no resolved criterion for region derivation", ErrResolve, member)
		}
		if c.SHA256 != pkg.CriterionSHA256 {
			return false, fmt.Errorf("%w: member %s: supplied criterion bytes do not match the constituent's conditioning tuple", ErrResolve, member)
		}
		within, err := DeriveNonRegression(c, pkg.Delta)
		if err != nil {
			return false, err
		}
		if !within {
			return false, nil
		}
	}
	return true, nil
}
