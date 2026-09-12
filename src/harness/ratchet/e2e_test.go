package ratchet

// M6 Register: the full governed slice — authored candidate →
// Governance registration fixtures (the owner act, simulated as
// on-disk governed files exactly as L9/L10 registers did) →
// comparison under exact pins → set-level regression evidence →
// stateless derivations → cold reconstruction → and the structural
// proof that L11 wrote none of the governed registries it read.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/state"
)

func TestRatchetEndToEndSlice(t *testing.T) {
	// ---- Governance plane (fixtures = owner acts, the L9/L10 test
	// precedent). L11 only ever READS these.
	govDir := t.TempDir()
	criterionRaw := marshalCriterion(t, func() map[string]any {
		m := validCriterionMap()
		m["config"] = map[string]any{"fields": []any{map[string]any{
			"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score",
		}}}
		return m
	}())
	if err := os.MkdirAll(filepath.Join(govDir, "criteria"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(govDir, "criteria", "bench-score-delta.json"), criterionRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	criteriaRegRaw, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "criteria",
		"entries": []any{map[string]any{
			"name": "bench-score-delta", "version": 1,
			"artifact_sha256": hashBytes(criterionRaw),
			"artifact_path":   "criteria/bench-score-delta.json",
			"state":           "active", "steward": "owner",
		}},
	})
	criteriaRegPath := filepath.Join(govDir, "criteria-registry.json")
	if err := os.WriteFile(criteriaRegPath, criteriaRegRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	setRaw, _ := json.Marshal(map[string]any{
		"version": 1, "name": "core-regression", "set_version": 1,
		"members": []any{"bench-score-delta@1"},
	})
	if err := os.WriteFile(filepath.Join(govDir, "core-regression.json"), setRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	setsRegRaw, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "regression-sets",
		"entries": []any{map[string]any{
			"name": "core-regression", "version": 1,
			"artifact_sha256": hashBytes(setRaw),
			"artifact_path":   "core-regression.json",
			"state":           "active",
		}},
	})
	setsRegPath := filepath.Join(govDir, "sets-registry.json")
	if err := os.WriteFile(setsRegPath, setsRegRaw, 0o644); err != nil {
		t.Fatal(err)
	}
	// The door the baseline lives at (an L9-catalog-shaped fixture):
	// its bytes ground the admission observation.
	baselineContent := []byte(`{"skill":"investigate-cve","version":1}`)
	doorRegistryRaw, _ := json.Marshal(map[string]any{
		"catalog": []any{map[string]any{"name": "investigate-cve", "version": 1,
			"sha256": hashBytes(baselineContent), "state": "active"}},
	})

	govSnapshot := func() map[string]string {
		out := map[string]string{}
		_ = filepath.WalkDir(govDir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, _ := os.ReadFile(p)
			out[p] = hashBytes(b)
			return nil
		})
		return out
	}
	before := govSnapshot()

	// ---- L6 record plane.
	root, err := state.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store := root.Store()

	// ---- 1. An AUTHORED candidate (attributable act; machinery
	// mints nothing — this test plays the author, as a human would).
	proposedContent := []byte(`{"skill":"investigate-cve","version":2,"change":"narrower tool grants"}`)
	candMap := validCandidateMap()
	candMap["content_sha256"] = hashBytes(proposedContent)
	candMap["claimed_baseline"].(map[string]any)["artifact_sha256"] = hashBytes(baselineContent)
	candRaw, _ := json.Marshal(candMap)
	cand, err := ParseCandidate(candRaw, "e2e")
	if err != nil {
		t.Fatal(err)
	}
	candID, _, err := StoreInstance(store, cand)
	if err != nil {
		t.Fatal(err)
	}

	// ---- 2. An evaluation plan (inert data; nothing runs because of
	// it — the runs below are "ordinary governed runs" represented by
	// their recorded facts).
	planMap := validPlanMap()
	planMap["candidate_hash"] = hashBytes(proposedContent)
	planRaw, _ := json.Marshal(planMap)
	plan, err := ParsePlan(planRaw, "e2e")
	if err != nil {
		t.Fatal(err)
	}
	planID, _, err := StoreInstance(store, plan)
	if err != nil {
		t.Fatal(err)
	}

	// ---- 3. Comparison under exact pins, conditioned on the door
	// observation.
	reg, err := LoadRegistry(criteriaRegPath)
	if err != nil {
		t.Fatal(err)
	}
	_, criterion, err := reg.ResolveCriterion("bench-score-delta@1")
	if err != nil {
		t.Fatal(err)
	}
	admission := &AdmissionObservation{
		Door: "l9-catalog", DoorRegistryHash: hashBytes(doorRegistryRaw),
		Name: "investigate-cve", Version: 1,
		ArtifactSHA256: hashBytes(baselineContent),
		State:          "active", CurrentActive: true,
		ObservedAt: "2026-09-12T12:00:00Z",
	}
	in := CompareInput{
		CriterionRef:    "bench-score-delta@1",
		Criterion:       criterion,
		CandidateHash:   hashBytes(proposedContent),
		ClaimedBaseline: hashBytes(baselineContent),
		Admission:       admission,
		CandidateFacts:  []EvidenceRef{fact("candidate_score", "benchmark_validated_score", 0.91)},
		BaselineFacts:   []EvidenceRef{fact("baseline_score", "benchmark_validated_score", 0.82)},
		RunIdentities:   []string{"l4:101", "l4:102"},
		PlanRef:         planID,
	}
	pkg, refusal, err := Compare(in)
	if err != nil || refusal != nil {
		t.Fatalf("comparison failed: %v %v", refusal, err)
	}
	pkgID, pkgBytes, err := StoreInstance(store, pkg)
	if err != nil {
		t.Fatal(err)
	}

	// ---- 4. Plan conformance: mechanical record-to-plan matching.
	if mm := CheckPlanConformance(plan, planID, pkg); len(mm) != 0 {
		t.Fatalf("conformance mismatches: %v", mm)
	}

	// ---- 5. Set-level regression evidence (complete-under-S).
	setsReg, err := LoadRegistry(setsRegPath)
	if err != nil {
		t.Fatal(err)
	}
	_, set, err := setsReg.ResolveSet("core-regression@1")
	if err != nil {
		t.Fatal(err)
	}
	sp, err := BuildRegressionPackage("core-regression@1", set, map[string]*ComparisonPackage{
		"bench-score-delta@1": pkg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := StoreInstance(store, sp); err != nil {
		t.Fatal(err)
	}

	// ---- 6. Stateless derivations for the door's reading.
	perField, err := DerivePerField(criterion, pkg.Delta)
	if err != nil || perField[0].Relation != RelBetter {
		t.Fatalf("derivation: %v %v", perField, err)
	}
	resistant, err := DeriveResistantUnderSet(set,
		map[string]*Criterion{"bench-score-delta@1": criterion},
		map[string]*ComparisonPackage{"bench-score-delta@1": pkg})
	if err != nil || !resistant {
		t.Fatalf("resistant derivation: %v %v", resistant, err)
	}

	// ---- 7. The PROMOTION ACT belongs to the door and the human:
	// nothing in this package can perform it. Structural proof: after
	// the entire arc, every Governance file is byte-identical — L11
	// read the registries and wrote nothing anywhere in the plane.
	after := govSnapshot()
	if len(before) != len(after) {
		t.Fatal("governance plane file set changed during the L11 arc")
	}
	for p, h := range before {
		if after[p] != h {
			t.Fatalf("governance artifact %s changed during the L11 arc", p)
		}
	}

	// ---- 8. Cold reconstruction from durable bytes only.
	stored, err := store.GetObject(pkgID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, pkgBytes) {
		t.Fatal("stored bytes differ from canonical bytes")
	}
	rec, err := Reconstruct(stored, criterionRaw)
	if err != nil || rec.Result != ReconConfirmed {
		t.Fatalf("cold reconstruction: %+v %v", rec, err)
	}

	// ---- 9. The refusal arc is equally durable: a withdrawn
	// baseline refuses with a recorded fact, not a weaker package.
	inRefused := in
	admWithdrawn := *admission
	admWithdrawn.State = "withdrawn"
	inRefused.Admission = &admWithdrawn
	p2, r2, err := Compare(inRefused)
	if err != nil || p2 != nil || r2 == nil {
		t.Fatalf("withdrawn arc: %v %v %v", p2, r2, err)
	}
	if _, _, err := StoreInstance(store, r2); err != nil {
		t.Fatal(err)
	}

	// ---- 10. Candidate and package identities are content
	// addresses; nothing carries a lifecycle state.
	for _, id := range []string{candID, planID, pkgID} {
		if !strings.HasPrefix(id, "sha256:") {
			t.Fatalf("instance %s is not content-addressed", id)
		}
	}
}

// The knowledge family: the candidate/egress REPRESENTATION is
// constructible and durable (family-agnostic machinery), but the
// handoff to the Themis ingestion door is NOT proven here — the door
// is not available in this environment, so per the Gate 0 rule the
// family remains explicitly unproved/residual and no substitute
// harness-side knowledge authority exists (none is even
// representable: the store call below uses the EXISTING egress
// object class and nothing reads it back as knowledge).
func TestKnowledgeCandidateRepresentationOnly(t *testing.T) {
	root, err := state.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := validCandidateMap()
	m["family"] = "knowledge"
	m["target"] = "themis-knowledge:cve-remediation-pattern"
	b, _ := json.Marshal(m)
	cand, err := ParseCandidate(b, "test")
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := CanonicalBytes(cand)
	if err != nil {
		t.Fatal(err)
	}
	// Egress representation: the existing governed egress class.
	id, err := root.Store().StoreObject(state.ObjEgressArtifact, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "sha256:") {
		t.Fatal("egress representation is not content-addressed")
	}
	// Successful handoff would NOT mean "knowledge promoted"
	// (D-L11-3); here not even handoff occurs. Residual recorded in
	// tasks.md.
}
