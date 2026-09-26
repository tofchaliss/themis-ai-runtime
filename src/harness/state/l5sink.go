package state

import (
	"encoding/json"
	"fmt"
)

// writerL5 is the canonical L5 writer identity. It is named in exactly
// one place — here — and reaches the record only through L5Sink
// (D-W-6 Wall B). No other package passes it to AppendEvent, and
// AppendEvent refuses the L5 classes outright (handleOnlyEvents).
const writerL5 = "l5"

// handleOnlyEvents are appendable only through a layer-scoped handle,
// never through the caller-facing AppendEvent, so the pair
// (class, writer) can only be produced by the emission path that owns
// it (D-W-6: the handle fixes the writer and closes the classes).
var handleOnlyEvents = map[string]bool{
	EvL5Transition: true, EvL5Op: true,
}

// L5Sink is the L5-scoped emission handle (D-W-6 Wall B): it exposes
// exactly the two L5 witness forms, carries no writer or class
// parameter, and cannot emit any other class. L7 constructs it from the
// task record and hands it to the environment; the environment never
// sees the TaskRecord.
type L5Sink struct {
	t *TaskRecord
}

// L5Sink returns the handle for this task's stream. Cheap; the handle
// holds no state of its own.
func (t *TaskRecord) L5Sink() *L5Sink { return &L5Sink{t: t} }

func (s *L5Sink) emit(class string, body []byte) error {
	if s == nil || s.t == nil {
		return fmt.Errorf("%w: L5 witness handle is not bound to a task", ErrStream)
	}
	s.t.mu.Lock()
	defer s.t.mu.Unlock()
	if terminal(s.t.man.Status) {
		return fmt.Errorf("%w: task %s is terminal", ErrLifecycle, s.t.man.TaskID)
	}
	_, err := s.t.appendLocked(class, writerL5, body, nil)
	return err
}

// Transition witnesses one committed edge of the L5 environment state
// machine (D-W-2): body {from, to, reason}, exactly as the environment
// trace holds it.
func (s *L5Sink) Transition(from, to, reason string) error {
	if from == "" || to == "" {
		return fmt.Errorf("%w: l5-transition requires from and to", ErrStream)
	}
	body, _ := json.Marshal(map[string]string{"from": from, "to": to, "reason": reason})
	return s.emit(EvL5Transition, body)
}

// Op witnesses one governed subprocess operation (D-W-3 form 1): body
// {phase, argv, exit, outcome}. Observed resource usage is deliberately
// not part of the witness.
func (s *L5Sink) Op(phase string, argv []string, exit int, outcome string) error {
	if phase == "" || outcome == "" {
		return fmt.Errorf("%w: l5-op requires phase and outcome", ErrStream)
	}
	if argv == nil {
		argv = []string{}
	}
	body, _ := json.Marshal(map[string]any{"phase": phase, "argv": argv, "exit": exit, "outcome": outcome})
	return s.emit(EvL5Op, body)
}

// Egress witnesses the egress operation (D-W-3 form 2): body
// {op: "egress", outcome, artifact_address, observed_total_bytes,
// observed_file_count}. Only outcome "acknowledged" carries an address;
// a refused or failed egress is witnessed with none and can never
// satisfy production (D-W-3 tightening).
func (s *L5Sink) Egress(outcome, address string, totalBytes, fileCount int64) error {
	if outcome == "" {
		return fmt.Errorf("%w: egress witness requires an outcome", ErrStream)
	}
	if (outcome == "acknowledged") != (address != "") {
		return fmt.Errorf("%w: egress witness: an address is present iff the outcome is acknowledged", ErrStream)
	}
	body, _ := json.Marshal(map[string]any{
		"op": "egress", "outcome": outcome, "artifact_address": address,
		"observed_total_bytes": totalBytes, "observed_file_count": fileCount,
	})
	return s.emit(EvL5Op, body)
}
