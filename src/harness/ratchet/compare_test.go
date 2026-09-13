package ratchet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func testCriterion(t *testing.T) *Criterion {
	t.Helper()
	m := validCriterionMap()
	m["config"] = map[string]any{
		"fields": []any{map[string]any{
			"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score",
		}},
	}
	c, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func fact(selector, source string, value any) EvidenceRef {
	b, _ := json.Marshal(value)
	return EvidenceRef{
		Selector: selector, Source: source,
		Ref:    "sha256:" + strings.Repeat("aa", 32),
		SHA256: hashBytes(b), Value: b,
	}
}

// scoreFact satisfies the fixture criterion's registered params
// ({"benchmark":"themis-bench-core"}) — params are mechanically
// applied at Compare and reconstruction.
func scoreFact(selector string, score float64) EvidenceRef {
	return fact(selector, "benchmark_validated_score",
		map[string]any{"score": score, "benchmark": "themis-bench-core"})
}

// doorFixture builds real door-registry bytes and the observation
// L11 would derive from them, so reconstruction can re-verify.
func doorFixture(t *testing.T) ([]byte, *AdmissionObservation) {
	t.Helper()
	doorBytes, _ := json.Marshal(map[string]any{
		"version": 1,
		"entries": []any{map[string]any{
			"name": "investigate-cve", "version": 1,
			"composition_sha256": strings.Repeat("22", 32),
			"state":              "active",
		}},
	})
	obs, err := deriveObservation("l9-catalog", doorBytes, "investigate-cve", 1)
	if err != nil || obs == nil {
		t.Fatalf("door fixture derivation failed: %v", err)
	}
	obs.DoorRegistryHash = hashBytes(doorBytes)
	obs.ObservedAt = "2026-09-12T00:00:00Z"
	return doorBytes, obs
}

func admittedObs(t *testing.T) *AdmissionObservation {
	_, obs := doorFixture(t)
	return obs
}

func validInput(t *testing.T, candScore, baseScore float64) CompareInput {
	return CompareInput{
		CriterionRef:   "bench-score-delta@1",
		Criterion:      testCriterion(t),
		RegistrySHA256: strings.Repeat("ab", 32),

		CandidateHash:   strings.Repeat("33", 32),
		ClaimedBaseline: strings.Repeat("22", 32),
		Admission:       admittedObs(t),
		CandidateFacts:  []EvidenceRef{scoreFact("candidate_score", candScore)},
		BaselineFacts:   []EvidenceRef{scoreFact("baseline_score", baseScore)},
		RunIdentities:   []string{"l4:17", "l4:18"},
	}
}

func TestComparePackage(t *testing.T) {
	pkg, ref, err := Compare(validInput(t, 0.91, 0.82))
	if err != nil || ref != nil {
		t.Fatalf("valid comparison failed: pkg=%v ref=%v err=%v", pkg, ref, err)
	}
	if pkg.Delta["score_delta"] != float64(0.91)-float64(0.82) {
		t.Fatalf("delta wrong: %v", pkg.Delta)
	}
	if pkg.BaselineHash != admittedObs(t).ArtifactSHA256 {
		t.Fatal("baseline hash must come from the observation")
	}
	if pkg.ClaimMismatch {
		t.Fatal("no mismatch expected")
	}
	if pkg.ConfigSHA256 == "" || pkg.CriterionSHA256 == "" {
		t.Fatal("conditioning tuple incomplete")
	}
}

func TestCompareDeterminism(t *testing.T) {
	var first []byte
	for i := 0; i < 50; i++ {
		pkg, _, err := Compare(validInput(t, 0.91, 0.82))
		if err != nil {
			t.Fatal(err)
		}
		cd := CanonicalDelta(pkg.Delta)
		if first == nil {
			first = cd
		} else if !bytes.Equal(first, cd) {
			t.Fatalf("canonical delta drifted on rerun %d: %s vs %s", i, first, cd)
		}
	}
}

// Direction symmetry: an unfavorable comparison is the same class of
// fact, packaged identically (D-L11-4 §6 — no success-only path).
func TestCompareDirectionSymmetry(t *testing.T) {
	worse, ref, err := Compare(validInput(t, 0.70, 0.82))
	if err != nil || ref != nil {
		t.Fatalf("unfavorable comparison did not produce a package: ref=%v err=%v", ref, err)
	}
	if worse.Delta["score_delta"] >= 0 {
		t.Fatal("expected negative delta")
	}
	if worse.Artifact != "l11-comparison" {
		t.Fatal("unfavorable result packaged differently")
	}
}

func TestCompareRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(in *CompareInput)
		reason ReasonClass
	}{
		{"no admission observation", func(in *CompareInput) { in.Admission = nil }, ReasonUnadmittedBaseline},
		{"withdrawn baseline", func(in *CompareInput) { in.Admission.State = "withdrawn" }, ReasonWithdrawnArtifact},
		{"unknown door state", func(in *CompareInput) { in.Admission.State = "proposed" }, ReasonUnadmittedBaseline},
		{"ungrounded observation", func(in *CompareInput) { in.Admission.DoorRegistryHash = "short" }, ReasonUnadmittedBaseline},
		{"stale under requires-current-active", func(in *CompareInput) { in.Admission.CurrentActive = false }, ReasonComparabilityViolation},
		{"missing evidence", func(in *CompareInput) { in.CandidateFacts = nil }, ReasonEvidenceUnavailable},
		{"tampered evidence", func(in *CompareInput) {
			in.CandidateFacts[0].Value = json.RawMessage(`0.99`)
		}, ReasonIntegrityFailure},
		{"undeclared selector fact", func(in *CompareInput) {
			in.CandidateFacts = append(in.CandidateFacts, fact("latency", "benchmark_validated_score", 1.0))
		}, ReasonComparabilityViolation},
		{"source mismatch", func(in *CompareInput) {
			in.CandidateFacts[0].Source = "l6_execution_record"
		}, ReasonComparabilityViolation},
		{"no record identity", func(in *CompareInput) { in.CandidateFacts[0].Ref = "" }, ReasonProvenanceViolation},
		{"no run identities", func(in *CompareInput) { in.RunIdentities = nil }, ReasonProvenanceViolation},
		{"malformed candidate hash", func(in *CompareInput) { in.CandidateHash = "xyz" }, ReasonComparabilityViolation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput(t, 0.9, 0.8)
			tc.mutate(&in)
			pkg, ref, err := Compare(in)
			if err != nil {
				t.Fatalf("machinery error instead of refusal: %v", err)
			}
			if pkg != nil {
				t.Fatal("package produced despite failed precondition")
			}
			if ref == nil || ref.Reason != tc.reason {
				t.Fatalf("wrong refusal: %+v (want %s)", ref, tc.reason)
			}
			// Refusal neutrality: the fact names the precondition and
			// identities only — no quality vocabulary (D-L11-16 §5).
			for _, banned := range []string{"bad", "poor", "quality", "regression", "worse", "failed candidate"} {
				if strings.Contains(strings.ToLower(ref.Detail), banned) {
					t.Fatalf("refusal detail carries quality language %q: %s", banned, ref.Detail)
				}
			}
			if err := CheckReasonClass(ref.Reason); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// The claim/observation mismatch is recorded, never silently
// resolved (D-L11-5 mismatch rule).
func TestCompareClaimMismatchRecorded(t *testing.T) {
	in := validInput(t, 0.9, 0.8)
	in.ClaimedBaseline = strings.Repeat("44", 32) // claims B; door observes B'
	pkg, ref, err := Compare(in)
	if err != nil || ref != nil {
		t.Fatalf("mismatch comparison failed: %v %v", ref, err)
	}
	if !pkg.ClaimMismatch {
		t.Fatal("claim mismatch not recorded")
	}
	if pkg.BaselineHash == in.ClaimedBaseline {
		t.Fatal("claim silently satisfied")
	}
}

func TestNoOrderingNoBetterClaim(t *testing.T) {
	m := validCriterionMap()
	delete(m, "ordering")
	m["config"] = map[string]any{"fields": []any{map[string]any{
		"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score",
	}}}
	c, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeriveRelation(c, map[string]float64{"score_delta": 5}); err == nil {
		t.Fatal("better-claim derived from a criterion with no ordering")
	}
	if _, err := DerivePerField(c, map[string]float64{"score_delta": 5}); err == nil {
		t.Fatal("per-field derivation from a criterion with no ordering")
	}
}

func TestPerMetricRefusesOverallRelation(t *testing.T) {
	c := testCriterion(t) // per-metric
	if _, err := DeriveRelation(c, map[string]float64{"score_delta": 0.1}); err == nil {
		t.Fatal("per-metric criterion yielded an overall relation — the tradeoff must stay unresolved")
	}
	fds, err := DerivePerField(c, map[string]float64{"score_delta": 0.1})
	if err != nil || len(fds) != 1 || fds[0].Relation != RelBetter {
		t.Fatalf("per-field derivation wrong: %v %v", fds, err)
	}
}

func multiMetricCriterion(t *testing.T, kind string, weights bool) *Criterion {
	t.Helper()
	m := validCriterionMap()
	m["delta_shape"] = []any{
		map[string]any{"name": "score_delta", "type": "number"},
		map[string]any{"name": "latency_delta", "type": "number"},
	}
	f1 := map[string]any{"name": "score_delta", "direction": "maximize", "equal_tolerance": 0.0, "non_regression_min": -0.05}
	f2 := map[string]any{"name": "latency_delta", "direction": "minimize", "equal_tolerance": 0.0, "non_regression_min": -10.0}
	if weights {
		f1["weight"] = 0.8
		f2["weight"] = 0.2
	}
	m["ordering"] = map[string]any{"kind": kind, "fields": []any{f1, f2}}
	c, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestDominance(t *testing.T) {
	c := multiMetricCriterion(t, "dominance", false)
	cases := []struct {
		score, latency float64
		want           Relation
	}{
		{+0.1, -5, RelBetter},       // better on both (latency minimized)
		{-0.1, +5, RelWorse},        // worse on both
		{+0.1, +5, RelIncomparable}, // better score, worse latency
		{0, 0, RelEqual},
	}
	for _, tc := range cases {
		rel, err := DeriveRelation(c, map[string]float64{"score_delta": tc.score, "latency_delta": tc.latency})
		if err != nil || rel != tc.want {
			t.Fatalf("dominance(%v,%v) = %v,%v want %v", tc.score, tc.latency, rel, err, tc.want)
		}
	}
}

func TestScalarization(t *testing.T) {
	c := multiMetricCriterion(t, "scalarization", true)
	// 0.8*(+0.1) + 0.2*(-(+30)) = 0.08 - 6 < 0 → worse.
	rel, err := DeriveRelation(c, map[string]float64{"score_delta": 0.1, "latency_delta": 30})
	if err != nil || rel != RelWorse {
		t.Fatalf("scalarization = %v,%v want worse", rel, err)
	}
	rel, err = DeriveRelation(c, map[string]float64{"score_delta": 0.1, "latency_delta": -1})
	if err != nil || rel != RelBetter {
		t.Fatalf("scalarization = %v,%v want better", rel, err)
	}
}

func TestNonRegressionDerivation(t *testing.T) {
	c := testCriterion(t)
	within, err := DeriveNonRegression(c, map[string]float64{"score_delta": -0.03})
	if err != nil || !within {
		t.Fatalf("within-region derivation wrong: %v %v", within, err)
	}
	within, err = DeriveNonRegression(c, map[string]float64{"score_delta": -0.5})
	if err != nil || within {
		t.Fatalf("out-of-region derivation wrong: %v %v", within, err)
	}
}

// A comparison package carries the FULL Δ even when K scalarizes —
// projection never replaces (D-L11-7 Amendment 2).
func TestScalarizationRetainsDelta(t *testing.T) {
	c := multiMetricCriterion(t, "scalarization", true)
	in := validInput(t, 0.9, 0.8)
	in.Criterion = c
	// Config maps only score; comparator must fill both declared
	// fields, so extend the config for the second field.
	m := validCriterionMap()
	_ = m
	cfg := map[string]any{"fields": []any{
		map[string]any{"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score"},
		map[string]any{"delta": "latency_delta", "candidate": "candidate_latency", "baseline": "baseline_latency"},
	}}
	cfgB, _ := json.Marshal(cfg)
	c.Config = cfgB
	// Criterion selectors don't include latency selectors; rebuild a
	// criterion whose selectors match.
	mm := validCriterionMap()
	mm["delta_shape"] = []any{
		map[string]any{"name": "score_delta", "type": "number"},
		map[string]any{"name": "latency_delta", "type": "number"},
	}
	mm["ordering"] = map[string]any{"kind": "scalarization", "fields": []any{
		map[string]any{"name": "score_delta", "direction": "maximize", "equal_tolerance": 0.0, "non_regression_min": -0.05, "weight": 0.8},
		map[string]any{"name": "latency_delta", "direction": "minimize", "equal_tolerance": 0.0, "non_regression_min": -10.0, "weight": 0.2},
	}}
	mm["candidate_selectors"] = []any{
		map[string]any{"name": "candidate_score", "source": "benchmark_validated_score", "params": map[string]any{}},
		map[string]any{"name": "candidate_latency", "source": "l6_execution_record", "params": map[string]any{}},
	}
	mm["baseline_selectors"] = []any{
		map[string]any{"name": "baseline_score", "source": "benchmark_validated_score", "params": map[string]any{}},
		map[string]any{"name": "baseline_latency", "source": "l6_execution_record", "params": map[string]any{}},
	}
	mm["config"] = cfg
	full, err := ParseCriterion(marshalCriterion(t, mm), "test")
	if err != nil {
		t.Fatal(err)
	}
	in.Criterion = full
	in.CandidateFacts = []EvidenceRef{
		fact("candidate_score", "benchmark_validated_score", 0.9),
		fact("candidate_latency", "l6_execution_record", 120.0),
	}
	in.BaselineFacts = []EvidenceRef{
		fact("baseline_score", "benchmark_validated_score", 0.8),
		fact("baseline_latency", "l6_execution_record", 100.0),
	}
	pkg, ref, err := Compare(in)
	if err != nil || ref != nil {
		t.Fatalf("scalarized comparison failed: %v %v", ref, err)
	}
	if len(pkg.Delta) != 2 {
		t.Fatalf("full delta not retained under scalarization: %v", pkg.Delta)
	}
	// The scalar relation is derivable but must not be stored.
	b, _ := json.Marshal(pkg)
	for _, banned := range []string{"relation", "better", "worse", "resistant", "scalar"} {
		if strings.Contains(strings.ToLower(string(b)), fmt.Sprintf("%q", banned)) {
			t.Fatalf("package stores derived ordering content %q", banned)
		}
	}
}

// The package schema itself carries no evaluative/security predicate
// fields (D-L11-18 schema wall) and no stored derivations.
func TestPackageSchemaHasNoEvaluativeFields(t *testing.T) {
	pkg, _, err := Compare(validInput(t, 0.9, 0.8))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(pkg)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for key := range m {
		if err := checkNameSemantics(key, "package field"); err != nil {
			t.Fatalf("package schema field violates the D-L11-18 wall: %v", err)
		}
	}
	for _, forbidden := range []string{"status", "outcome", "relation", "resistant", "recommended", "eligible"} {
		if _, ok := m[forbidden]; ok {
			t.Fatalf("package stores forbidden field %q", forbidden)
		}
	}
}
