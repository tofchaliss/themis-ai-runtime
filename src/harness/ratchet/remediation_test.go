package ratchet

// M7 remediation evidence: the close reviews' surviving mutants and
// uncovered refusal branches, killed here.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/state"
)

// --- Loader refusal expansion (test review §1).

func TestCriterionLoaderRefusalExpansion(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"duplicate family", func(m map[string]any) { m["families"] = []any{"skill-revision", "skill-revision"} }},
		{"criterion_version zero", func(m map[string]any) { m["criterion_version"] = 0 }},
		{"bad name syntax", func(m map[string]any) { m["name"] = "Bench_Score" }},
		{"bad selector name", func(m map[string]any) {
			m["candidate_selectors"].([]any)[0].(map[string]any)["name"] = "Candidate-Score"
		}},
		{"normative selector name", func(m map[string]any) {
			m["candidate_selectors"].([]any)[0].(map[string]any)["name"] = "improved_score"
		}},
		{"duplicate selector", func(m map[string]any) {
			s := m["candidate_selectors"].([]any)[0]
			m["candidate_selectors"] = []any{s, s}
		}},
		{"bad comparator name", func(m map[string]any) {
			m["comparator"].(map[string]any)["name"] = "Numeric_Delta"
		}},
		{"bad delta field syntax", func(m map[string]any) {
			m["delta_shape"].([]any)[0].(map[string]any)["name"] = "Score-Delta"
			m["ordering"] = nil
		}},
		{"duplicate delta field", func(m map[string]any) {
			d := m["delta_shape"].([]any)[0]
			m["delta_shape"] = []any{d, d}
		}},
		{"ordering kind with no fields", func(m map[string]any) {
			m["ordering"].(map[string]any)["fields"] = []any{}
		}},
		{"duplicate ordering field", func(m map[string]any) {
			f := m["ordering"].(map[string]any)["fields"].([]any)[0]
			m["ordering"].(map[string]any)["fields"] = []any{f, f}
		}},
		{"bad direction", func(m map[string]any) {
			m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any)["direction"] = "optimize"
		}},
		{"negative equal_tolerance", func(m map[string]any) {
			m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any)["equal_tolerance"] = -0.1
		}},
		{"provenance duplicates at correct count", func(m map[string]any) {
			m["provenance"] = []any{"evidence_records", "evidence_records", "run_identities"}
		}},
		{"provenance junk at wrong count", func(m map[string]any) {
			m["provenance"] = []any{"evidence_records", "run_identities", "admission_observation", "vibes"}
		}},
		{"nested selector params", func(m map[string]any) {
			m["candidate_selectors"].([]any)[0].(map[string]any)["params"] = map[string]any{"filter": map[string]any{"gt": 3}}
		}},
		{"normative selector param key", func(m map[string]any) {
			m["candidate_selectors"].([]any)[0].(map[string]any)["params"] = map[string]any{"acceptable_score": 0.5}
		}},
		{"normative config key", func(m map[string]any) {
			m["config"] = map[string]any{"safe_threshold": 3}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validCriterionMap()
			tc.mutate(m)
			if _, err := ParseCriterion(marshalCriterion(t, m), "test"); err == nil {
				t.Fatal("doctored criterion accepted")
			}
		})
	}

	t.Run("trailing content", func(t *testing.T) {
		raw := marshalCriterion(t, validCriterionMap())
		if _, err := ParseCriterion(append(raw, []byte(" {}")...), "test"); err == nil {
			t.Fatal("trailing content accepted")
		}
	})
	t.Run("oversize config", func(t *testing.T) {
		m := validCriterionMap()
		m["config"] = map[string]any{"pad": strings.Repeat("x", maxConfigBytes)}
		if _, err := ParseCriterion(marshalCriterion(t, m), "test"); err == nil {
			t.Fatal("oversize config accepted")
		}
	})
}

