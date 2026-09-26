package ratchet

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

func producedPackage(t *testing.T) (*ComparisonPackage, *Criterion, ReconstructInputs) {
	t.Helper()
	in := validInput(t, 0.91, 0.82)
	pkg, ref, err := Compare(in)
	if err != nil || ref != nil {
		t.Fatalf("compare failed: %v %v", ref, err)
	}
	doorBytes, _ := doorFixture(t)
	bench := benchFixture(t, append(append([]EvidenceRef{}, in.CandidateFacts...), in.BaselineFacts...)...)
	return pkg, in.Criterion, ReconstructInputs{CriterionBytes: in.Criterion.Raw, DoorRegistryBytes: doorBytes, BenchRoot: bench}
}

func TestReconstructConfirmed(t *testing.T) {
	pkg, _, inputs := producedPackage(t)
	pb, err := CanonicalBytes(pkg)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Reconstruct(pb, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Result != ReconConfirmed {
		t.Fatalf("expected confirmed: %+v", rec)
	}
}

// Cold: reconstruction consumes ONLY package bytes + criterion bytes
// (this test never touches the original Compare inputs) — the
// delete-every-cache architectural proof (D-L11-17).
func TestReconstructIsCold(t *testing.T) {
	pkg, c, inputs := producedPackage(t)
	pb, _ := CanonicalBytes(pkg)
	cb := append([]byte(nil), c.Raw...)
	// Round-trip through serialization to sever any in-memory link.
	rec, err := Reconstruct(append([]byte(nil), pb...), ReconstructInputs{CriterionBytes: cb, DoorRegistryBytes: inputs.DoorRegistryBytes, BenchRoot: inputs.BenchRoot})
	if err != nil || rec.Result != ReconConfirmed {
		t.Fatalf("cold reconstruction failed: %+v %v", rec, err)
	}
}

func TestReconstructMissingInputs(t *testing.T) {
	pkg, _, inputs := producedPackage(t)
	pb, _ := CanonicalBytes(pkg)

	rec, err := Reconstruct(pb, ReconstructInputs{DoorRegistryBytes: inputs.DoorRegistryBytes, BenchRoot: inputs.BenchRoot})
	if err != nil || rec.Result != ReconMissingInputs {
		t.Fatalf("nil criterion: want missing-inputs, got %+v %v", rec, err)
	}
	if len(rec.Discrepancies) != 0 {
		t.Fatal("missing inputs recorded as disagreement — availability is neutral")
	}

	// Wrong criterion bytes = the true input is still missing.
	rec, err = Reconstruct(pb, ReconstructInputs{CriterionBytes: []byte(`{"other":"criterion"}`), DoorRegistryBytes: inputs.DoorRegistryBytes, BenchRoot: inputs.BenchRoot})
	if err != nil || rec.Result != ReconMissingInputs {
		t.Fatalf("wrong criterion bytes: want missing-inputs, got %+v %v", rec, err)
	}
}

func TestReconstructDiscrepancy(t *testing.T) {
	pkg, _, inputs := producedPackage(t)

	t.Run("tampered delta", func(t *testing.T) {
		doctored := *pkg
		doctored.Delta = map[string]float64{"score_delta": 0.5}
		pb, _ := CanonicalBytes(&doctored)
		rec, err := Reconstruct(pb, inputs)
		if err != nil || rec.Result != ReconDiscrepancy {
			t.Fatalf("tampered delta not a discrepancy: %+v %v", rec, err)
		}
	})
	t.Run("tampered evidence bytes", func(t *testing.T) {
		doctored := *pkg
		ev := make([]EvidenceRef, len(pkg.CandidateEvidence))
		copy(ev, pkg.CandidateEvidence)
		ev[0].Value = json.RawMessage(`0.999`)
		doctored.CandidateEvidence = ev
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, inputs)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("tampered evidence not a discrepancy: %+v", rec)
		}
	})
	t.Run("baseline/observation disagreement", func(t *testing.T) {
		doctored := *pkg
		doctored.BaselineHash = strings.Repeat("55", 32)
		pb, _ := CanonicalBytes(&doctored)
		rec, _ := Reconstruct(pb, inputs)
		if rec.Result != ReconDiscrepancy {
			t.Fatalf("baseline drift not a discrepancy: %+v", rec)
		}
	})
}

