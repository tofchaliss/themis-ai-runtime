package ratchet

// Registered comparators: the deterministic Δ computation as
// in-harness code identity (D-L11-6 option A, LOCKED — the
// canonicalizer precedent). The table is closed reviewed Go; a
// criterion binds name@version; a NEW comparator or any semantic
// change is a code change through the ordinary pipeline as a NEW
// version — same version, same semantics, forever (D-L11-17 §2iv).
// Comparators are pure and environment-free: no clock, no
// randomness, no host state — same inputs, bit-identical canonical Δ.

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Comparator computes the declared Δ fields from resolved evidence
// values. It performs arithmetic over established facts and NOTHING
// else: no interpretation, no judgment, no consequence (D-L11-18).
type Comparator interface {
	// Compute maps (pinned config, candidate facts, baseline facts)
	// to the Δ fields. Facts are keyed by selector name; values are
	// the resolved fact bytes. Errors are machinery/config errors —
	// they mint no outcome.
	Compute(config json.RawMessage, candidate, baseline map[string]json.RawMessage) (map[string]float64, error)
}

// comparators is the closed registered table. Additions go through
// the full Class-2/3 pipeline; nothing here is data-extensible.
var comparators = map[string]Comparator{
	"numeric-score-delta@1": numericScoreDelta{},
}

// LookupComparator resolves a criterion's comparator binding against
// the registered table, fail-closed.
func LookupComparator(b ComparatorBinding) (Comparator, error) {
	key := fmt.Sprintf("%s@%d", b.Name, b.Version)
	c, ok := comparators[key]
	if !ok {
		return nil, fmt.Errorf("%w: comparator %s is not registered in-harness code — comparators are never data", ErrResolve, key)
	}
	return c, nil
}

// numericScoreDelta@1: for each configured field mapping, Δ =
// numeric(candidate fact) − numeric(baseline fact). The config pins
// which selector feeds which Δ field:
//
//	{"fields":[{"delta":"score_delta","candidate":"candidate_score","baseline":"baseline_score"}]}
//
// Facts must be JSON numbers (or objects with a "score" number —
// the benchmark validated-score shape). Semantics of version 1 are
// frozen by this comment and the proofs; any change is @2.
type numericScoreDelta struct{}

type nsdConfig struct {
	Fields []struct {
		Delta     string `json:"delta"`
		Candidate string `json:"candidate"`
		Baseline  string `json:"baseline"`
	} `json:"fields"`
}

func (numericScoreDelta) Compute(config json.RawMessage, candidate, baseline map[string]json.RawMessage) (map[string]float64, error) {
	dec := jsonDecoder(config)
	var cfg nsdConfig
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("numeric-score-delta@1: bad config: %v", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("numeric-score-delta@1: trailing config content")
	}
	if len(cfg.Fields) == 0 {
		return nil, fmt.Errorf("numeric-score-delta@1: config.fields required")
	}
	out := map[string]float64{}
	for _, f := range cfg.Fields {
		if f.Delta == "" || f.Candidate == "" || f.Baseline == "" {
			return nil, fmt.Errorf("numeric-score-delta@1: field mapping requires delta, candidate, baseline names")
		}
		if _, dup := out[f.Delta]; dup {
			return nil, fmt.Errorf("numeric-score-delta@1: duplicate delta field %q", f.Delta)
		}
		cv, err := numericFact(candidate, f.Candidate)
		if err != nil {
			return nil, fmt.Errorf("numeric-score-delta@1: candidate: %v", err)
		}
		bv, err := numericFact(baseline, f.Baseline)
		if err != nil {
			return nil, fmt.Errorf("numeric-score-delta@1: baseline: %v", err)
		}
		d := cv - bv
		if math.IsNaN(d) || math.IsInf(d, 0) {
			return nil, fmt.Errorf("numeric-score-delta@1: non-finite delta for %q", f.Delta)
		}
		out[f.Delta] = d
	}
	return out, nil
}

func numericFact(facts map[string]json.RawMessage, name string) (float64, error) {
	raw, ok := facts[name]
	if !ok {
		return 0, fmt.Errorf("selector %q resolved no fact", name)
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, fmt.Errorf("selector %q: non-finite number", name)
		}
		return n, nil
	}
	var obj struct {
		Score *float64 `json:"score"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Score != nil {
		if math.IsNaN(*obj.Score) || math.IsInf(*obj.Score, 0) {
			return 0, fmt.Errorf("selector %q: non-finite score", name)
		}
		return *obj.Score, nil
	}
	return 0, fmt.Errorf("selector %q: fact is not a number or {score: number}", name)
}

// CanonicalDelta serializes Δ deterministically: sorted keys, shortest
// round-trip float encoding. This serialization is part of registered
// comparator semantics (D-L11-17 §3), not a storage detail.
func CanonicalDelta(delta map[string]float64) []byte {
	keys := make([]string, 0, len(delta))
	for k := range delta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteByte(':')
		b.WriteString(strconv.FormatFloat(delta[k], 'g', -1, 64))
	}
	b.WriteByte('}')
	return []byte(b.String())
}
