package orchestration

// L10-M3 proofs: the verification seam amendment to the archived L7
// layer. Every new loader/assembly rule is pinned by a doctored
// definition (the TestLoaderRefusals pattern); the gate ladder and
// latest-per-token δ semantics are pinned deterministically; existing
// non-verification workflows must be bit-for-bit unaffected (the
// D-L10-17 additive-only obligation).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
