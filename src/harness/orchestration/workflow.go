package orchestration

// The workflow definition: the sole executable control structure
// (Q-L7-1/2/3). Loaded fail-closed with STATIC verification: totality
// (every phase maps every declared event exactly once), bounded
// counters with explicit forward exhaustion edges, no counter-free
// non-forward edges, a computed finite worst-case walk, and ⊆ the
// WorkflowCeiling. Invalid governed artifacts fail before execution
// rather than becoming runtime policy.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Edge is one explicit governed transition. On a countered edge,
// ExhaustedTo names the mandatory exhaustion target, which must be
// strictly forward or terminal (static boundedness, Q-L7-4).
type Edge struct {
	On          string `json:"on"`
	To          string `json:"to"`
	Counter     int64  `json:"counter,omitempty"`
	ExhaustedTo string `json:"exhausted_to,omitempty"`
}

// Phase is one workflow position: its granted capability subset, its
// model-turn budget, and its total edge map.
type Phase struct {
	Name          string   `json:"name"`
	Capabilities  []string `json:"capabilities"`
	MaxModelTurns int64    `json:"max_model_turns"`
	Edges         []Edge   `json:"edges"`
}

// WorkflowDef is the governed, per-task-immutable lattice.
type WorkflowDef struct {
	Version        int      `json:"version"`
	Name           string   `json:"name"`
	DeclaredEvents []string `json:"declared_events"`
	Initial        string   `json:"initial"`
	Phases         []Phase  `json:"phases"`

	Hash         string `json:"-"`
	WorstCaseLen int64  `json:"-"` // statically computed walk bound
}

// WorkflowCeiling bounds what any definition may declare
// (definition ⊆ ceiling ⊆ registry — the cross-layer principle).
type WorkflowCeiling struct {
	Version          int      `json:"version"`
	AllowedTools     []string `json:"allowed_tools"`
	MaxTotalCalls    int      `json:"max_total_calls"`
	MaxWalkLength    int64    `json:"max_walk_length"`
	MaxTurnsPerPhase int64    `json:"max_turns_per_phase"`

	Hash string `json:"-"`
}

func LoadWorkflowCeiling(path string) (*WorkflowCeiling, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCeiling, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var c WorkflowCeiling
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCeiling, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrCeiling, path)
	}
	if c.Version < 1 || len(c.AllowedTools) == 0 || c.MaxTotalCalls <= 0 || c.MaxWalkLength <= 0 || c.MaxTurnsPerPhase <= 0 {
		return nil, fmt.Errorf("%w: %s: version, allowed tools, and positive bounds required", ErrCeiling, path)
	}
	sum := sha256.Sum256(raw)
	c.Hash = hex.EncodeToString(sum[:])
	return &c, nil
}

