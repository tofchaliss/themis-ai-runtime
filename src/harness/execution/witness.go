package execution

import "fmt"

// Witness is the L5-scoped emission handle as L5 consumes it (D-W-6
// Wall B): three fixed forms, no writer, no class, no generic append.
// L6 supplies the concrete handle (state.L5Sink) bound to the task
// record; this package declares only what it needs, so it imports
// nothing of the record plane and can reach no other event class.
//
// The handle constrains HOW L5 witnesses its own facts. It decides
// nothing about whether an operation is authorized — authorization is
// L4's and the ceiling's, exactly as before (owner, W-M2).
type Witness interface {
	// Transition witnesses one committed edge of the environment
	// state machine {from, to, reason} (D-W-2).
	Transition(from, to, reason string) error
	// Op witnesses one governed subprocess operation
	// {phase, argv, exit, outcome} (D-W-3 form 1).
	Op(phase string, argv []string, exit int, outcome string) error
	// Egress witnesses the egress operation; only "acknowledged"
	// carries an address (D-W-3 form 2).
	Egress(outcome, address string, totalBytes, fileCount int64) error
}

// ErrUnwitnessed: governed execution was attempted in an environment
// that has no witness handle. Fail closed: an unwitnessed active
// operation would leave a gap in the full-machine record D-W-2 requires.
var ErrUnwitnessed = fmt.Errorf("%w: environment is not witnessed", ErrLifecycle)

// pendingEmission is one witness call recorded before a handle was
// attached — the provisioning edges and ops happen before L7 can
// create the task record (Provision precedes CreateTask by design of
// the assembly: the effective grant depends on the workspace). They
// are replayed in exact order at Attach, before any further act, so
// the record still shows the whole machine in sequence.
type pendingEmission func(Witness) error

// Attach binds the witness handle once. Every emission buffered so far
// is replayed in order; a replay failure refuses the attachment (and
// the environment stays unwitnessed, so no active work can proceed).
func (e *Env) Attach(w Witness) error {
	if w == nil {
		return fmt.Errorf("%w: nil witness", ErrLifecycle)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.witness != nil {
		return fmt.Errorf("%w: environment already witnessed", ErrLifecycle)
	}
	for i, p := range e.pending {
		if err := p(w); err != nil {
			return fmt.Errorf("witness replay of buffered emission %d: %w", i, err)
		}
	}
	e.pending = nil
	e.witness = w
	return nil
}

// Witnessed reports whether a handle is attached.
func (e *Env) Witnessed() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.witness != nil
}

// emitLocked delivers one witness call live, or buffers it when no
// handle is attached yet. Caller holds e.mu.
func (e *Env) emitLocked(p pendingEmission) error {
	if e.witness == nil {
		e.pending = append(e.pending, p)
		return nil
	}
	return p(e.witness)
}

// witnessTransitionLocked witnesses the edge the trace just recorded
// (the last Transition). Ordering is chosen per edge by the caller
// (D-W-2): after-effect edges call this after the effect; before-effect
// edges call it before the operation proceeds.
func (e *Env) witnessTransitionLocked() error {
	n := len(e.trace.Transitions)
	if n == 0 {
		return nil
	}
	tr := e.trace.Transitions[n-1]
	from, to, reason := string(tr.From), string(tr.To), tr.Reason
	return e.emitLocked(func(w Witness) error { return w.Transition(from, to, reason) })
}

// witnessOpLocked witnesses one recorded subprocess op (after effect).
func (e *Env) witnessOpLocked(op OpRecord) error {
	argv := append([]string(nil), op.Argv...)
	phase, exit, outcome := op.Phase, op.Exit, op.Outcome
	return e.emitLocked(func(w Witness) error { return w.Op(phase, argv, exit, outcome) })
}

// witnessEgressLocked witnesses the egress operation (after the store
// acknowledged, or on a typed refusal/failure).
func (e *Env) witnessEgressLocked(outcome, address string, totalBytes, fileCount int64) error {
	return e.emitLocked(func(w Witness) error { return w.Egress(outcome, address, totalBytes, fileCount) })
}