func TestRegistryLoaderRefusalExpansion(t *testing.T) {
	dir := t.TempDir()
	write := func(m map[string]any) string {
		b, _ := json.Marshal(m)
		p := filepath.Join(dir, "reg.json")
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	entry := map[string]any{
		"name": "k", "version": 1,
		"artifact_sha256": strings.Repeat("ab", 32),
		"artifact_path":   "k.json", "state": "active",
	}
	cases := []struct {
		name string
		reg  map[string]any
	}{
		{"version zero", map[string]any{"version": 0, "kind": "criteria", "entries": []any{}}},
		{"unknown state", func() map[string]any {
			e := map[string]any{}
			for k, v := range entry {
				e[k] = v
			}
			e["state"] = "proposed" // a proposed entry must NEVER load as admitted
			return map[string]any{"version": 1, "kind": "criteria", "entries": []any{e}}
		}()},
		{"bad identity", func() map[string]any {
			e := map[string]any{}
			for k, v := range entry {
				e[k] = v
			}
			e["name"] = "Bad_Name"
			return map[string]any{"version": 1, "kind": "criteria", "entries": []any{e}}
		}()},
		{"malformed artifact hash", func() map[string]any {
			e := map[string]any{}
			for k, v := range entry {
				e[k] = v
			}
			e["artifact_sha256"] = "zz"
			return map[string]any{"version": 1, "kind": "criteria", "entries": []any{e}}
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := LoadRegistry(write(tc.reg)); err == nil {
				t.Fatal("doctored registry accepted")
			}
		})
	}
	t.Run("duplicate key", func(t *testing.T) {
		p := filepath.Join(dir, "dup.json")
		os.WriteFile(p, []byte(`{"version":1,"version":1,"kind":"criteria","entries":[]}`), 0o644)
		if _, err := LoadRegistry(p); err == nil {
			t.Fatal("duplicate key accepted")
		}
	})
	t.Run("non-regular artifact", func(t *testing.T) {
		sub := filepath.Join(dir, "adir")
		os.MkdirAll(sub, 0o755)
		e := map[string]any{}
		for k, v := range entry {
			e[k] = v
		}
		e["artifact_path"] = "adir"
		r, err := LoadRegistry(write(map[string]any{"version": 1, "kind": "criteria", "entries": []any{e}}))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := r.ResolveCriterion("k@1"); err == nil {
			t.Fatal("directory artifact accepted")
		}
	})
	t.Run("set two-way identity", func(t *testing.T) {
		sb, _ := json.Marshal(map[string]any{"version": 1, "name": "other-set", "set_version": 1, "members": []any{"k@1"}})
		os.WriteFile(filepath.Join(dir, "s.json"), sb, 0o644)
		p := write(map[string]any{"version": 1, "kind": "regression-sets", "entries": []any{map[string]any{
			"name": "core-set", "version": 1, "artifact_sha256": hashBytes(sb), "artifact_path": "s.json", "state": "active",
		}}})
		r, _ := LoadRegistry(p)
		if _, _, err := r.ResolveSet("core-set@1"); err == nil || !strings.Contains(err.Error(), "two-way identity") {
			t.Fatalf("set self-declaration disagreement not refused: %v", err)
		}
	})
}

func TestCandidatePlanRefusalExpansion(t *testing.T) {
	t.Run("candidate empty target", func(t *testing.T) {
		m := validCandidateMap()
		m["target"] = ""
		b, _ := json.Marshal(m)
		if _, err := ParseCandidate(b, "t"); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("candidate malformed claimed baseline", func(t *testing.T) {
		m := validCandidateMap()
		m["claimed_baseline"].(map[string]any)["version"] = 0
		b, _ := json.Marshal(m)
		if _, err := ParseCandidate(b, "t"); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("candidate wrong artifact", func(t *testing.T) {
		m := validCandidateMap()
		m["artifact"] = "l11-comparison"
		b, _ := json.Marshal(m)
		if _, err := ParseCandidate(b, "t"); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("plan malformed candidate hash", func(t *testing.T) {
		m := validPlanMap()
		m["candidate_hash"] = "nope"
		b, _ := json.Marshal(m)
		if _, err := ParsePlan(b, "t"); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("plan duplicate pin", func(t *testing.T) {
		m := validPlanMap()
		m["criteria"] = []any{"bench-score-delta@1", "bench-score-delta@1"}
		b, _ := json.Marshal(m)
		if _, err := ParsePlan(b, "t"); err == nil {
			t.Fatal("accepted")
		}
	})
	t.Run("plan duplicate provenance", func(t *testing.T) {
		m := validPlanMap()
		m["required_provenance"] = []any{"model_identity", "model_identity"}
		b, _ := json.Marshal(m)
		if _, err := ParsePlan(b, "t"); err == nil {
			t.Fatal("accepted")
		}
	})
}

// --- Compare-side gaps (test review §2 mutants 4-8).

func TestCompareBaselineSideRefusals(t *testing.T) {
	t.Run("baseline evidence missing", func(t *testing.T) {
		in := validInput(t, 0.9, 0.8)
		in.BaselineFacts = nil
		_, ref, err := Compare(in)
		if err != nil || ref == nil || ref.Reason != ReasonEvidenceUnavailable {
			t.Fatalf("baseline-side missing evidence: %+v %v", ref, err)
		}
	})
	t.Run("baseline evidence tampered", func(t *testing.T) {
		in := validInput(t, 0.9, 0.8)
		in.BaselineFacts[0].Value = json.RawMessage(`{"score":0.1,"benchmark":"themis-bench-core"}`)
		_, ref, err := Compare(in)
		if err != nil || ref == nil || ref.Reason != ReasonIntegrityFailure {
			t.Fatalf("baseline-side tamper: %+v %v", ref, err)
		}
	})
}

func TestCompareDuplicateFactPerSelector(t *testing.T) {
	in := validInput(t, 0.9, 0.8)
	in.CandidateFacts = append(in.CandidateFacts, scoreFact("candidate_score", 0.99))
	_, ref, err := Compare(in)
	if err != nil || ref == nil || ref.Reason != ReasonComparabilityViolation {
		t.Fatalf("duplicate fact accepted: %+v %v", ref, err)
	}
}

func TestCompareEmptyClaimIsNotMismatch(t *testing.T) {
	in := validInput(t, 0.9, 0.8)
	in.ClaimedBaseline = ""
	pkg, ref, err := Compare(in)
	if err != nil || ref != nil {
		t.Fatalf("empty claim refused: %v %v", ref, err)
	}
	if pkg.ClaimMismatch {
		t.Fatal("empty claim recorded as mismatch")
	}
}

func TestCompareAdmissionFieldRefusals(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(a *AdmissionObservation)
	}{
		{"empty door", func(a *AdmissionObservation) { a.Door = "" }},
		{"malformed artifact hash", func(a *AdmissionObservation) { a.ArtifactSHA256 = "zz" }},
		{"empty entry name", func(a *AdmissionObservation) { a.Name = "" }},
		{"version zero", func(a *AdmissionObservation) { a.Version = 0 }},
		{"no observation instant", func(a *AdmissionObservation) { a.ObservedAt = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput(t, 0.9, 0.8)
			tc.mutate(in.Admission)
			_, ref, err := Compare(in)
			if err != nil || ref == nil || ref.Reason != ReasonUnadmittedBaseline {
				t.Fatalf("%s: %+v %v", tc.name, ref, err)
			}
		})
	}
}

func TestCompareParamsApplied(t *testing.T) {
	in := validInput(t, 0.9, 0.8)
	// A fact from the wrong benchmark violates the registered params.
	in.CandidateFacts = []EvidenceRef{fact("candidate_score", "benchmark_validated_score",
		map[string]any{"score": 0.9, "benchmark": "other-bench"})}
	_, ref, err := Compare(in)
	if err != nil || ref == nil || ref.Reason != ReasonComparabilityViolation {
		t.Fatalf("params violation accepted: %+v %v", ref, err)
	}
}

// --- checkDeltaShape (test review mutant 8).

func TestDeltaShapeEnforced(t *testing.T) {
	t.Run("undeclared delta field", func(t *testing.T) {
		in := validInput(t, 0.9, 0.8)
		m := validCriterionMap()
		m["config"] = map[string]any{"fields": []any{
			map[string]any{"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score"},
			map[string]any{"delta": "surprise_delta", "candidate": "candidate_score", "baseline": "baseline_score"},
		}}
		c, err := ParseCriterion(marshalCriterion(t, m), "test")
		if err != nil {
			t.Fatal(err)
		}
		in.Criterion = c
		_, _, err = Compare(in)
		if err == nil || !strings.Contains(err.Error(), "undeclared delta field") {
			t.Fatalf("undeclared delta field not a machinery error: %v", err)
		}
	})
	t.Run("omitted declared field", func(t *testing.T) {
		in := validInput(t, 0.9, 0.8)
		m := validCriterionMap()
		m["delta_shape"] = []any{
			map[string]any{"name": "score_delta", "type": "number"},
			map[string]any{"name": "latency_delta", "type": "number"},
		}
		m["ordering"] = nil
		m["config"] = map[string]any{"fields": []any{
			map[string]any{"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score"},
		}}
		c, err := ParseCriterion(marshalCriterion(t, m), "test")
		if err != nil {
			t.Fatal(err)
		}
		in.Criterion = c
		_, _, err = Compare(in)
		if err == nil || !strings.Contains(err.Error(), "omitted declared delta field") {
			t.Fatalf("omitted field not a machinery error: %v", err)
		}
	})
	t.Run("integer integrality", func(t *testing.T) {
		in := validInput(t, 0.9, 0.8) // delta 0.1 is not integral
		m := validCriterionMap()
		m["delta_shape"].([]any)[0].(map[string]any)["type"] = "integer"
		m["ordering"] = nil
		m["config"] = map[string]any{"fields": []any{
			map[string]any{"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score"},
		}}
		c, err := ParseCriterion(marshalCriterion(t, m), "test")
		if err != nil {
			t.Fatal(err)
		}
		in.Criterion = c
		_, _, err = Compare(in)
		if err == nil || !strings.Contains(err.Error(), "declared integer") {
			t.Fatalf("non-integral integer not a machinery error: %v", err)
		}
	})
}

// --- Derivation bands and boundaries (test review mutants 1-3).

func TestTieBandNonZero(t *testing.T) {
	m := validCriterionMap()
	m["ordering"].(map[string]any)["fields"].([]any)[0].(map[string]any)["equal_tolerance"] = 0.05
	c, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err != nil {
		t.Fatal(err)
	}
	fds, err := DerivePerField(c, map[string]float64{"score_delta": 0.03})
	if err != nil || fds[0].Relation != RelEqual {
		t.Fatalf("within-band not equal: %v %v", fds, err)
	}
	fds, _ = DerivePerField(c, map[string]float64{"score_delta": 0.06})
	if fds[0].Relation != RelBetter {
		t.Fatalf("outside band not better: %v", fds)
	}
	fds, _ = DerivePerField(c, map[string]float64{"score_delta": -0.06})
	if fds[0].Relation != RelWorse {
		t.Fatalf("outside band not worse: %v", fds)
	}
}

func TestNonRegressionBoundaryInclusive(t *testing.T) {
	c := testCriterion(t) // non_regression_min = -0.05
	within, err := DeriveNonRegression(c, map[string]float64{"score_delta": -0.05})
	if err != nil || !within {
		t.Fatalf("boundary value excluded: %v %v", within, err)
	}
}

func TestScalarizationEqualBand(t *testing.T) {
	c := multiMetricCriterion(t, "scalarization", true)
	// Weighted sum exactly zero with zero tolerances → equal.
	rel, err := DeriveRelation(c, map[string]float64{"score_delta": 0, "latency_delta": 0})
	if err != nil || rel != RelEqual {
		t.Fatalf("zero scalar not equal: %v %v", rel, err)
	}
}

// --- Canonical serialization golden bytes (test review §4/mutant 9).

func TestCanonicalDeltaGoldenBytes(t *testing.T) {
	delta := map[string]float64{
		"zeta_delta":  -0.25,
		"alpha_delta": 1e21,
		"mid_delta":   3,
	}
	const golden = `{"alpha_delta":1e+21,"mid_delta":3,"zeta_delta":-0.25}`
	got := string(CanonicalDelta(delta))
	if got != golden {
		t.Fatalf("canonical serialization drifted:\n got: %s\nwant: %s", got, golden)
	}
}

// --- Reconstruction clause coverage (test review mutants 10-13).

func TestReconstructClauseCoverage(t *testing.T) {
	pkg, c, door := producedPackage(t)

	t.Run("comparator identity drift", func(t *testing.T) {
		doctored := *pkg
		doctored.ComparatorVer = 2
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, c.Raw, door)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("comparator drift not a discrepancy: %+v", rec)
		}
	})
	t.Run("config hash drift", func(t *testing.T) {
		doctored := *pkg
		doctored.ConfigSHA256 = strings.Repeat("77", 32)
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, c.Raw, door)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("config drift not a discrepancy: %+v", rec)
		}
	})
	t.Run("wrong artifact bytes", func(t *testing.T) {
		rec, _ := Reconstruct([]byte(`{"artifact":"l11-candidate","schema":1}`), c.Raw, door)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("wrong artifact not a discrepancy: %+v", rec)
		}
	})
	t.Run("duplicate selector fact", func(t *testing.T) {
		doctored := *pkg
		ev := append([]EvidenceRef{}, pkg.CandidateEvidence...)
		forged := scoreFact("candidate_score", 0.999)
		doctored.CandidateEvidence = append(ev, forged)
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, c.Raw, door)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("duplicate-selector package not a discrepancy: %+v", rec)
		}
	})
	t.Run("no run identities", func(t *testing.T) {
		doctored := *pkg
		doctored.RunIdentities = nil
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, c.Raw, door)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("runless package not a discrepancy: %+v", rec)
		}
	})
	t.Run("door registry hash mismatch is missing input", func(t *testing.T) {
		pb, _ := CanonicalBytes(pkg)
		rec, _ := Reconstruct(pb, c.Raw, []byte(`{"version":1,"entries":[]}`))
		if rec.Result != ReconMissingInputs {
			t.Fatalf("wrong door bytes should be missing input: %+v", rec)
		}
	})
	t.Run("doctored observation is a discrepancy", func(t *testing.T) {
		doctored := *pkg
		adm := pkg.Admission
		adm.CurrentActive = false
		adm.State = "active"
		doctored.Admission = adm
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, c.Raw, door)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("doctored observation not a discrepancy: %+v", rec)
		}
	})
}

