// Package orchestration implements Layer 7 of the Themis AI Harness:
// Orchestration — a deterministic executor of a governed workflow
// lattice. Governing design:
// openspec/changes/layer-07-orchestration/design.md (§2 locked
// decisions D-L7-1..12, §3 grill record Q-L7-1..12).
// Locked: L7 composes and paces, never grants; governance defines
// the lattice, the model walks within it, L7 enforces the walk; δ
// has no implicit control semantics; model content never reaches δ;
// for every transition there is exactly one governing edge and one
// typed causing event.
package orchestration

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

var (
	ErrConstitution = errors.New("orchestration constitution violation")
	ErrWorkflow     = errors.New("invalid workflow definition")
	ErrCeiling      = errors.New("invalid workflow ceiling")
	ErrEnvelope     = errors.New("invalid task envelope")
	ErrAssembly     = errors.New("task-assembly refused")
	ErrInvariant    = errors.New("orchestration invariant violation")
)

// The L7 constitution lives in code (Q-L6-5 pattern): it contains no
// variable content, and its canonical hash is recorded per task so
// reconstruction can answer "which orchestration regime governed
// this walk". Changing anything below is a Class-3 change.

// Control-verb vocabulary (Q-L7-6): closed, verb-per-signal, zero
// semantic arguments. v1 ships exactly one verb.
const (
	VerbDeclareDone = "declare_done"

	SignalPhaseCompletionRequested = "signal:phase-completion-requested"
)

// controlVerbs maps each constitution verb to its fixed 1:1 typed
// signal. The verb means nothing; the definition's edge means
// everything.
var controlVerbs = map[string]string{
	VerbDeclareDone: SignalPhaseCompletionRequested,
}

// Structural turn facts (Q-L7-5): properties of a model turn's
// shape, computed by the harness — never claims by the model.
// turn-tool-calls is loop-internal pacing and deliberately NOT
// declarable as a transition event.
const (
	EvTurnNoAction      = "turn-no-action"
	EvTurnProviderError = "turn-provider-error"
	EvTurnsExhausted    = "turns-exhausted"
	EvToolError         = "tool-error"
)

// transitionEvents is the complete system vocabulary a workflow
// definition may declare (declared ⊆ this set; Q-L7-3). Invariant
// and recovery classes are structurally absent — a workflow cannot
// mention what it must never handle (Q-L7-7). The approval
// vocabulary is RESERVED: named here so the reservation is a
// compilable fact, and refused at load until an approval channel
// exists (Q-L7-9).
var transitionEvents = map[string]bool{
	EvTurnNoAction:                 true,
	EvTurnProviderError:            true,
	EvTurnsExhausted:               true,
	EvToolError:                    true,
	SignalPhaseCompletionRequested: true,
}

// reservedApprovalPrefix marks the approval condition vocabulary:
// constitutionally reserved, unloadable in v1 (a gate without a
// channel is an eternal await).
const reservedApprovalPrefix = "approval:"

// Terminal edge targets — the closed termination vocabulary, mapped
// onto L5 seal reasons and the L6 lifecycle (D-L7-4/Q-L7-7):
//
//	@complete → seal(task-complete) → egress → COMPLETED
//	@fail     → seal(caller-abort)  → FAILED   (workflow-chosen)
//	@stay     → hold position (explicit governed stay, never implicit)
//
// Floors (constitution-owned, not edges): budget/deadline exhaustion
// → seal(env-deadline) → FAILED; invariant violation →
// seal(fatal-breach) → FAILED.
const (
	TargetStay     = "@stay"
	TargetComplete = "@complete"
	TargetFail     = "@fail"
)

// ConstitutionHash is the canonical hash of the L7 constitution,
// recorded in every task's governed hashes.
func ConstitutionHash() string {
	var parts []string
	for v, s := range controlVerbs {
		parts = append(parts, "verb:"+v+">"+s)
	}
	for e := range transitionEvents {
		parts = append(parts, "event:"+e)
	}
	parts = append(parts,
		"terminal:@complete>seal:task-complete>COMPLETED",
		"terminal:@fail>seal:caller-abort>FAILED",
		"floor:budget>seal:env-deadline>FAILED",
		"floor:invariant>seal:fatal-breach>FAILED",
		"reserved:"+reservedApprovalPrefix,
	)
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}
