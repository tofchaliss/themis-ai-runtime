package orchestration

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/instructions"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/runtime/model"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

func evJSON(seq int64, class string, body string) state.Event {
	return state.Event{Seq: seq, Class: class, Body: json.RawMessage(body)}
}

// D-P-2: the fact is the closed vocabulary, derived from the record —
// the entering transition's seq is its provenance, the initial phase
// cites RUNNING, `completed` is the ordered chain, and a phase with no
// entering event is an invariant, never a default.
func TestDerivePhaseStateFromRecord(t *testing.T) {
	wf := &WorkflowDef{Version: 3, Name: "remediate-dependency", Initial: "ANALYZE"}
	events := []state.Event{
		evJSON(0, state.EvLifecycle, `{"to":"CREATED"}`),
		evJSON(6, state.EvLifecycle, `{"reason":"assembled","to":"RUNNING"}`),
		evJSON(13, state.EvWorkflowTransition, `{"from":"ANALYZE","to":"REMEDIATE","edge_id":"ANALYZE/signal:phase-completion-requested","cause_seq":12}`),
	}
	ps, raw, seq, err := DerivePhaseState(wf, events[:2], "ANALYZE", []string{"get_finding", "read_file"})
	if err != nil {
		t.Fatal(err)
	}
	if seq != 6 || ps.Workflow != "remediate-dependency@3" || ps.Phase != "ANALYZE" || len(ps.Completed) != 0 || len(ps.Capabilities) != 2 {
		t.Fatalf("initial: %+v seq %d", ps, seq)
	}
	if string(raw) != `{"workflow":"remediate-dependency@3","phase":"ANALYZE","completed":[],"capabilities":["get_finding","read_file"]}` {
		t.Fatalf("canonical bytes: %s", raw)
	}
	ps, _, seq, err = DerivePhaseState(wf, events, "REMEDIATE", []string{"write_file"})
	if err != nil || seq != 13 || len(ps.Completed) != 1 || ps.Completed[0] != "ANALYZE" {
		t.Fatalf("second phase: %+v seq %d err %v", ps, seq, err)
	}
	// A stay edge is not a completion; an unknown phase has no entry.
	stay := append(events, evJSON(20, state.EvWorkflowTransition, `{"from":"REMEDIATE","to":"@stay"}`))
	ps, _, _, _ = DerivePhaseState(wf, stay, "REMEDIATE", nil)
	if len(ps.Completed) != 1 {
		t.Fatalf("@stay must not count as completed: %+v", ps.Completed)
	}
	if _, _, _, err := DerivePhaseState(wf, events[:1], "ANALYZE", nil); !errors.Is(err, ErrInvariant) {
		t.Fatalf("no RUNNING, no entry: %v", err)
	}
	if _, _, _, err := DerivePhaseState(wf, events, "VERIFY", nil); !errors.Is(err, ErrInvariant) {
		t.Fatalf("never entered: %v", err)
	}
	// Currency: the record's latest position wins over the loop's cursor
	// — entering ANALYZE again while the record says REMEDIATE is an
	// invariant, and the initial phase is current only before any move.
	if _, _, _, err := DerivePhaseState(wf, events, "ANALYZE", nil); !errors.Is(err, ErrInvariant) {
		t.Fatalf("stale cursor: %v", err)
	}
	back := append(events, evJSON(21, state.EvWorkflowTransition, `{"from":"REMEDIATE","to":"ANALYZE","edge_id":"REMEDIATE/retry"}`))
	ps, _, seq, err = DerivePhaseState(wf, back, "ANALYZE", nil)
	if err != nil || seq != 21 || len(ps.Completed) != 2 || ps.Completed[1] != "REMEDIATE" {
		t.Fatalf("countered re-entry: %+v seq %d err %v", ps, seq, err)
	}
}

// The L1 phase rule reaches the effective instruction set from the
// real harness-system root: the sentence the model is governed by is
// in the render, under the harness-owned section (D-P-1, D-P-7).
func TestPhaseRuleIsInTheRenderedEIS(t *testing.T) {
	policy, err := instructions.LoadPolicy(filepath.Join(repoRoot, "policies/security/instruction-directive-patterns.json"))
	if err != nil {
		t.Fatal(err)
	}
	eis, err := instructions.Resolve(instructions.Config{Policy: policy},
		instructions.Source{Kind: instructions.ScopeHarnessSafety, Root: filepath.Join(repoRoot, "instructions/global/safety")},
		instructions.Source{Kind: instructions.ScopeHarnessSystem, Root: filepath.Join(repoRoot, "instructions/global/system")})
	if err != nil {
		t.Fatal(err)
	}
	text, _, err := eis.Render(policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Each phase begins a fresh conversation", "A phase completes only when you call the `declare_done` capability"} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered EIS lacks the phase rule sentence %q", want)
		}
	}
	if strings.Index(text, "## Procedure") > strings.Index(text, "Each phase begins a fresh conversation") {
		t.Fatal("the phase rule must render under the harness-owned section")
	}
}

// D-P-1 (Plan ⊆ Contract): a contract that does not declare the
// phase-state slot refuses at composition, link-named; the runtime
// never injects the slot silently and never treats it as optional.
func TestPhaseStateSlotUndeclaredRefuses(t *testing.T) {
	// "withheld" is refused earlier, by L2 contract validation (a required
	// slot cannot be withheld), so it is asserted by that message below.
	cases := map[string]struct{ slots, want string }{
		"undeclared": {`{"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}`, `no slot "phase-state"`},
		"optional": {`{"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]},
		  {"name":"phase-state","kind":"phase-state","requirement":"optional","classes":["derived"]}`, `must be required with classes [derived]`},
		"wrong class": {`{"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]},
		  {"name":"phase-state","kind":"phase-state","requirement":"required","classes":["external-untrusted"]}`, `must be required with classes [derived]`},
		"withheld": {`{"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]},
		  {"name":"phase-state","kind":"phase-state","requirement":"required","classes":["derived"],"withhold":true}`, `invalid context contract`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := &scriptedModel{steps: []model.ExecutionResponse{toolCall("declare_done", `{}`)}}
			f := setupVerif(t, m, &scriptedEvaluator{outcomes: []string{"PASS"}})
			writeJSON(t, f.envDir, "context-contract.json", `{"version":1,"workflow":"remediate-verify","slots":[`+c.slots+`],"sensitivity_ceiling":"public"}`)
			task := "verif-" + strings.ReplaceAll(name, " ", "-")
			_, err := f.o.SubmitTask(f.verifEnvelope(t, task))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%s: must refuse naming the cause: %v", name, err)
			}
			if name != "withheld" && !errors.Is(err, ErrInvariant) {
				t.Fatalf("%s: a composition the contract cannot satisfy is an L7 invariant: %v", name, err)
			}
			// The record: no model turn — the composition never reached the
			// model; where a task record exists it is sealed FAILED.
			if st, err := f.o.root.ReadStatus(task); err == nil && st.Status != state.StatusFailed {
				t.Fatalf("%s: status %v", name, st.Status)
			}
			evs, _ := f.o.root.ReadEvents(task)
			for _, ev := range evs {
				if ev.Class == state.EvModelTurn {
					t.Fatalf("%s: a refused composition must not reach the model", name)
				}
			}
		})
	}
}
