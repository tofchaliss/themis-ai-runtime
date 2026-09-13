package main

// Compiled-binary tests (test review §3): the invocation surface's
// contract lives here and nowhere else — refusal → exit 0 + durably
// stored; machinery/usage error → exit 1 + nothing minted. The
// reconstruct run doubles as the cross-process determinism evidence
// for D-L11-17 (test review §4): a separate process re-derives the
// package produced by another process.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/ratchet"
	"github.com/tofchaliss/themis/state"
)

type cliWorld struct {
	bin, gov, stateRoot, evidenceRoot string
	criterionPath, doorPath           string
	criteriaReg                       string
	candFacts, baseFacts              string
}

func buildWorld(t *testing.T) *cliWorld {
	t.Helper()
	w := &cliWorld{}
	w.bin = filepath.Join(t.TempDir(), "themis-ratchet")
	build := exec.Command("go", "build", "-o", w.bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	w.gov = t.TempDir()
	w.stateRoot = filepath.Join(t.TempDir(), "state")
	w.evidenceRoot = t.TempDir()

	// Criterion + registry (Governance fixtures).
	criterion := map[string]any{
		"version": 1, "name": "bench-score-delta", "criterion_version": 1,
		"families": []any{"skill-revision"},
		"candidate_selectors": []any{map[string]any{
			"name": "candidate_score", "source": "benchmark_validated_score",
			"params": map[string]any{"benchmark": "themis-bench-core"}}},
		"baseline_selectors": []any{map[string]any{
			"name": "baseline_score", "source": "benchmark_validated_score",
			"params": map[string]any{"benchmark": "themis-bench-core"}}},
		"comparator": map[string]any{"name": "numeric-score-delta", "version": 1},
		"config": map[string]any{"fields": []any{map[string]any{
			"delta": "score_delta", "candidate": "candidate_score", "baseline": "baseline_score"}}},
		"delta_shape": []any{map[string]any{"name": "score_delta", "type": "number"}},
		"ordering": map[string]any{"kind": "per-metric", "fields": []any{map[string]any{
			"name": "score_delta", "direction": "maximize",
			"equal_tolerance": 0.0, "non_regression_min": -0.05}}},
		"baseline_constraints": []any{"requires-current-active"},
		"provenance":           []any{"evidence_records", "run_identities", "admission_observation"},
	}
	cb, _ := json.Marshal(criterion)
	w.criterionPath = filepath.Join(w.gov, "criterion.json")
	os.WriteFile(w.criterionPath, cb, 0o644)
	regB, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "criteria",
		"entries": []any{map[string]any{
			"name": "bench-score-delta", "version": 1,
			"artifact_sha256": sha256hex(cb), "artifact_path": "criterion.json",
			"state": "active"}},
	})
	w.criteriaReg = filepath.Join(w.gov, "criteria-registry.json")
	os.WriteFile(w.criteriaReg, regB, 0o644)

	// Door registry (the baseline's owning door).
	doorB, _ := json.Marshal(map[string]any{
		"version": 1,
		"entries": []any{map[string]any{
			"name": "investigate-cve", "version": 1,
			"composition_sha256": strings.Repeat("22", 32), "state": "active"}},
	})
	w.doorPath = filepath.Join(w.gov, "l9-catalog.json")
	os.WriteFile(w.doorPath, doorB, 0o644)

	// External-plane evidence records + refs.
	candRecord := []byte(`{"score":0.91,"benchmark":"themis-bench-core"}`)
	baseRecord := []byte(`{"score":0.82,"benchmark":"themis-bench-core"}`)
	os.WriteFile(filepath.Join(w.evidenceRoot, "cand.json"), candRecord, 0o644)
	os.WriteFile(filepath.Join(w.evidenceRoot, "base.json"), baseRecord, 0o644)
	cf, _ := json.Marshal([]map[string]any{{
		"selector": "candidate_score", "source": "benchmark_validated_score",
		"ref": "cand.json", "sha256": sha256hex(candRecord), "value": json.RawMessage(candRecord)}})
	bf, _ := json.Marshal([]map[string]any{{
		"selector": "baseline_score", "source": "benchmark_validated_score",
		"ref": "base.json", "sha256": sha256hex(baseRecord), "value": json.RawMessage(baseRecord)}})
	w.candFacts = filepath.Join(w.gov, "cand-facts.json")
	w.baseFacts = filepath.Join(w.gov, "base-facts.json")
	os.WriteFile(w.candFacts, cf, 0o644)
	os.WriteFile(w.baseFacts, bf, 0o644)
	return w
}

func sha256hex(b []byte) string { return ratchet.InstanceID(b) }

