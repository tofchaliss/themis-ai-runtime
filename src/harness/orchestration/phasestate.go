package orchestration

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

// Phase framing (D-P-1, D-P-2, 2026-09-27). Each phase entry composes a
// fresh conversation; the model is told where it is by a DERIVED L2
// fact computed from the recorded workflow state — never an L1
// instruction, never task-supplied data — and only when the skill's
// context contract declares the `phase-state` slot (Plan ⊆ Contract).

const (
	phaseStateSlot = "phase-state"
	phaseStateKind = "phase-state"
)

// PhaseState is the closed vocabulary of the fact (D-P-2): workflow
// position and current capability only. No counters, no verification
// results, no prior-phase outputs, no tool history.
type PhaseState struct {
	Workflow     string   `json:"workflow"`     // name@version of the governed workflow
	Phase        string   `json:"phase"`        // the phase being entered
	Completed    []string `json:"completed"`    // phases LEFT so far, in order of leaving (a countered re-entry repeats names)
	Capabilities []string `json:"capabilities"` // the NARROWED phase grant: Phase.capabilities ∩ task grant
}

// DerivePhaseState computes the fact for entering `phase` from the
// record as it stands at composition time. `capabilities` is the
// narrowed phase grant the loop will expose (what L4 authorizes).
// Returns the fact, its canonical bytes, and the provenance seq: the
// seq of the `workflow-transition` EVENT that entered the phase (its
// body carries the cause_seq of what fired it), or — for the initial
// phase — the seq of the RUNNING lifecycle event (D-P-2). The phase
// must be the record's CURRENT position: the latest non-`@` transition
// target, or the initial phase when none has fired. A record that
// disagrees with the loop's cursor is an invariant violation, never a
// default (D-P-1: the fact comes from the record, not loop memory).
func DerivePhaseState(wf *WorkflowDef, events []state.Event, phase string, capabilities []string) (PhaseState, []byte, int64, error) {
	ps := PhaseState{
		Workflow:     fmt.Sprintf("%s@%d", wf.Name, wf.Version),
		Phase:        phase,
		Completed:    []string{},
		Capabilities: append([]string{}, capabilities...),
	}
	var entrySeq int64 = -1
	var runningSeq int64 = -1
	current := wf.Initial
	for _, ev := range events {
		switch ev.Class {
		case state.EvLifecycle:
			var b struct {
				To string `json:"to"`
			}
			if json.Unmarshal(ev.Body, &b) == nil && b.To == string(state.StatusRunning) {
				runningSeq = ev.Seq
			}
		case state.EvWorkflowTransition:
			var b struct {
				From string `json:"from"`
				To   string `json:"to"`
			}
			if json.Unmarshal(ev.Body, &b) != nil || strings.HasPrefix(b.To, "@") {
				continue
			}
			ps.Completed = append(ps.Completed, b.From)
			current = b.To
			if b.To == phase {
				entrySeq = ev.Seq
			}
		}
	}
	if current != phase {
		return ps, nil, 0, fmt.Errorf("%w: the record's current phase is %q, not %q", ErrInvariant, current, phase)
	}
	if entrySeq < 0 {
		if phase != wf.Initial || runningSeq < 0 {
			return ps, nil, 0, fmt.Errorf("%w: phase %q has no entering event in the record", ErrInvariant, phase)
		}
		entrySeq = runningSeq
	}
	raw, err := json.Marshal(ps)
	if err != nil {
		return ps, nil, 0, err
	}
	return ps, raw, entrySeq, nil
}