// LoadWorkflow loads and STATICALLY verifies a definition against
// the constitution and the ceiling. Every rule below is a Q-L7-3/4
// lock; none is advisory.
func LoadWorkflow(path string, ceiling *WorkflowCeiling) (*WorkflowDef, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrWorkflow, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var w WorkflowDef
	if err := dec.Decode(&w); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrWorkflow, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrWorkflow, path)
	}
	if w.Version < 1 || w.Name == "" || len(w.Phases) == 0 || w.Initial == "" {
		return nil, fmt.Errorf("%w: %s: version, name, initial, and phases required", ErrWorkflow, path)
	}

	// Declared events: ⊆ the constitution's transition vocabulary;
	// approval vocabulary reserved-unloadable; turns-exhausted
	// mandatory (every phase has a turn budget, so its exhaustion
	// must have a governed meaning).
	declared := map[string]bool{}
	for _, e := range w.DeclaredEvents {
		if strings.HasPrefix(e, reservedApprovalPrefix) {
			return nil, fmt.Errorf("%w: %q: approval conditions are reserved — no approval channel exists (Q-L7-9)", ErrWorkflow, e)
		}
		if !transitionEvents[e] {
			return nil, fmt.Errorf("%w: undeclarable event %q — outside the transition vocabulary", ErrWorkflow, e)
		}
		if declared[e] {
			return nil, fmt.Errorf("%w: duplicate declared event %q", ErrWorkflow, e)
		}
		declared[e] = true
	}
	// Runtime-producible events are load-mandatory (security review
	// MED-3): any model turn can produce no-action or a provider
	// error, and every phase has a turn budget — a definition that
	// cannot map them would route ordinary model behavior onto the
	// fatal-breach invariant path, which is never model-triggerable.
	for _, mandatory := range []string{EvTurnsExhausted, EvTurnNoAction, EvTurnProviderError} {
		if !declared[mandatory] {
			return nil, fmt.Errorf("%w: %s must be declared — it is runtime-producible in every phase", ErrWorkflow, mandatory)
		}
	}

	phaseIdx := map[string]int{}
	for i, p := range w.Phases {
		if p.Name == "" || strings.HasPrefix(p.Name, "@") {
			return nil, fmt.Errorf("%w: bad phase name %q", ErrWorkflow, p.Name)
		}
		if _, dup := phaseIdx[p.Name]; dup {
			return nil, fmt.Errorf("%w: duplicate phase %q", ErrWorkflow, p.Name)
		}
		phaseIdx[p.Name] = i
	}
	if _, ok := phaseIdx[w.Initial]; !ok {
		return nil, fmt.Errorf("%w: initial phase %q not defined", ErrWorkflow, w.Initial)
	}

	terminalOK := map[string]bool{TargetStay: true, TargetComplete: true, TargetFail: true}
	isForward := func(from int, to string) bool {
		if to == TargetComplete || to == TargetFail {
			return true
		}
		j, ok := phaseIdx[to]
		return ok && j > from
	}

	var backwardCounterSum int64
	for i, p := range w.Phases {
		if p.MaxModelTurns <= 0 || p.MaxModelTurns > ceiling.MaxTurnsPerPhase {
			return nil, fmt.Errorf("%w: phase %q: max_model_turns must be positive and ≤ ceiling %d", ErrWorkflow, p.Name, ceiling.MaxTurnsPerPhase)
		}
		// Phase capabilities ⊆ ceiling allowed tools.
		allowed := map[string]bool{}
		for _, t := range ceiling.AllowedTools {
			allowed[t] = true
		}
		for _, c := range p.Capabilities {
			if !allowed[c] {
				return nil, fmt.Errorf("%w: phase %q capability %q exceeds the workflow ceiling", ErrWorkflow, p.Name, c)
			}
			// A phase exposing a control verb must declare its signal,
			// else an authorized call would reach δ undeclared — the
			// invariant path fed by legal model behavior (MED-3).
			if sig, isControl := controlVerbs[c]; isControl && !declared[sig] {
				return nil, fmt.Errorf("%w: phase %q exposes control verb %q but does not declare its signal %q", ErrWorkflow, p.Name, c, sig)
			}
		}
		// TOTALITY: exactly one edge per declared event; no undeclared
		// edges; no implicit anything (Q-L7-3).
		seen := map[string]bool{}
		for _, e := range p.Edges {
			if !declared[e.On] {
				return nil, fmt.Errorf("%w: phase %q edge on undeclared event %q", ErrWorkflow, p.Name, e.On)
			}
			if seen[e.On] {
				return nil, fmt.Errorf("%w: phase %q maps event %q twice — edges must be unique", ErrWorkflow, p.Name, e.On)
			}
			seen[e.On] = true
			if _, ok := phaseIdx[e.To]; !ok && !terminalOK[e.To] {
				return nil, fmt.Errorf("%w: phase %q edge targets unknown %q", ErrWorkflow, p.Name, e.To)
			}
			// turns-exhausted cannot stay: the budget is spent, so a
			// stay target would be statically-verified-yet-unexecutable
			// (security review MED-4).
			if e.On == EvTurnsExhausted && (e.To == TargetStay || e.ExhaustedTo == TargetStay) {
				return nil, fmt.Errorf("%w: phase %q: %s cannot target @stay — the turn budget is spent", ErrWorkflow, p.Name, EvTurnsExhausted)
			}
			// Static boundedness (Q-L7-4): every non-forward edge must
			// carry a counter, and every counter its explicit forward
			// exhaustion edge.
			forward := isForward(i, e.To)
			if !forward && e.Counter <= 0 {
				return nil, fmt.Errorf("%w: phase %q edge %q->%q is non-forward without a counter — counter-free cycles are unloadable", ErrWorkflow, p.Name, e.On, e.To)
			}
			if e.Counter > 0 {
				if e.ExhaustedTo == "" {
					return nil, fmt.Errorf("%w: phase %q countered edge %q lacks its exhaustion edge", ErrWorkflow, p.Name, e.On)
				}
				if !isForward(i, e.ExhaustedTo) {
					return nil, fmt.Errorf("%w: phase %q exhaustion target %q must be strictly forward or terminal", ErrWorkflow, p.Name, e.ExhaustedTo)
				}
				backwardCounterSum += e.Counter
			}
			if e.Counter == 0 && e.ExhaustedTo != "" {
				return nil, fmt.Errorf("%w: phase %q edge %q declares exhaustion without a counter", ErrWorkflow, p.Name, e.On)
			}
		}
		for ev := range declared {
			if !seen[ev] {
				return nil, fmt.Errorf("%w: phase %q has no edge for declared event %q — totality is static", ErrWorkflow, p.Name, ev)
			}
		}
	}

	// Worst-case walk: phase visits ≤ #phases + Σ non-forward
	// counters; turns per visit ≤ each phase's budget. Conservative,
	// finite, computed at load and bounded by the ceiling.
	var maxTurns int64
	for _, p := range w.Phases {
		if p.MaxModelTurns > maxTurns {
			maxTurns = p.MaxModelTurns
		}
	}
	visits := int64(len(w.Phases)) + backwardCounterSum
	w.WorstCaseLen = visits * (maxTurns + 1)
	if w.WorstCaseLen > ceiling.MaxWalkLength {
		return nil, fmt.Errorf("%w: worst-case walk %d exceeds ceiling %d", ErrWorkflow, w.WorstCaseLen, ceiling.MaxWalkLength)
	}

	sum := sha256.Sum256(raw)
	w.Hash = hex.EncodeToString(sum[:])
	return &w, nil
}

func (w *WorkflowDef) phase(name string) *Phase {
	for i := range w.Phases {
		if w.Phases[i].Name == name {
			return &w.Phases[i]
		}
	}
	return nil
}