func (w *cliWorld) compare(t *testing.T, extra ...string) (string, int) {
	t.Helper()
	args := append([]string{"compare",
		"--criteria-registry", w.criteriaReg,
		"--criterion", "bench-score-delta@1",
		"--candidate", strings.Repeat("33", 32),
		"--claimed-baseline", strings.Repeat("22", 32),
		"--door", "l9-catalog",
		"--door-registry", w.doorPath,
		"--baseline", "investigate-cve@1",
		"--observed-at", "2026-09-13T00:00:00Z",
		"--candidate-facts", w.candFacts,
		"--baseline-facts", w.baseFacts,
		"--evidence-root", w.evidenceRoot,
		"--runs", "l4:101,l4:102",
		"--state-root", w.stateRoot,
	}, extra...)
	// Later flags override earlier ones in the stdlib flag package —
	// extras replace defaults.
	cmd := exec.Command(w.bin, args...)
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("exec: %v\n%s", err, out)
	}
	return string(out), code
}

func objectFrom(t *testing.T, out string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out)
	}
	id, _ := m["object"].(string)
	if id == "" {
		t.Fatalf("no object id in output: %s", out)
	}
	return id
}

func TestCLIContract(t *testing.T) {
	w := buildWorld(t)

	var pkgID string
	t.Run("happy path exits 0 and stores", func(t *testing.T) {
		out, code := w.compare(t)
		if code != 0 || !strings.Contains(out, `"kind": "comparison"`) {
			t.Fatalf("code=%d out=%s", code, out)
		}
		pkgID = objectFrom(t, out)
		root, _ := state.OpenRoot(w.stateRoot)
		if _, err := root.Store().GetObject(pkgID); err != nil {
			t.Fatalf("emitted object not durable: %v", err)
		}
	})

	t.Run("refusal exits 0 and stores the fact", func(t *testing.T) {
		// Baseline absent at the door → unadmitted refusal, durable.
		out, code := w.compare(t, "--baseline", "no-such-skill@1")
		if code != 0 || !strings.Contains(out, `"kind": "refusal"`) {
			t.Fatalf("refusal contract broken: code=%d out=%s", code, out)
		}
		id := objectFrom(t, out)
		root, _ := state.OpenRoot(w.stateRoot)
		if _, err := root.Store().GetObject(id); err != nil {
			t.Fatalf("refusal not durable: %v", err)
		}
	})

	t.Run("unregistered criterion is a durable refusal", func(t *testing.T) {
		out, code := w.compare(t, "--criterion", "shadow-criterion@1")
		if code != 0 || !strings.Contains(out, string(ratchet.ReasonUnregisteredArtifact)) {
			t.Fatalf("unregistered-artifact contract broken: code=%d out=%s", code, out)
		}
	})

	t.Run("malformed request is a usage error, nothing minted", func(t *testing.T) {
		before := countObjects(t, w.stateRoot)
		_, code := w.compare(t, "--candidate", "not-a-hash")
		if code != 1 {
			t.Fatalf("usage error exited %d", code)
		}
		if after := countObjects(t, w.stateRoot); after != before {
			t.Fatal("usage error minted an object")
		}
	})

	t.Run("registry pin mismatch is a durable refusal", func(t *testing.T) {
		out, code := w.compare(t, "--registry-sha256", strings.Repeat("77", 32))
		if code != 0 || !strings.Contains(out, string(ratchet.ReasonIntegrityFailure)) {
			t.Fatalf("pin contract broken: code=%d out=%s", code, out)
		}
	})

	t.Run("cross-process reconstruction confirms", func(t *testing.T) {
		if pkgID == "" {
			t.Skip("no package from happy path")
		}
		cmd := exec.Command(w.bin, "reconstruct",
			"--state-root", w.stateRoot,
			"--package", pkgID,
			"--criterion-file", w.criterionPath,
			"--door-registry", w.doorPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("reconstruct: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), `"result": "confirmed"`) {
			t.Fatalf("cross-process reconstruction not confirmed: %s", out)
		}
	})

	t.Run("set refuses non-member binding", func(t *testing.T) {
		setB, _ := json.Marshal(map[string]any{"version": 1, "name": "core-regression",
			"set_version": 1, "members": []any{"bench-score-delta@1"}})
		os.WriteFile(filepath.Join(w.gov, "set.json"), setB, 0o644)
		setsRegB, _ := json.Marshal(map[string]any{"version": 1, "kind": "regression-sets",
			"entries": []any{map[string]any{"name": "core-regression", "version": 1,
				"artifact_sha256": sha256hex(setB), "artifact_path": "set.json", "state": "active"}}})
		setsReg := filepath.Join(w.gov, "sets-registry.json")
		os.WriteFile(setsReg, setsRegB, 0o644)
		cmd := exec.Command(w.bin, "set",
			"--sets-registry", setsReg,
			"--criteria-registry", w.criteriaReg,
			"--set", "core-regression@1",
			"--state-root", w.stateRoot,
			"--member", "rogue-criterion@1=sha256:"+strings.Repeat("aa", 32))
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("non-member binding accepted: %s", out)
		}
	})
}

func countObjects(t *testing.T, stateRoot string) int {
	t.Helper()
	n := 0
	filepath.WalkDir(filepath.Join(stateRoot, "objects"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