// Reconstruction never repairs: the input bytes are untouched and no
// new package is produced under any result.
func TestReconstructNeverRepairs(t *testing.T) {
	pkg, _, inputs := producedPackage(t)
	doctored := *pkg
	doctored.Delta = map[string]float64{"score_delta": 0.5}
	pb, _ := CanonicalBytes(&doctored)
	before := append([]byte(nil), pb...)
	rec, err := Reconstruct(pb, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, pb) {
		t.Fatal("reconstruction mutated the package bytes")
	}
	if rec.Result != ReconDiscrepancy {
		t.Fatalf("expected discrepancy: %+v", rec)
	}
	b, _ := json.Marshal(rec)
	if strings.Contains(string(b), `"artifact":"l11-comparison"`) {
		t.Fatal("reconstruction produced a package")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	root, err := state.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, _ := producedPackage(t)
	id, canonical, err := StoreInstance(root.Store(), pkg)
	if err != nil {
		t.Fatal(err)
	}
	if id != "sha256:"+InstanceID(canonical) {
		t.Fatalf("instance identity is not the address: %s", id)
	}
	back, err := LoadComparison(root.Store(), id)
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := CanonicalBytes(back)
	if !bytes.Equal(cb, canonical) {
		t.Fatal("round-trip changed canonical bytes")
	}
}

func TestRegressionPackageCompleteness(t *testing.T) {
	pkg, c, _ := producedPackage(t)
	set := &RegressionSet{Version: 1, Name: "core-regression", Set: 1,
		Members: []string{"bench-score-delta@1", "second-check@1"}, SHA256: strings.Repeat("66", 32)}

	t.Run("incomplete has no set package", func(t *testing.T) {
		_, err := BuildRegressionPackage("core-regression@1", set, map[string]*ComparisonPackage{
			"bench-score-delta@1": pkg,
		}, map[string]*Criterion{"bench-score-delta@1": c, "second-check@1": c})
		if err == nil {
			t.Fatal("partial coverage produced a set package")
		}
		if !strings.Contains(err.Error(), "no set-level package exists") {
			t.Fatalf("wrong error: %v", err)
		}
	})
	t.Run("member identity must match", func(t *testing.T) {
		_, err := BuildRegressionPackage("core-regression@1", set, map[string]*ComparisonPackage{
			"bench-score-delta@1": pkg,
			"second-check@1":      pkg, // produced under bench-score-delta@1
		}, map[string]*Criterion{"bench-score-delta@1": c, "second-check@1": c})
		if err == nil || !strings.Contains(err.Error(), "member identity") {
			t.Fatalf("criterion mismatch accepted: %v", err)
		}
	})
	t.Run("complete builds, no resistant flag stored", func(t *testing.T) {
		small := &RegressionSet{Version: 1, Name: "core-regression", Set: 1,
			Members: []string{"bench-score-delta@1"}, SHA256: strings.Repeat("66", 32)}
		sp, err := BuildRegressionPackage("core-regression@1", small, map[string]*ComparisonPackage{
			"bench-score-delta@1": pkg,
		}, map[string]*Criterion{"bench-score-delta@1": c})
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(sp)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		for _, forbidden := range []string{"resistant", "coverage", "partial", "status", "cases_completed"} {
			if _, ok := m[forbidden]; ok {
				t.Fatalf("set package stores forbidden field %q", forbidden)
			}
		}
		// resistant-under-S is derivable, statelessly.
		within, err := DeriveResistantUnderSet(small,
			map[string]*Criterion{"bench-score-delta@1": c},
			map[string]*ComparisonPackage{"bench-score-delta@1": pkg})
		if err != nil || !within {
			t.Fatalf("derivation failed: %v %v", within, err)
		}
	})
	t.Run("regression found is first-class", func(t *testing.T) {
		in := validInput(t, 0.30, 0.82) // far outside the region
		worse, ref, err := Compare(in)
		if err != nil || ref != nil {
			t.Fatal("unfavorable comparison refused")
		}
		small := &RegressionSet{Version: 1, Name: "core-regression", Set: 1,
			Members: []string{"bench-score-delta@1"}, SHA256: strings.Repeat("66", 32)}
		if _, err := BuildRegressionPackage("core-regression@1", small, map[string]*ComparisonPackage{
			"bench-score-delta@1": worse,
		}, map[string]*Criterion{"bench-score-delta@1": in.Criterion}); err != nil {
			t.Fatalf("complete-with-regression package refused — completeness is coverage, not favorability: %v", err)
		}
		within, err := DeriveResistantUnderSet(small,
			map[string]*Criterion{"bench-score-delta@1": in.Criterion},
			map[string]*ComparisonPackage{"bench-score-delta@1": worse})
		if err != nil || within {
			t.Fatalf("out-of-region derived as resistant: %v %v", within, err)
		}
	})
}

// D-L11-9 Amendment 1: resistant-under-S is recomputed from the
// constituent packages and their criteria. The criterion supplied for
// that recomputation must be the one the constituent comparison was
// CONDITIONED on — its hash is part of the package's conditioning
// tuple for exactly this reason.
//
// Without the binding, the derivation is a question you can answer by
// choosing the question: pair a regressed comparison with a more
// permissive criterion and "resistant under S" comes out true. The
// substitute here is a real criterion the registry would accept; only
// its non-regression region is wider.
func TestResistantUnderSetRefusesUnboundCriterion(t *testing.T) {
	in := validInput(t, 0.30, 0.82) // -0.52: far outside the fixture region
	worse, ref, err := Compare(in)
	if err != nil || ref != nil {
		t.Fatalf("unfavorable comparison refused: %v %v", ref, err)
	}
	set := &RegressionSet{Version: 1, Name: "core-regression", Set: 1,
		Members: []string{"bench-score-delta@1"}, SHA256: strings.Repeat("66", 32)}

	// A criterion wide enough to call -0.52 a non-regression.
	m := validCriterionMap()
	m["config"] = map[string]any{"fields": []any{map[string]any{
		"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score"}}}
	m["ordering"] = map[string]any{"kind": "per-metric", "fields": []any{
		map[string]any{"name": "score_delta", "direction": "maximize",
			"equal_tolerance": 0.0, "non_regression_min": -1.0}}}
	lenient, perr := ParseCriterion(marshalCriterion(t, m), "lenient")
	if perr != nil {
		t.Fatalf("the substitute must be a criterion the registry accepts: %v", perr)
	}
	if lenient.SHA256 == worse.CriterionSHA256 {
		t.Fatal("the substitute hashes to the conditioning criterion — it is not a substitute")
	}
	// Premise: the substitute really would flip the answer, so the
	// refusal below is the binding and not a coincidence of region.
	if within, derr := DeriveNonRegression(lenient, worse.Delta); derr != nil || !within {
		t.Fatalf("the substitute must accept this delta, or nothing is being bypassed: %v %v", within, derr)
	}

	within, err := DeriveResistantUnderSet(set,
		map[string]*Criterion{"bench-score-delta@1": lenient},
		map[string]*ComparisonPackage{"bench-score-delta@1": worse})
	if err == nil {
		t.Fatalf("a criterion that did not condition the comparison was ACCEPTED (derived resistant=%v)", within)
	}
	if !errors.Is(err, ErrResolve) {
		t.Errorf("refused as %v, want a resolve refusal", err)
	}
	if !strings.Contains(err.Error(), "conditioning tuple") {
		t.Errorf("refused for the wrong reason: %v", err)
	}
	if within {
		t.Error("a refusing derivation must not also report resistant")
	}

	// And the bound criterion still derives, so the check narrows to
	// substitution rather than refusing everything.
	if _, derr := DeriveResistantUnderSet(set,
		map[string]*Criterion{"bench-score-delta@1": in.Criterion},
		map[string]*ComparisonPackage{"bench-score-delta@1": worse}); derr != nil {
		t.Fatalf("the conditioning criterion must still derive: %v", derr)
	}
}
