package orchestration

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
}

// D-P-1 (Plan ⊆ Contract): a contract that does not declare the
// phase-state slot refuses at composition, link-named; the runtime
// never injects the slot silently and never treats it as optional.
func TestPhaseStateSlotUndeclaredRefuses(t *testing.T) {
	m := &scriptedModel{steps: []model.ExecutionResponse{toolCall("declare_done", `{}`)}}
	f := setupVerif(t, m, &scriptedEvaluator{outcomes: []string{"PASS"}})
	writeJSON(t, f.envDir, "context-contract.json",
		`{"version":1,"workflow":"remediate-verify","slots":[
		  {"name":"task-payload","kind":"task-brief","requirement":"required","classes":["external-untrusted"]}],
		  "sensitivity_ceiling":"public"}`)
	_, err := f.o.SubmitTask(f.verifEnvelope(t, "verif-noslot"))
	if err == nil || !strings.Contains(err.Error(), `no slot "phase-state"`) {
		t.Fatalf("undeclared phase-state must refuse by name: %v", err)
	}
}
