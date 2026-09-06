package execution

import (
	"errors"
	"fmt"
	"sync"
)

var ErrLifecycle = errors.New("illegal lifecycle transition")

// State is the monotonic environment lifecycle (D-L5-9). The
// environment machine is not the tool-call machine: L4 governs
// individual capability invocations inside ACTIVE.
type State string

const (
	StateProvisioning      State = "PROVISIONING"
	StateActive            State = "ACTIVE"
	StateSealed            State = "SEALED"
	StateEgressing         State = "EGRESSING"
	StateAcknowledged      State = "ACKNOWLEDGED"
	StateTeardown          State = "TEARDOWN"
	StateDestroyed         State = "DESTROYED"
	StateTeardownAnomalous State = "TEARDOWN_ANOMALOUS"
)

// legalNext encodes the canonical machine. Invariants hold by
// reachability, not discipline: there is no ACTIVE→EGRESSING edge, no
// backward edge, no terminal state with a live workspace.
var legalNext = map[State]map[State]bool{
	StateProvisioning: {StateActive: true, StateTeardown: true},
	StateActive:       {StateSealed: true},
	StateSealed:       {StateEgressing: true, StateTeardown: true},
	StateEgressing:    {StateAcknowledged: true, StateTeardown: true},
	StateAcknowledged: {StateTeardown: true},
	StateTeardown:     {StateDestroyed: true, StateTeardownAnomalous: true},
	// DESTROYED and TEARDOWN_ANOMALOUS are terminal.
}

// SealReason is the typed cause of the one-way ACTIVE→SEALED edge.
// Only SealTaskComplete is a clean seal; only a clean seal may egress.
type SealReason string

const (
	SealTaskComplete SealReason = "task-complete"
	SealDeadline     SealReason = "env-deadline"
	SealFatalBreach  SealReason = "fatal-breach"
	SealCallerAbort  SealReason = "caller-abort"
)

// Env is one provisioned environment instance. Instances are
// single-use: there are no backward transitions; retry means a new
// environment with a new identity.
type Env struct {
	mu    sync.Mutex
	state State
	trace Trace

	provider *LocalProvider
	// env-owned dirs (workspace root lives in trace.Workspace)
	homeDir, tmpDir, hooksDir, baseDir string
}

// State returns the current lifecycle state.
func (e *Env) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// Trace returns a copy of the durable environment record.
func (e *Env) Trace() Trace {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.trace
	t.Transitions = append([]Transition(nil), e.trace.Transitions...)
	t.Ops = append([]OpRecord(nil), e.trace.Ops...)
	return t
}

// Workspace returns the provisioned workspace identity.
func (e *Env) Workspace() Workspace {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.trace.Workspace
}

// transition moves the machine along one legal edge, recording the
// typed event. Illegal edges are refused — never silently absorbed.
func (e *Env) transition(to State, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.transitionLocked(to, reason)
}

func (e *Env) transitionLocked(to State, reason string) error {
	if !legalNext[e.state][to] {
		return fmt.Errorf("%w: %s -> %s (%s)", ErrLifecycle, e.state, to, reason)
	}
	e.trace.Transitions = append(e.trace.Transitions, Transition{From: e.state, To: to, Reason: reason})
	e.state = to
	return nil
}

// Seal is the one-way end of execution: after it, the environment
// refuses all further executions, typed. Only a sealed environment
// may be read by the Artifact Egress Contract (Q-L5-12: the object
// being copied is stable before inspection begins).
func (e *Env) Seal(reason SealReason) error {
	switch reason {
	case SealTaskComplete, SealDeadline, SealFatalBreach, SealCallerAbort:
	default:
		return fmt.Errorf("%w: unknown seal reason %q", ErrLifecycle, reason)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.transitionLocked(StateSealed, string(reason)); err != nil {
		return err
	}
	e.trace.SealReason = reason
	return nil
}

// CleanlySealed reports whether the environment sealed clean —
// the only seal outcome from which egress is reachable.
func (e *Env) CleanlySealed() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state == StateSealed && e.trace.SealReason == SealTaskComplete
}

// beginEgress guards the SEALED→EGRESSING edge: clean seal only.
// A non-clean seal produces no artifact — timeout, breach, or abort
// routes to teardown (Q-L5-10: no partial artifacts).
func (e *Env) beginEgress() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != StateSealed || e.trace.SealReason != SealTaskComplete {
		return fmt.Errorf("%w: egress requires a cleanly sealed environment (state=%s seal=%s)", ErrLifecycle, e.state, e.trace.SealReason)
	}
	return e.transitionLocked(StateEgressing, "egress")
}
