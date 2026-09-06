package execution

import (
	"errors"
	"fmt"
	"sync"
	"time"
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

	// inflight counts executions currently running. Seal refuses
	// while any are live: SEALED means the workspace is stable, and
	// that must hold by mechanism, not by orchestrator discipline
	// (M1 security review MED-4).
	inflight int
	// remaining is the environment's wall-clock budget
	// (wall_deadline_s). Provisioning and active ops all draw from
	// the one budget; exhaustion seals the environment with
	// SealDeadline — the envelope-level enforcement the declaration
	// claims (M1 security review MED-3).
	remaining time.Duration

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

// Trace returns a deep copy of the durable environment record —
// consumers can never mutate the provider declaration or recorded
// argv through it (M1 security review LOW).
func (e *Env) Trace() Trace {
	e.mu.Lock()
	defer e.mu.Unlock()
	t := e.trace
	t.Provider.Limits = copyLimits(e.trace.Provider.Limits)
	t.Transitions = append([]Transition(nil), e.trace.Transitions...)
	t.Ops = make([]OpRecord, len(e.trace.Ops))
	for i, op := range e.trace.Ops {
		op.Argv = append([]string(nil), op.Argv...)
		t.Ops[i] = op
	}
	return t
}

func copyLimits(m map[string]Strength) map[string]Strength {
	out := make(map[string]Strength, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
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
// being copied is stable before inspection begins) — which is also
// why Seal refuses while an execution is in flight.
func (e *Env) Seal(reason SealReason) error {
	switch reason {
	case SealTaskComplete, SealDeadline, SealFatalBreach, SealCallerAbort:
	default:
		return fmt.Errorf("%w: unknown seal reason %q", ErrLifecycle, reason)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.inflight > 0 {
		return fmt.Errorf("%w: %d execution(s) in flight — a sealed workspace must be stable", ErrLifecycle, e.inflight)
	}
	return e.sealLocked(reason)
}

func (e *Env) sealLocked(reason SealReason) error {
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
