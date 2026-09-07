// Package state implements Layer 6 of the Themis AI Harness: Durable
// State — the record plane. Governing design:
// openspec/changes/layer-06-durable-state/design.md (§2 locked
// decisions D-L6-1..11, §3 grill record Q-L6-1..11).
// Locked: the event stream is the record and the manifest is a
// projection; durable means fsync-committed at local-substrate
// strength, acknowledged only afterward; a durable record may
// reference only already-durable material (orphans are waste,
// dangles are lies); recovery verifies and appends, never rewrites;
// L6 owns persistence, identity, integrity, and classification
// preservation — never meaning or control consequences.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

var (
	ErrConstitution = errors.New("durability constitution violation")
	ErrPersist      = errors.New("durable-commit failed")
	ErrIdentity     = errors.New("durable identity violation")
	ErrLifecycle    = errors.New("illegal task lifecycle transition")
	ErrStream       = errors.New("event-stream failure")
	ErrCorrupt      = errors.New("record corruption")
)

// The durability constitution lives in code because it contains no
// variable content (Q-L6-5): a governed artifact whose every field is
// constant would be configuration theater. Changing anything below is
// a Class-3 change. Its canonical hash is recorded in every task
// manifest so reconstruction can always answer "which durability
// regime governed this task" — constant today, meaningful the day the
// first genuine governance knob (retention) arrives.

// Declarable object classes — the complete v1 vocabulary (Q-L6-1/5).
// Trace events and task manifests are floors expressed through their
// own primitives, not declarable classes. Secrets are not a class.
const (
	ObjEvidencePayload = "evidence-payload"
	ObjEgressArtifact  = "egress-artifact"
)

var objectClasses = map[string]bool{
	ObjEvidencePayload: true,
	ObjEgressArtifact:  true,
}

// Event classes — the closed sink vocabulary. L6 owns lifecycle,
// recovery, verdict, and contamination-suspected; the remaining
// classes are owned by their emitting layers (L6 validates the
// envelope, never the content — Q-L6-4).
const (
	EvLifecycle     = "lifecycle"
	EvRecovery      = "recovery"
	EvVerdict       = "verdict"
	EvContamination = "contamination-suspected"
	EvL1Conflict    = "l1-conflict"
	EvL2Delivery    = "l2-delivery"
	EvL3Selection   = "l3-selection"
	EvL4Audit       = "l4-audit"
	EvL5Transition  = "l5-transition"
	EvL5Op          = "l5-op"
)

var eventClasses = map[string]bool{
	EvLifecycle: true, EvRecovery: true, EvVerdict: true, EvContamination: true,
	EvL1Conflict: true, EvL2Delivery: true, EvL3Selection: true,
	EvL4Audit: true, EvL5Transition: true, EvL5Op: true,
}

// recoveryOnlyEvents may be appended only by L6's own recovery and
// verification passes, never by callers (Q-L6-4/6).
var recoveryOnlyEvents = map[string]bool{EvRecovery: true, EvVerdict: true}

// Task lifecycle — the closed monotonic machine (Q-L6-6). Terminals
// are immutable. FAILED_PARTIAL is recovery-owned: it is not in the
// caller-requestable set. Record verdicts (VERIFIED/TORN/CORRUPT) are
// deliberately NOT lifecycle states: corruption is a verdict about
// the record, never a state of the task.
type TaskStatus string

const (
	StatusCreated       TaskStatus = "CREATED"
	StatusRunning       TaskStatus = "RUNNING"
	StatusCompleted     TaskStatus = "COMPLETED"
	StatusFailed        TaskStatus = "FAILED"
	StatusFailedPartial TaskStatus = "FAILED_PARTIAL"
)

var legalNext = map[TaskStatus]map[TaskStatus]bool{
	StatusCreated: {StatusRunning: true, StatusFailed: true},
	StatusRunning: {StatusCompleted: true, StatusFailed: true, StatusFailedPartial: true},
	// COMPLETED, FAILED, FAILED_PARTIAL are terminal.
}

// callerRequestable excludes the recovery-owned structural fact.
var callerRequestable = map[TaskStatus]bool{
	StatusRunning: true, StatusCompleted: true, StatusFailed: true,
}

// Record verdicts — orthogonal to lifecycle (Q-L6-6).
type Verdict string

const (
	VerdictVerified Verdict = "VERIFIED"
	VerdictTorn     Verdict = "TORN"
	VerdictCorrupt  Verdict = "CORRUPT"
)

// Object identity: algorithm-prefixed content hash in a closed v1
// namespace (Q-L6-3). An unknown prefix refuses.
const hashAlgo = "sha256"

func objectID(b []byte) string {
	sum := sha256.Sum256(b)
	return hashAlgo + ":" + hex.EncodeToString(sum[:])
}

// ConstitutionHash is the SHA-256 of the constitution's canonical
// form, recorded in every task manifest (Q-L6-5).
func ConstitutionHash() string {
	var parts []string
	for c := range objectClasses {
		parts = append(parts, "object:"+c)
	}
	for c := range eventClasses {
		parts = append(parts, "event:"+c)
	}
	for from, tos := range legalNext {
		for to := range tos {
			parts = append(parts, "edge:"+string(from)+">"+string(to))
		}
	}
	parts = append(parts, "algo:"+hashAlgo, "retention:retain-all", "commit:fsync-local")
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}