// --- Set cross-constituent + criteria binding (mutants 16-17).

func TestSetCrossConstituentConsistency(t *testing.T) {
	pkg, c, _ := producedPackage(t)
	// A second registered criterion so a two-member set is buildable.
	m2 := validCriterionMap()
	m2["name"] = "second-check"
	m2["config"] = map[string]any{"fields": []any{map[string]any{
		"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score",
	}}}
	c2, err := ParseCriterion(marshalCriterion(t, m2), "test")
	if err != nil {
		t.Fatal(err)
	}
	in2 := validInput(t, 0.5, 0.4)
	in2.CriterionRef = "second-check@1"
	in2.Criterion = c2
	in2.CandidateHash = strings.Repeat("44", 32) // DIFFERENT candidate
	other, _, err := Compare(in2)
	if err != nil {
		t.Fatal(err)
	}
	set := &RegressionSet{Version: 1, Name: "core-regression", Set: 1,
		Members: []string{"bench-score-delta@1", "second-check@1"}, SHA256: strings.Repeat("66", 32)}
	_, err = BuildRegressionPackage("core-regression@1", set,
		map[string]*ComparisonPackage{"bench-score-delta@1": pkg, "second-check@1": other},
		map[string]*Criterion{"bench-score-delta@1": c, "second-check@1": c2})
	if err == nil || !strings.Contains(err.Error(), "different artifacts") {
		t.Fatalf("cross-constituent divergence accepted: %v", err)
	}
	// Same pair under both criteria builds.
	in3 := validInput(t, 0.5, 0.4)
	in3.CriterionRef = "second-check@1"
	in3.Criterion = c2
	same, _, err := Compare(in3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildRegressionPackage("core-regression@1", set,
		map[string]*ComparisonPackage{"bench-score-delta@1": pkg, "second-check@1": same},
		map[string]*Criterion{"bench-score-delta@1": c, "second-check@1": c2}); err != nil {
		t.Fatalf("consistent two-member set refused: %v", err)
	}
}

func TestSetCriteriaHashBinding(t *testing.T) {
	pkg, _, _ := producedPackage(t)
	set := &RegressionSet{Version: 1, Name: "core-regression", Set: 1,
		Members: []string{"bench-score-delta@1"}, SHA256: strings.Repeat("66", 32)}
	// A criterion whose bytes differ from the package's conditioning
	// tuple: registered under the same ref, different content.
	m := validCriterionMap()
	m["baseline_constraints"] = []any{}
	rogue, err := ParseCriterion(marshalCriterion(t, m), "test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildRegressionPackage("core-regression@1", set,
		map[string]*ComparisonPackage{"bench-score-delta@1": pkg},
		map[string]*Criterion{"bench-score-delta@1": rogue}); err == nil {
		t.Fatal("constituent under non-registered criterion bytes claimed set identity")
	}
	if _, err := BuildRegressionPackage("core-regression@1", set,
		map[string]*ComparisonPackage{"bench-score-delta@1": pkg}, nil); err == nil {
		t.Fatal("nil criteria binding accepted")
	}
}

// --- Plan conformance clauses (mutant 18).

func TestPlanConformanceClauses(t *testing.T) {
	pb, _ := json.Marshal(validPlanMap())
	plan, err := ParsePlan(pb, "test")
	if err != nil {
		t.Fatal(err)
	}
	planID := "sha256:" + strings.Repeat("99", 32)
	in := validInput(t, 0.9, 0.8)
	in.PlanRef = planID
	in.CandidateHash = plan.CandidateHash
	pkg, _, err := Compare(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("plan_ref mismatch", func(t *testing.T) {
		if got := CheckPlanConformance(plan, "sha256:"+strings.Repeat("88", 32), pkg); len(got) == 0 {
			t.Fatal("foreign plan_ref reported clean")
		}
	})
	t.Run("unpinned criterion", func(t *testing.T) {
		doctored := *pkg
		doctored.CriterionRef = "other-criterion@1"
		if got := CheckPlanConformance(plan, planID, &doctored); len(got) == 0 {
			t.Fatal("unpinned criterion reported clean")
		}
	})
}

// --- Grounding (H-1 remediation evidence).

func TestGroundFacts(t *testing.T) {
	root, err := state.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store := root.Store()
	recordBytes := []byte(`{"score":0.91,"benchmark":"themis-bench-core"}`)
	objID, err := store.StoreObject(state.ObjEvidencePayload, recordBytes)
	if err != nil {
		t.Fatal(err)
	}
	sel := []Selector{{Name: "candidate_score", Source: "l6_execution_record",
		Params: json.RawMessage(`{"benchmark":"themis-bench-core"}`)}}
	good := EvidenceRef{Selector: "candidate_score", Source: "l6_execution_record",
		Ref: objID, SHA256: hashBytes(recordBytes), Value: recordBytes}

	t.Run("grounded L6 fact passes", func(t *testing.T) {
		if r := GroundFacts(store, nil, sel, []EvidenceRef{good}, "candidate"); r != nil {
			t.Fatalf("grounded fact refused: %+v", r)
		}
	})
	t.Run("nonexistent ref refused", func(t *testing.T) {
		bad := good
		bad.Ref = "sha256:" + strings.Repeat("ee", 32)
		if r := GroundFacts(store, nil, sel, []EvidenceRef{bad}, "candidate"); r == nil || r.Reason != ReasonEvidenceUnavailable {
			t.Fatalf("nonexistent ref grounded: %+v", r)
		}
	})
	t.Run("value drift refused", func(t *testing.T) {
		bad := good
		bad.Value = []byte(`{"score":0.99,"benchmark":"themis-bench-core"}`)
		bad.SHA256 = hashBytes(bad.Value)
		if r := GroundFacts(store, nil, sel, []EvidenceRef{bad}, "candidate"); r == nil || r.Reason != ReasonIntegrityFailure {
			t.Fatalf("drifted value grounded: %+v", r)
		}
	})
	t.Run("malformed L6 ref refused", func(t *testing.T) {
		bad := good
		bad.Ref = "l4:999"
		if r := GroundFacts(store, nil, sel, []EvidenceRef{bad}, "candidate"); r == nil || r.Reason != ReasonProvenanceViolation {
			t.Fatalf("malformed ref grounded: %+v", r)
		}
	})
	t.Run("external plane via resolver", func(t *testing.T) {
		extSel := []Selector{{Name: "candidate_score", Source: "benchmark_validated_score",
			Params: json.RawMessage(`{}`)}}
		ext := func(ref string) ([]byte, error) { return recordBytes, nil }
		f := good
		f.Source = "benchmark_validated_score"
		f.Ref = "scores/run-17.json"
		if r := GroundFacts(store, ext, extSel, []EvidenceRef{f}, "candidate"); r != nil {
			t.Fatalf("external fact refused: %+v", r)
		}
		if r := GroundFacts(store, nil, extSel, []EvidenceRef{f}, "candidate"); r == nil || r.Reason != ReasonEvidenceUnavailable {
			t.Fatalf("resolverless external fact grounded: %+v", r)
		}
	})
}

// --- Door resolution (C-1 remediation evidence).

func TestObserveAdmission(t *testing.T) {
	dir := t.TempDir()
	writeDoor := func(entries ...map[string]any) string {
		var es []any
		for _, e := range entries {
			es = append(es, e)
		}
		b, _ := json.Marshal(map[string]any{"version": 1, "entries": es})
		p := filepath.Join(dir, "door.json")
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	active := map[string]any{"name": "investigate-cve", "version": 1,
		"composition_sha256": strings.Repeat("22", 32), "state": "active"}

	t.Run("resolves grounded observation", func(t *testing.T) {
		p := writeDoor(active)
		obs, err := ObserveAdmission("l9-catalog", p, "investigate-cve", 1, "now")
		if err != nil || obs == nil {
			t.Fatalf("%v %v", obs, err)
		}
		raw, _ := os.ReadFile(p)
		if obs.DoorRegistryHash != hashBytes(raw) {
			t.Fatal("observation not grounded in actual registry bytes")
		}
		if !obs.CurrentActive || obs.State != "active" {
			t.Fatalf("current-active derivation wrong: %+v", obs)
		}
	})
	t.Run("superseded version is not current", func(t *testing.T) {
		v2 := map[string]any{"name": "investigate-cve", "version": 2,
			"composition_sha256": strings.Repeat("33", 32), "state": "active"}
		p := writeDoor(active, v2)
		obs, err := ObserveAdmission("l9-catalog", p, "investigate-cve", 1, "now")
		if err != nil || obs == nil {
			t.Fatal(err)
		}
		if obs.CurrentActive {
			t.Fatal("superseded version observed as current")
		}
	})
	t.Run("absent entry yields nil, never invented", func(t *testing.T) {
		p := writeDoor(active)
		obs, err := ObserveAdmission("l9-catalog", p, "other-skill", 1, "now")
		if err != nil || obs != nil {
			t.Fatalf("absent entry: %+v %v", obs, err)
		}
	})
	t.Run("unknown door refused", func(t *testing.T) {
		p := writeDoor(active)
		if _, err := ObserveAdmission("shadow-door", p, "investigate-cve", 1, "now"); err == nil {
			t.Fatal("unknown door accepted")
		}
	})
	t.Run("duplicate keys refused", func(t *testing.T) {
		p := filepath.Join(dir, "dup.json")
		os.WriteFile(p, []byte(`{"entries":[{"name":"x","name":"y","version":1,"state":"active","composition_sha256":"`+strings.Repeat("22", 32)+`"}]}`), 0o644)
		if _, err := ObserveAdmission("l9-catalog", p, "x", 1, "now"); err == nil {
			t.Fatal("duplicate-key door registry accepted")
		}
	})
	t.Run("forged observation cannot reconstruct", func(t *testing.T) {
		// The C-1 attack, now closed at re-verification: a package
		// whose observation claims active/current against door bytes
		// that never said so is a discrepancy.
		pkg, c, _ := producedPackage(t)
		withdrawnDoor, _ := json.Marshal(map[string]any{"version": 1,
			"entries": []any{map[string]any{"name": "investigate-cve", "version": 1,
				"composition_sha256": strings.Repeat("22", 32), "state": "withdrawn"}}})
		doctored := *pkg
		adm := pkg.Admission
		adm.DoorRegistryHash = hashBytes(withdrawnDoor) // claims grounding in these bytes
		doctored.Admission = adm
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, c.Raw, withdrawnDoor)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("forged observation not exposed at re-verification: %+v", rec)
		}
	})
}

// --- Strict evidence-file parsing (M-2/M-4 remediation evidence).

func TestParseEvidenceRefsStrict(t *testing.T) {
	good := `[{"selector":"s","source":"benchmark_validated_score","ref":"r","sha256":"` + strings.Repeat("ab", 32) + `","value":1}]`
	if _, err := ParseEvidenceRefs([]byte(good), "t"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseEvidenceRefs([]byte(`[{"selector":"s","selector":"x","source":"b","ref":"r","sha256":"ab","value":1}]`), "t"); err == nil {
		t.Fatal("duplicate key accepted")
	}
	if _, err := ParseEvidenceRefs([]byte(`[{"selector":"s","source":"b","ref":"r","sha256":"ab","value":1,"grade":"A"}]`), "t"); err == nil {
		t.Fatal("unknown field accepted")
	}
	if _, err := ParseEvidenceRefs([]byte(good+" []"), "t"); err == nil {
		t.Fatal("trailing content accepted")
	}
}
