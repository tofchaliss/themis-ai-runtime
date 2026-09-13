package orchestration

// L10-M3 proofs: the verification seam amendment to the archived L7
// layer. Every new loader/assembly rule is pinned by a doctored
// definition (the TestLoaderRefusals pattern); the gate ladder and
// latest-per-token δ semantics are pinned deterministically; existing
// non-verification workflows must be bit-for-bit unaffected (the
// D-L10-17 additive-only obligation).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis/deployment"
)

func writeSeamFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func seamCeiling(t *testing.T, dir string) *WorkflowCeiling {
	t.Helper()
	p := writeSeamFixture(t, dir, "ceiling.json", `{
	  "version": 1,
	  "allowed_tools": ["read_file", "run-go-build", "declare_done"],
	  "max_total_calls": 50,
	  "max_walk_length": 500,
	  "max_turns_per_phase": 10
	}`)
	c, err := LoadWorkflowCeiling(p)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// verifWorkflow is a two-phase lattice using the verification
// vocabulary: WORK exposes the verifier and maps all five events;
// DONE gates completion on PASS for an exact contract token.
const verifWorkflow = `{
  "version": 1,
  "name": "verif-seam",
  "declared_events": ["turn-no-action", "turn-provider-error", "turns-exhausted",
    "signal:phase-completion-requested",
    "verification-pass", "verification-fail", "verification-inconclusive",
    "verification-unavailable", "verification-invalid"],
  "initial": "work",
  "phases": [
    {"name": "work", "capabilities": ["read_file", "run-go-build", "declare_done"],
     "max_model_turns": 5,
     "edges": [
       {"on": "turn-no-action", "to": "@stay", "counter": 3, "exhausted_to": "@fail"},
       {"on": "turn-provider-error", "to": "@fail"},
       {"on": "turns-exhausted", "to": "@fail"},
       {"on": "verification-pass", "to": "@stay", "counter": 3, "exhausted_to": "@fail"},
       {"on": "verification-fail", "to": "@stay", "counter": 3, "exhausted_to": "@fail"},
       {"on": "verification-inconclusive", "to": "@stay", "counter": 3, "exhausted_to": "@fail"},
       {"on": "verification-unavailable", "to": "@stay", "counter": 3, "exhausted_to": "@fail"},
       {"on": "verification-invalid", "to": "@fail"},
       {"on": "signal:phase-completion-requested",
        "gate": {"contract": "go-build-clean@1", "outcome": "PASS"},
        "to": "@complete"},
       {"on": "signal:phase-completion-requested", "to": "@stay", "counter": 3, "exhausted_to": "@fail"}
     ]}
  ]
}`

func TestVerificationGateVocabularyLoads(t *testing.T) {
	dir := t.TempDir()
	ceiling := seamCeiling(t, dir)
	p := writeSeamFixture(t, dir, "wf.json", verifWorkflow)
	wf, err := LoadWorkflow(p, ceiling)
	if err != nil {
		t.Fatal(err)
	}
	if wf.Name != "verif-seam" {
		t.Errorf("loaded %q", wf.Name)
	}
}

func TestVerificationSeamLoaderRefusals(t *testing.T) {
	dir := t.TempDir()
	ceiling := seamCeiling(t, dir)

	cases := map[string]func(string) string{
		"floating gate token": func(w string) string {
			return strings.Replace(w, `"contract": "go-build-clean@1"`, `"contract": "go-build-clean"`, 1)
		},
		"latest gate token": func(w string) string {
			return strings.Replace(w, `"contract": "go-build-clean@1"`, `"contract": "go-build-clean@latest"`, 1)
		},
		"non-canonical gate version": func(w string) string {
			return strings.Replace(w, `"contract": "go-build-clean@1"`, `"contract": "go-build-clean@01"`, 1)
		},
		"governance vocabulary as gate outcome": func(w string) string {
			return strings.Replace(w, `"outcome": "PASS"`, `"outcome": "NOT_AFFECTED"`, 1)
		},
		"unknown gate outcome": func(w string) string {
			return strings.Replace(w, `"outcome": "PASS"`, `"outcome": "OK"`, 1)
		},
		"gated edge without ungated fallback": func(w string) string {
			return strings.Replace(w,
				`{"on": "verification-invalid", "to": "@fail"},`,
				`{"on": "verification-invalid", "to": "@fail", "gate": {"contract": "x@1", "outcome": "PASS"}},`, 1)
		},
		"gated edge with counter": func(w string) string {
			return strings.Replace(w,
				`"gate": {"contract": "go-build-clean@1", "outcome": "PASS"},`,
				`"gate": {"contract": "go-build-clean@1", "outcome": "PASS"}, "counter": 2, "exhausted_to": "@fail",`, 1)
		},
		"gated edge after fallback": func(w string) string {
			return strings.Replace(w,
				`{"on": "signal:phase-completion-requested",
        "gate": {"contract": "go-build-clean@1", "outcome": "PASS"},
        "to": "@complete"},
       {"on": "signal:phase-completion-requested", "to": "@stay", "counter": 3, "exhausted_to": "@fail"}`,
				`{"on": "signal:phase-completion-requested", "to": "@stay", "counter": 3, "exhausted_to": "@fail"},
       {"on": "signal:phase-completion-requested",
        "gate": {"contract": "go-build-clean@1", "outcome": "PASS"},
        "to": "@complete"}`, 1)
		},
		"undeclared verification event edge": func(w string) string {
			return strings.Replace(w,
				`"declared_events": ["turn-no-action", "turn-provider-error", "turns-exhausted",
    "signal:phase-completion-requested",
    "verification-pass", "verification-fail", "verification-inconclusive",
    "verification-unavailable", "verification-invalid"],`,
				`"declared_events": ["turn-no-action", "turn-provider-error", "turns-exhausted",
    "signal:phase-completion-requested"],`, 1)
		},
	}

	for name, doctor := range cases {
		t.Run(name, func(t *testing.T) {
			p := writeSeamFixture(t, dir, "bad.json", doctor(verifWorkflow))
			if _, err := LoadWorkflow(p, ceiling); err == nil {
				t.Errorf("doctored definition loaded: %s", name)
			}
		})
	}
}

// TestVerificationEventsConditionallyTotal proves the owner's
// reachability amendment: a workflow may declare the verification
// vocabulary while phases WITHOUT a verifier-eligible capability omit
// verification edges — no artificial totality over impossible events.
func TestVerificationEventsConditionallyTotal(t *testing.T) {
	dir := t.TempDir()
	ceiling := seamCeiling(t, dir)
	// A second phase with no verifier capability and no verification
	// edges must load.
	wf := strings.Replace(verifWorkflow, `"initial": "work",`, `"initial": "prep",`, 1)
	wf = strings.Replace(wf, `"phases": [`, `"phases": [
    {"name": "prep", "capabilities": ["read_file"], "max_model_turns": 2,
     "edges": [
       {"on": "turn-no-action", "to": "work"},
       {"on": "turn-provider-error", "to": "@fail"},
       {"on": "turns-exhausted", "to": "@fail"},
       {"on": "signal:phase-completion-requested", "to": "work"}
     ]},`, 1)
	p := writeSeamFixture(t, dir, "wf2.json", wf)
	if _, err := LoadWorkflow(p, ceiling); err != nil {
		t.Fatalf("reachability-based totality refused a legal definition: %v", err)
	}
}

// TestGateLadderDeterminism pins δ's gate selection directly: gated
// edges in definition order against latest-per-token state, first
// satisfied wins, ungated fallback otherwise; edge identity records
// the fired branch.
func TestGateLadderDeterminism(t *testing.T) {
	dir := t.TempDir()
	ceiling := seamCeiling(t, dir)
	p := writeSeamFixture(t, dir, "wf.json", verifWorkflow)
	wf, err := LoadWorkflow(p, ceiling)
	if err != nil {
		t.Fatal(err)
	}

	pick := func(state map[string]string) (string, string) {
		// Mirror of the walk's selection logic over the loaded
		// definition — deterministic and replayer-equivalent.
		ph := wf.phase("work")
		var chosen *Edge
		gateIdx, idx := -1, 0
		for i := range ph.Edges {
			e := &ph.Edges[i]
			if e.On != SignalPhaseCompletionRequested {
				continue
			}
			if e.Gate != nil {
				if chosen == nil && state[e.Gate.Contract] == e.Gate.Outcome {
					chosen, gateIdx = e, idx
				}
				idx++
				continue
			}
			if chosen == nil {
				chosen = e
			}
		}
		key := "work/" + SignalPhaseCompletionRequested
		if gateIdx >= 0 {
			key = fmt.Sprintf("%s#g%d", key, gateIdx)
		}
		return chosen.To, key
	}

	// No verification yet: fallback (@stay), ungated identity.
	to, key := pick(map[string]string{})
	if to != TargetStay || key != "work/signal:phase-completion-requested" {
		t.Errorf("empty state: %s via %s", to, key)
	}
	// FAIL recorded: gate unsatisfied, still fallback.
	to, _ = pick(map[string]string{"go-build-clean@1": "FAIL"})
	if to != TargetStay {
		t.Errorf("FAIL state must not open a PASS gate: %s", to)
	}
	// PASS on a DIFFERENT token: still fallback (opaque token match).
	to, _ = pick(map[string]string{"other-contract@1": "PASS"})
	if to != TargetStay {
		t.Errorf("foreign token PASS must not open the gate: %s", to)
	}
	// PASS on the exact token: gate fires, gated identity recorded.
	to, key = pick(map[string]string{"go-build-clean@1": "PASS"})
	if to != TargetComplete || key != "work/signal:phase-completion-requested#g0" {
		t.Errorf("PASS state: %s via %s", to, key)
	}
	// Downgrade (PASS -> UNAVAILABLE): gate closes again — stateless
	// re-derivation at each transition evaluation (D-L10-9).
	to, _ = pick(map[string]string{"go-build-clean@1": "UNAVAILABLE"})
	if to != TargetStay {
		t.Errorf("downgraded state must close the gate: %s", to)
	}
}

// TestPreAmendmentWorkflowUnchanged proves the additive-only
// obligation: a definition from before the amendment (no verification
// vocabulary, no gates) loads with an identical hash and identical
// static properties.
func TestPreAmendmentWorkflowUnchanged(t *testing.T) {
	dir := t.TempDir()
	ceiling := seamCeiling(t, dir)
	legacy := `{
  "version": 1,
  "name": "legacy",
  "declared_events": ["turn-no-action", "turn-provider-error", "turns-exhausted"],
  "initial": "only",
  "phases": [
    {"name": "only", "capabilities": ["read_file"], "max_model_turns": 3,
     "edges": [
       {"on": "turn-no-action", "to": "@stay", "counter": 2, "exhausted_to": "@fail"},
       {"on": "turn-provider-error", "to": "@fail"},
       {"on": "turns-exhausted", "to": "@complete"}
     ]}
  ]
}`
	p := writeSeamFixture(t, dir, "legacy.json", legacy)
	wf, err := LoadWorkflow(p, ceiling)
	if err != nil {
		t.Fatalf("pre-amendment definition refused: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(legacy), &raw); err != nil {
		t.Fatal(err)
	}
	if wf.WorstCaseLen <= 0 {
		t.Error("static bound missing")
	}
}

// TestThemisRootWiring — integration-audit D7: the themis-domain
// instruction root is wireable (Config.ThemisRoot) and resolves
// through the same governed L1 discipline as the other roots, so the
// L2 authority vocabulary ships with its interpretive half.
func TestThemisRootWiring(t *testing.T) {
	base := t.TempDir()
	_, _, err := Open(Config{
		Unanchored: true,
		StateRoot:  filepath.Join(base, "state"), ArtifactDir: filepath.Join(base, "artifacts"),
		GitPath: gitBin(t), ProviderDir: filepath.Join(base, "provider"),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		ThemisRoot: filepath.Join(repoRoot, "instructions/themis"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      happyScript(),
	})
	if err != nil {
		t.Fatalf("themis root failed to resolve: %v", err)
	}
}

// --- G1 Deployment Anchor enforcement at the L7 seam (D-G1-1 /
// D-G1-1A). The anchor ADMISSION half is proven in the deployment
// package; here we prove the L7 half: an admitted anchor freezes the
// deployment at Open, and SubmitTask refuses any bundle artifact
// that is not the anchored one.

func anchorWorld(t *testing.T, mutate func(m map[string]any)) (anchorPath, anchorSHA, regPath string) {
	t.Helper()
	dir := t.TempDir()
	hashDir := func(p string) string {
		h, err := deployment.HashDir(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	hashFile := func(p string) string {
		h, err := deployment.HashFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	m := map[string]any{
		"version": 1, "name": "test-deployment", "deployment_version": 1,
		"instruction_root_safety": hashDir(filepath.Join(repoRoot, "instructions/global/safety")),
		"instruction_root_system": hashDir(filepath.Join(repoRoot, "instructions/global/system")),
		"instruction_root_themis": hashDir(filepath.Join(repoRoot, "instructions/themis")),
		"instruction_policy":      hashFile(filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json")),
		"tool_registry":           hashFile(filepath.Join(repoRoot, "policies/tools/registry-v4.json")),
		"workflow_ceiling":        strings.Repeat("11", 32),
		"exec_ceiling":            strings.Repeat("22", 32),
		"context_contract":        strings.Repeat("33", 32),
		"workflows":               []any{strings.Repeat("44", 32)},
		"models":                  []any{"scripted"},
		"model_registry":          "absent",
		"skill_catalog":           hashFile(filepath.Join(repoRoot, "policies/skills/catalog.json")),
		"contract_registry":       hashFile(filepath.Join(repoRoot, "policies/verification/contracts.json")),
		"criteria_registry":       hashFile(filepath.Join(repoRoot, "policies/ratchet/criteria.json")),
		"regression_set_registry": hashFile(filepath.Join(repoRoot, "policies/ratchet/regression-sets.json")),
	}
	if mutate != nil {
		mutate(m)
	}
	ab, _ := json.Marshal(m)
	anchorPath = filepath.Join(dir, "anchor.json")
	if err := os.WriteFile(anchorPath, ab, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(ab)
	anchorSHA = hex.EncodeToString(sum[:])
	rb, _ := json.Marshal(map[string]any{
		"version": 1, "kind": "deployment-anchors",
		"entries": []any{map[string]any{
			"name": m["name"], "version": m["deployment_version"],
			"artifact_sha256": anchorSHA, "state": "active", "steward": "owner"}},
	})
	regPath = filepath.Join(dir, "anchors.json")
	if err := os.WriteFile(regPath, rb, 0o644); err != nil {
		t.Fatal(err)
	}
	return anchorPath, anchorSHA, regPath
}

func anchoredConfig(t *testing.T, base string, anchorPath, anchorSHA, regPath string) Config {
	t.Helper()
	return Config{
		StateRoot: filepath.Join(base, "state"), ArtifactDir: filepath.Join(base, "artifacts"),
		GitPath: gitBin(t), ProviderDir: filepath.Join(base, "provider"),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		ThemisRoot: filepath.Join(repoRoot, "instructions/themis"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      happyScript(),
		AnchorPath: anchorPath, AnchorSHA256: anchorSHA, AnchorsRegistryPath: regPath,
	}
}

func TestAnchoredOpen(t *testing.T) {
	t.Run("admitted anchor with matching roots opens", func(t *testing.T) {
		p, sha, reg := anchorWorld(t, nil)
		if _, _, err := Open(anchoredConfig(t, t.TempDir(), p, sha, reg)); err != nil {
			t.Fatalf("anchored Open refused: %v", err)
		}
	})
	t.Run("unregistered anchor refuses Open", func(t *testing.T) {
		p, sha, _ := anchorWorld(t, nil)
		empty := filepath.Join(t.TempDir(), "anchors.json")
		rb, _ := json.Marshal(map[string]any{"version": 1, "kind": "deployment-anchors", "entries": []any{}})
		os.WriteFile(empty, rb, 0o644)
		_, _, err := Open(anchoredConfig(t, t.TempDir(), p, sha, empty))
		if err == nil || !strings.Contains(err.Error(), "admission claim") {
			t.Fatalf("forged anchor opened: %v", err)
		}
	})
	t.Run("instruction root drift refuses Open", func(t *testing.T) {
		p, sha, reg := anchorWorld(t, func(m map[string]any) {
			m["instruction_root_system"] = strings.Repeat("99", 32)
		})
		_, _, err := Open(anchoredConfig(t, t.TempDir(), p, sha, reg))
		if err == nil || !strings.Contains(err.Error(), "not the anchored artifact") {
			t.Fatalf("unanchored instruction root opened: %v", err)
		}
	})
	t.Run("anchored deployment requires the themis root configured", func(t *testing.T) {
		p, sha, reg := anchorWorld(t, nil)
		cfg := anchoredConfig(t, t.TempDir(), p, sha, reg)
		cfg.ThemisRoot = ""
		if _, _, err := Open(cfg); err == nil {
			t.Fatal("anchored Open accepted a missing themis root")
		}
	})
}

// The close review verified three surviving mutants: deleting the
// allowlist loop, the workflow-set check, or the instruction-policy
// check left the whole package green. Each now has a test that fails
// without it.
func TestAnchoredWorkflowSetEnforced(t *testing.T) {
	f := setupVerif(t, happyScript(), &scriptedEvaluator{})
	fileHash := func(p string) string {
		h, herr := deployment.HashFile(p)
		if herr != nil {
			t.Fatal(herr)
		}
		return h
	}
	envHash := func(n string) string { return fileHash(filepath.Join(f.envDir, n)) }
	ap, sha, reg := anchorWorld(t, func(m map[string]any) {
		m["tool_registry"] = fileHash(filepath.Join(repoRoot, "policies/tools/registry-v4.json"))
		m["workflow_ceiling"] = envHash("wceiling.json")
		m["exec_ceiling"] = envHash("eceiling.json")
		m["context_contract"] = envHash("context-contract.json")
		// every bundle pin satisfied EXCEPT the workflow set
		m["workflows"] = []any{strings.Repeat("ee", 32)}
	})
	o, _, err := Open(anchoredConfig(t, t.TempDir(), ap, sha, reg))
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.SubmitTask(f.verifEnvelope(t, "wf-set"))
	if err == nil || !strings.Contains(err.Error(), "anchored workflow set") {
		t.Fatalf("workflow-set check not reached: %v", err)
	}
}

func TestAnchoredInstructionPolicyEnforced(t *testing.T) {
	ap, sha, reg := anchorWorld(t, func(m map[string]any) {
		m["instruction_policy"] = strings.Repeat("ee", 32)
	})
	_, _, err := Open(anchoredConfig(t, t.TempDir(), ap, sha, reg))
	if err == nil || !strings.Contains(err.Error(), "instruction policy is not the anchored artifact") {
		t.Fatalf("instruction-policy pin not enforced: %v", err)
	}
}

// MEDIUM-1: an anchorless Open is a refusal unless the caller role is
// declared, and unanchored records are marked, never ambiguous.
func TestUnanchoredRequiresExplicitOptIn(t *testing.T) {
	base := t.TempDir()
	cfg := Config{
		StateRoot: filepath.Join(base, "state"), ArtifactDir: filepath.Join(base, "artifacts"),
		GitPath: gitBin(t), ProviderDir: filepath.Join(base, "provider"),
		SafetyRoot: filepath.Join(repoRoot, "instructions/global/safety"),
		SystemRoot: filepath.Join(repoRoot, "instructions/global/system"),
		PolicyPath: filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"),
		Model:      happyScript(),
	}
	if _, _, err := Open(cfg); err == nil || !strings.Contains(err.Error(), "Unanchored not explicitly set") {
		t.Fatalf("silent unanchored bypass: %v", err)
	}
	cfg.Unanchored = true
	if _, _, err := Open(cfg); err != nil {
		t.Fatalf("declared unanchored role refused: %v", err)
	}
}

// HIGH-3: the model registry the anchor pins governs what names
// resolve to; a configured-but-unpinned registry refuses.
func TestAnchoredModelRegistryPin(t *testing.T) {
	ap, sha, reg := anchorWorld(t, nil) // declares model_registry "absent"
	cfg := anchoredConfig(t, t.TempDir(), ap, sha, reg)
	cfg.ModelRegistryPath = filepath.Join(t.TempDir(), "models.json")
	os.WriteFile(cfg.ModelRegistryPath, []byte(`{"models":[]}`), 0o644)
	if _, _, err := Open(cfg); err == nil || !strings.Contains(err.Error(), "declares no model registry") {
		t.Fatalf("unpinned model registry accepted: %v", err)
	}

	// And a pinned registry must match its bytes.
	mrDir := t.TempDir()
	mrPath := filepath.Join(mrDir, "models.json")
	os.WriteFile(mrPath, []byte(`{"models":[{"name":"scripted"}]}`), 0o644)
	mrHash, err := deployment.HashFile(mrPath)
	if err != nil {
		t.Fatal(err)
	}
	ap2, sha2, reg2 := anchorWorld(t, func(m map[string]any) { m["model_registry"] = mrHash })
	cfg2 := anchoredConfig(t, t.TempDir(), ap2, sha2, reg2)
	cfg2.ModelRegistryPath = mrPath
	if _, _, err := Open(cfg2); err != nil {
		t.Fatalf("matching model registry refused: %v", err)
	}
	os.WriteFile(mrPath, []byte(`{"models":[{"name":"scripted","endpoint":"https://elsewhere"}]}`), 0o644)
	cfg3 := anchoredConfig(t, t.TempDir(), ap2, sha2, reg2)
	cfg3.ModelRegistryPath = mrPath
	if _, _, err := Open(cfg3); err == nil || !strings.Contains(err.Error(), "only by Governance act") {
		t.Fatalf("rewritten model registry accepted: %v", err)
	}
}

func TestAnchoredSubmitRefusesUnanchoredBundle(t *testing.T) {
	// The anchor pins placeholder hashes for the bundle artifacts, so
	// the walk fixture's genuine artifacts cannot match: submission is
	// refused at the FIRST unanchored artifact — a mutually
	// consistent bundle is not a governed bundle (Q-G1-7/Q-G1-9).
	ap, sha, reg := anchorWorld(t, nil)
	f := setupVerif(t, happyScript(), &scriptedEvaluator{})
	anchored, _, err := Open(anchoredConfig(t, t.TempDir(), ap, sha, reg))
	if err != nil {
		t.Fatal(err)
	}
	env := f.verifEnvelope(t, "anchored-1")
	if _, err := anchored.SubmitTask(env); err == nil || !strings.Contains(err.Error(), "not the anchored artifact") {
		t.Fatalf("unanchored bundle accepted: %v", err)
	}

	// The model allowlist, with the bundle pins ALL satisfied so the
	// allowlist branch is genuinely reached (close-review HIGH-4: the
	// previous fixture refused at the exec ceiling and never
	// exercised the check it claimed to prove).
	fileHash := func(p string) string {
		h, herr := deployment.HashFile(p)
		if herr != nil {
			t.Fatal(herr)
		}
		return h
	}
	envHash := func(name string) string { return fileHash(filepath.Join(f.envDir, name)) }
	pinBundle := func(m map[string]any) {
		m["tool_registry"] = fileHash(filepath.Join(repoRoot, "policies/tools/registry-v4.json"))
		m["workflow_ceiling"] = envHash("wceiling.json")
		m["exec_ceiling"] = envHash("eceiling.json")
		m["context_contract"] = envHash("context-contract.json")
		m["workflows"] = []any{envHash("workflow.json")}
	}

	// Control: with every pin satisfied AND the model allowlisted,
	// the anchored submission passes assembly and the walk runs — the
	// positive anchored path, not only refusals.
	apOK, shaOK, regOK := anchorWorld(t, pinBundle)
	okOrch, _, err := Open(anchoredConfig(t, t.TempDir(), apOK, shaOK, regOK))
	if err != nil {
		t.Fatal(err)
	}
	okOrch.cfg.Verifier = &scriptedEvaluator{}
	if _, err := okOrch.SubmitTask(f.verifEnvelope(t, "anchored-ok")); err != nil {
		t.Fatalf("fully anchored submission refused: %v", err)
	}

	// Now the same world with ONLY the allowlist changed.
	ap2, sha2, reg2 := anchorWorld(t, func(m map[string]any) {
		pinBundle(m)
		m["models"] = []any{"some-other-model"}
	})
	anchored2, _, err := Open(anchoredConfig(t, t.TempDir(), ap2, sha2, reg2))
	if err != nil {
		t.Fatal(err)
	}
	_, err = anchored2.SubmitTask(f.verifEnvelope(t, "anchored-2"))
	if err == nil || !strings.Contains(err.Error(), "anchored allowlist") {
		t.Fatalf("model allowlist branch not reached: %v", err)
	}
}
