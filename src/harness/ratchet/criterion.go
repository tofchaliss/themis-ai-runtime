package ratchet

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Families is the closed candidate-family vocabulary (D-L11-3 v1 +
// the D-L11-19 revision families).
var Families = map[string]bool{
	"skill-revision":          true,
	"instruction-revision":    true,
	"contract-revision":       true,
	"regression-test":         true,
	"knowledge":               true,
	"criterion-revision":      true,
	"regression-set-revision": true,
}

// selectorSources is the closed v1 evidence-selector source
// vocabulary: established governed facts ONLY. L11 comparative
// packages are deliberately absent — L11 output is terminal and can
// never become comparator input (D-L11-7 Amendment 1, D-L11-15
// Class 4).
var selectorSources = map[string]bool{
	"l10_evaluation_record":     true,
	"l6_execution_record":       true,
	"benchmark_validated_score": true,
	"gate_verdict":              true,
}

// baselineConstraints is the closed v1 constraint vocabulary
// (D-L11-5 §4 / D-L11-6 §4). Extension is a schema revision.
var baselineConstraints = map[string]bool{
	"requires-current-active": true,
}

// orderingKinds is the closed D-L11-7 taxonomy.
var orderingKinds = map[string]bool{
	"none":          true,
	"per-metric":    true,
	"dominance":     true,
	"scalarization": true,
}

// deltaFieldTypes is the closed v1 Δ-shape type vocabulary.
var deltaFieldTypes = map[string]bool{"number": true, "integer": true}

// normativeTokens is the mechanical face of the D-L11-18 schema wall:
// no name inside a criterion may assert an evaluative or security
// proposition. Registration review remains the semantic gate; this
// closes the obvious lexical surface. Tokens match whole snake_case /
// kebab-case segments only (no substring false positives).
var normativeTokens = map[string]bool{
	"safe": true, "secure": true, "security": true, "safer": true,
	"acceptable": true, "recommended": true, "improvement": true,
	"improved": true, "better": true, "worse": true, "eligible": true,
	"promote": true, "promoted": true, "promotion": true, "risk": true,
	"good": true, "bad": true,
}

func checkNameSemantics(name, where string) error {
	for _, seg := range strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' }) {
		if normativeTokens[seg] {
			return fmt.Errorf("%s name %q asserts a proposition (%q) — names describe provenance, never meaning (D-L11-18)", where, name, seg)
		}
	}
	return nil
}

// Selector declares one evidence input by its fact identity: what the
// comparison requires, referenced from an established governed plane.
// The selector identifies evidence; L2/L6/L10 remain the owners of it
// (D-L11-6, owner confirmation). Params configure WHICH facts are
// selected (e.g. a contract token, a benchmark id) — carried by value
// so the criterion hash covers them.
type Selector struct {
	Name   string          `json:"name"`
	Source string          `json:"source"`
	Params json.RawMessage `json:"params"`
}

// ComparatorBinding pins the registered in-harness comparator by exact
// identity (D-L11-6 option A, LOCKED). Semantic change = new version,
// same version = same semantics forever (D-L11-17).
type ComparatorBinding struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

// DeltaField declares one field of the Δ output shape.
type DeltaField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// OrderingField declares the closed, fixed ordering semantics for one
// Δ field (D-L11-7: parameters yes, programs never). The delta is
// ORIENTED by direction (positive = toward improvement); the equal
// tolerance is the declared tie band (D-L11-7 §3 — ties are answers);
// the non-regression region is oriented-delta >= non_regression_min.
type OrderingField struct {
	Name             string   `json:"name"`
	Direction        string   `json:"direction"` // maximize | minimize
	EqualTolerance   *float64 `json:"equal_tolerance"`
	NonRegressionMin *float64 `json:"non_regression_min"`
	Weight           *float64 `json:"weight,omitempty"` // scalarization only
}

// Ordering declares which D-L11-7 structure the criterion commits to.
// Absent ordering = kind "none": Δ is descriptive and no better-claim
// is derivable, ever.
type Ordering struct {
	Kind   string          `json:"kind"`
	Fields []OrderingField `json:"fields"`
}

// Criterion is the governed, versioned, hash-pinned declarative
// Comparison Criterion (D-L11-6). It contains no executable logic and
// no continuation language: nothing here can express a consequence,
// an obligation, or a security meaning — structurally, because the
// schema is closed and those fields do not exist.
type Criterion struct {
	Version   int    `json:"version"`
	Name      string `json:"name"`
	Criterion int    `json:"criterion_version"`

	Families []string `json:"families"`

	CandidateSelectors []Selector `json:"candidate_selectors"`
	BaselineSelectors  []Selector `json:"baseline_selectors"`

	Comparator ComparatorBinding `json:"comparator"`
	Config     json.RawMessage   `json:"config"`

	DeltaShape []DeltaField `json:"delta_shape"`
	Ordering   *Ordering    `json:"ordering,omitempty"`

	BaselineConstraints []string `json:"baseline_constraints"`
	Provenance          []string `json:"provenance"`

	SHA256 string `json:"-"` // of the exact criterion bytes
	Raw    []byte `json:"-"` // verified bytes (no-reopen, R-L9-2)
}

