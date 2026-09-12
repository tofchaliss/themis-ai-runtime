package ratchet

// Stateless ordering derivations (D-L11-4 claim 2, D-L11-7): pure
// rewritings of a package's Δ under K's registered ordering. NEVER
// stored — a stored ordering could drift from K; single-home for
// ordering semantics is K itself. When K declares no ordering, no
// better-claim exists anywhere and these functions refuse.

import (
	"fmt"
	"math"
)

// Relation is the four-value derivation vocabulary of D-L11-7 §4 —
// always criterion-relative ("-under-K"), never free-standing. It is
// NOT an outcome vocabulary (D-L11-16: L11 has none); it is derived
// content.
type Relation string

const (
	RelBetter       Relation = "better-under-k"
	RelWorse        Relation = "worse-under-k"
	RelEqual        Relation = "equal-under-k"
	RelIncomparable Relation = "incomparable-under-k"
)

// oriented maps a raw Δ value to improvement orientation: positive =
// toward improvement under the declared direction.
func oriented(f OrderingField, v float64) float64 {
	if f.Direction == "minimize" {
		return -v
	}
	return v
}

// FieldDerivation is the per-metric derivation: relation and
// non-regression region membership for ONE declared field.
// Enumerated, never aggregated across metrics (D-L11-7 taxonomy ii).
type FieldDerivation struct {
	Field           string
	Relation        Relation
	WithinNonRegres bool
}

// DerivePerField computes the per-field derivations for any ordering
// kind that declares fields. Refuses when K declares no ordering.
func DerivePerField(c *Criterion, delta map[string]float64) ([]FieldDerivation, error) {
	if c.Ordering == nil || c.Ordering.Kind == "none" {
		return nil, fmt.Errorf("%w: criterion %s@%d declares no ordering — Δ is descriptive and no relation is derivable", ErrResolve, c.Name, c.Criterion)
	}
	out := make([]FieldDerivation, 0, len(c.Ordering.Fields))
	for _, f := range c.Ordering.Fields {
		v, ok := delta[f.Name]
		if !ok {
			return nil, fmt.Errorf("%w: delta missing declared field %q", ErrResolve, f.Name)
		}
		ov := oriented(f, v)
		rel := RelEqual
		if math.Abs(ov) > *f.EqualTolerance {
			if ov > 0 {
				rel = RelBetter
			} else {
				rel = RelWorse
			}
		}
		out = append(out, FieldDerivation{
			Field:           f.Name,
			Relation:        rel,
			WithinNonRegres: ov >= *f.NonRegressionMin,
		})
	}
	return out, nil
}

// DeriveRelation computes the single criterion-level relation where —
// and only where — K's registered ordering defines one:
//
//   - per-metric: REFUSED. Per-metric criteria resolve no tradeoff;
//     the multi-metric Δ stays visibly unresolved (D-L11-7 §2) — use
//     DerivePerField.
//   - dominance: Pareto over oriented deltas — better iff
//     better-or-equal on every field and strictly better on at least
//     one; symmetrically worse; equal iff equal on all; else
//     INCOMPARABLE (a result, not a failure — D-L11-7 §4).
//   - scalarization: s = Σ wᵢ·orientedᵢ, compared against the
//     linearly-combined declared bands (Σ wᵢ·tolᵢ). Fixed registered
//     semantics of the kind; the full Δ is always retained beside the
//     projection (D-L11-7 Amendment 2 — the caller stores Δ, never
//     the scalar).
func DeriveRelation(c *Criterion, delta map[string]float64) (Relation, error) {
	if c.Ordering == nil || c.Ordering.Kind == "none" {
		return "", fmt.Errorf("%w: criterion %s@%d declares no ordering — no better-claim exists (D-L11-4 claim 2)", ErrResolve, c.Name, c.Criterion)
	}
	switch c.Ordering.Kind {
	case "per-metric":
		return "", fmt.Errorf("%w: criterion %s@%d is per-metric — the tradeoff is unresolved inside L11 and remains so; derive per field", ErrResolve, c.Name, c.Criterion)
	case "dominance":
		perField, err := DerivePerField(c, delta)
		if err != nil {
			return "", err
		}
		anyBetter, anyWorse := false, false
		for _, fd := range perField {
			switch fd.Relation {
			case RelBetter:
				anyBetter = true
			case RelWorse:
				anyWorse = true
			}
		}
		switch {
		case anyBetter && anyWorse:
			return RelIncomparable, nil
		case anyBetter:
			return RelBetter, nil
		case anyWorse:
			return RelWorse, nil
		default:
			return RelEqual, nil
		}
	case "scalarization":
		var s, tol float64
		for _, f := range c.Ordering.Fields {
			v, ok := delta[f.Name]
			if !ok {
				return "", fmt.Errorf("%w: delta missing declared field %q", ErrResolve, f.Name)
			}
			s += *f.Weight * oriented(f, v)
			tol += *f.Weight * *f.EqualTolerance
		}
		switch {
		case math.Abs(s) <= tol:
			return RelEqual, nil
		case s > 0:
			return RelBetter, nil
		default:
			return RelWorse, nil
		}
	}
	return "", fmt.Errorf("%w: unknown ordering kind %q", ErrResolve, c.Ordering.Kind)
}

// DeriveNonRegression reports whether every declared field's oriented
// Δ lies within its declared non-regression region — the constituent
// question for resistant-under-S (D-L11-9 §4). A false result is the
// same class of fact as a true one (direction symmetry).
func DeriveNonRegression(c *Criterion, delta map[string]float64) (bool, error) {
	perField, err := DerivePerField(c, delta)
	if err != nil {
		return false, err
	}
	for _, fd := range perField {
		if !fd.WithinNonRegres {
			return false, nil
		}
	}
	return true, nil
}