// provenanceRequired is the closed, complete v1 conditioning-evidence
// set a criterion must demand of every comparison (D-L11-17 §1;
// completeness is not criterion-relaxable — the D-L10-10 pattern).
var provenanceRequired = []string{
	"evidence_records",
	"run_identities",
	"admission_observation",
}

const maxConfigBytes = 64 * 1024

// LoadCriterion reads and validates a Comparison Criterion
// fail-closed: closed schema, unknown fields refused, trailing
// content refused, nothing defaulted.
func LoadCriterion(path string) (*Criterion, error) {
	raw, err := readGoverned(path, maxCriterionBytes, ErrCriterion)
	if err != nil {
		return nil, err
	}
	return ParseCriterion(raw, path)
}

// ParseCriterion validates criterion bytes fail-closed. The SHA-256
// of the exact bytes is the identity component (D-L11-6 two-way
// identity with the registry binding).
func ParseCriterion(raw []byte, origin string) (*Criterion, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCriterion, origin, err)
	}
	dec := jsonDecoder(raw)
	var c Criterion
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCriterion, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrCriterion, origin)
	}
	if c.Version != 1 {
		return nil, fmt.Errorf("%w: %s: version must be 1", ErrCriterion, origin)
	}
	if !nameSyntax.MatchString(c.Name) || len(c.Name) > MaxNameLen {
		return nil, fmt.Errorf("%w: %s: bad criterion name %q", ErrCriterion, origin, c.Name)
	}
	if err := checkNameSemantics(c.Name, "criterion"); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCriterion, origin, err)
	}
	if c.Criterion < 1 {
		return nil, fmt.Errorf("%w: %s: criterion_version must be a positive integer", ErrCriterion, origin)
	}
	if len(c.Families) == 0 {
		return nil, fmt.Errorf("%w: %s: at least one applicable family required", ErrCriterion, origin)
	}
	seenFam := map[string]bool{}
	for _, f := range c.Families {
		if !Families[f] {
			return nil, fmt.Errorf("%w: %s: unknown candidate family %q", ErrCriterion, origin, f)
		}
		if seenFam[f] {
			return nil, fmt.Errorf("%w: %s: duplicate family %q", ErrCriterion, origin, f)
		}
		seenFam[f] = true
	}
	// Applicability is metadata, never selection (D-L11-6 Amendment 1)
	// — nothing here can cause execution; the schema simply has no
	// field able to express it.
	if err := checkSelectors(c.CandidateSelectors, "candidate", origin); err != nil {
		return nil, err
	}
	if err := checkSelectors(c.BaselineSelectors, "baseline", origin); err != nil {
		return nil, err
	}
	if !nameSyntax.MatchString(c.Comparator.Name) {
		return nil, fmt.Errorf("%w: %s: bad comparator name %q", ErrCriterion, origin, c.Comparator.Name)
	}
	if c.Comparator.Version < 1 {
		return nil, fmt.Errorf("%w: %s: comparator.version must be a positive integer", ErrCriterion, origin)
	}
	if len(c.Config) > maxConfigBytes {
		return nil, fmt.Errorf("%w: %s: config exceeds %d bytes", ErrCriterion, origin, maxConfigBytes)
	}
	trimmed := strings.TrimSpace(string(c.Config))
	if !strings.HasPrefix(trimmed, "{") || !json.Valid(c.Config) {
		return nil, fmt.Errorf("%w: %s: config must be a JSON object (use {} for none)", ErrCriterion, origin)
	}
	if len(c.DeltaShape) == 0 {
		return nil, fmt.Errorf("%w: %s: delta_shape required", ErrCriterion, origin)
	}
	deltaFields := map[string]bool{}
	for _, d := range c.DeltaShape {
		if !fieldSyntax.MatchString(d.Name) {
			return nil, fmt.Errorf("%w: %s: bad delta field name %q", ErrCriterion, origin, d.Name)
		}
		if err := checkNameSemantics(d.Name, "delta field"); err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrCriterion, origin, err)
		}
		if deltaFields[d.Name] {
			return nil, fmt.Errorf("%w: %s: duplicate delta field %q", ErrCriterion, origin, d.Name)
		}
		deltaFields[d.Name] = true
		if !deltaFieldTypes[d.Type] {
			return nil, fmt.Errorf("%w: %s: delta field %q: unknown type %q", ErrCriterion, origin, d.Name, d.Type)
		}
	}
	if err := checkOrdering(c.Ordering, deltaFields, origin); err != nil {
		return nil, err
	}
	seenBC := map[string]bool{}
	for _, bc := range c.BaselineConstraints {
		if !baselineConstraints[bc] {
			return nil, fmt.Errorf("%w: %s: unknown baseline constraint %q", ErrCriterion, origin, bc)
		}
		if seenBC[bc] {
			return nil, fmt.Errorf("%w: %s: duplicate baseline constraint %q", ErrCriterion, origin, bc)
		}
		seenBC[bc] = true
	}
	if err := checkProvenance(c.Provenance); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCriterion, origin, err)
	}
	c.SHA256 = hashBytes(raw)
	c.Raw = raw
	return &c, nil
}

func checkSelectors(sels []Selector, side, origin string) error {
	if len(sels) == 0 {
		return fmt.Errorf("%w: %s: at least one %s selector required", ErrCriterion, origin, side)
	}
	seen := map[string]bool{}
	for _, s := range sels {
		if !fieldSyntax.MatchString(s.Name) {
			return fmt.Errorf("%w: %s: bad %s selector name %q", ErrCriterion, origin, side, s.Name)
		}
		if err := checkNameSemantics(s.Name, "selector"); err != nil {
			return fmt.Errorf("%w: %s: %v", ErrCriterion, origin, err)
		}
		if seen[s.Name] {
			return fmt.Errorf("%w: %s: duplicate %s selector %q", ErrCriterion, origin, side, s.Name)
		}
		seen[s.Name] = true
		if !selectorSources[s.Source] {
			// The refusal message names the wall on purpose: L11
			// output is not a source and never will be by accident.
			return fmt.Errorf("%w: %s: selector %q: source %q is not an established governed fact source — L11 output is terminal and criteria consume only L2/L6/L10/benchmark facts (D-L11-15)", ErrCriterion, origin, s.Name, s.Source)
		}
		if len(s.Params) == 0 || !json.Valid(s.Params) || !strings.HasPrefix(strings.TrimSpace(string(s.Params)), "{") {
			return fmt.Errorf("%w: %s: selector %q: params must be a JSON object (use {} for none)", ErrCriterion, origin, s.Name)
		}
	}
	return nil
}

func checkOrdering(o *Ordering, deltaFields map[string]bool, origin string) error {
	if o == nil {
		return nil // kind "none": descriptive Δ, no better-claim derivable.
	}
	if !orderingKinds[o.Kind] {
		return fmt.Errorf("%w: %s: unknown ordering kind %q", ErrCriterion, origin, o.Kind)
	}
	if o.Kind == "none" {
		if len(o.Fields) != 0 {
			return fmt.Errorf("%w: %s: ordering kind none declares no fields", ErrCriterion, origin)
		}
		return nil
	}
	if len(o.Fields) == 0 {
		return fmt.Errorf("%w: %s: ordering kind %q requires fields", ErrCriterion, origin, o.Kind)
	}
	seen := map[string]bool{}
	for _, f := range o.Fields {
		if !deltaFields[f.Name] {
			return fmt.Errorf("%w: %s: ordering field %q is not a delta_shape field", ErrCriterion, origin, f.Name)
		}
		if seen[f.Name] {
			return fmt.Errorf("%w: %s: duplicate ordering field %q", ErrCriterion, origin, f.Name)
		}
		seen[f.Name] = true
		if f.Direction != "maximize" && f.Direction != "minimize" {
			return fmt.Errorf("%w: %s: ordering field %q: direction must be maximize or minimize", ErrCriterion, origin, f.Name)
		}
		// Nothing defaulted: every declared field states its tie band
		// and its non-regression region explicitly (D-L11-7 §3/§5).
		if f.EqualTolerance == nil || *f.EqualTolerance < 0 {
			return fmt.Errorf("%w: %s: ordering field %q: equal_tolerance must be an explicit non-negative number", ErrCriterion, origin, f.Name)
		}
		if f.NonRegressionMin == nil {
			return fmt.Errorf("%w: %s: ordering field %q: non_regression_min must be explicit", ErrCriterion, origin, f.Name)
		}
		if o.Kind == "scalarization" {
			if f.Weight == nil || *f.Weight <= 0 {
				return fmt.Errorf("%w: %s: ordering field %q: scalarization requires an explicit positive weight", ErrCriterion, origin, f.Name)
			}
		} else if f.Weight != nil {
			return fmt.Errorf("%w: %s: ordering field %q: weight is only meaningful under scalarization", ErrCriterion, origin, f.Name)
		}
	}
	return nil
}

func checkProvenance(got []string) error {
	if len(got) != len(provenanceRequired) {
		return fmt.Errorf("provenance must declare exactly %v — completeness is not criterion-relaxable (D-L11-17)", provenanceRequired)
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p] = true
	}
	for _, req := range provenanceRequired {
		if !seen[req] {
			return fmt.Errorf("provenance missing required element %q", req)
		}
	}
	return nil
}
