// Package instructions implements Layer 1 of the Themis AI Harness:
// deterministic loading, validation, and (in later milestones) conflict
// resolution, hashing, and rendering of the Effective Instruction Set.
//
// Governing design: openspec/changes/layer-01-instructions/design.md
// (the 2026-09-05 grill record, §6, supersedes earlier decisions where
// stated). Layer 1 is not a security enforcement dependency: its failure
// may produce incorrect or incomplete instruction resolution but must
// not provide a path around deterministic authorization, security
// controls, verification, or Themis governance.
package instructions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Scope is an instruction's rank in the fixed seven-level precedence
// order. Smaller values rank higher. A lower scope may specialize a
// higher one, never contradict it. The order is fixed forever: enum
// values exist even for source kinds v1 refuses to load (repository,
// directory, skill), so precedence never changes when they activate.
type Scope int

const (
	ScopeHarnessSafety Scope = iota
	ScopeHarnessSystem
	ScopeThemisDomain
	ScopeRepository
	ScopeDirectory
	ScopeSkill
	ScopeTask
)

var scopeNames = map[Scope]string{
	ScopeHarnessSafety: "harness-safety",
	ScopeHarnessSystem: "harness-system",
	ScopeThemisDomain:  "themis-domain",
	ScopeRepository:    "repository",
	ScopeDirectory:     "directory",
	ScopeSkill:         "skill",
	ScopeTask:          "task",
}

func (s Scope) String() string {
	if n, ok := scopeNames[s]; ok {
		return n
	}
	return fmt.Sprintf("scope(%d)", int(s))
}

// ParseScope returns the Scope for its canonical string form.
func ParseScope(s string) (Scope, error) {
	for sc, n := range scopeNames {
		if n == s {
			return sc, nil
		}
	}
	return 0, fmt.Errorf("%w: unknown scope %q", ErrMalformed, s)
}

// Category groups instructions under fixed render headings.
type Category string

const (
	CategorySafety      Category = "safety"
	CategoryHarness     Category = "harness"
	CategoryThemis      Category = "themis-domain"
	CategoryEngineering Category = "engineering"
	CategorySkill       Category = "skill"
	CategoryTask        Category = "task"
)

var validCategories = map[Category]bool{
	CategorySafety:      true,
	CategoryHarness:     true,
	CategoryThemis:      true,
	CategoryEngineering: true,
	CategorySkill:       true,
	CategoryTask:        true,
}

// ConflictClass distinguishes the three recorded resolution events
// (design §6, Q-L1-1): an identity claim rejected on namespace
// ownership (shadowed), a directive-pattern rejection
// (rejected-pattern), and an instruction naming a non-overridable
// capability boundary (rejected-prohibited).
type ConflictClass string

const (
	ConflictShadowed           ConflictClass = "shadowed"
	ConflictRejectedPattern    ConflictClass = "rejected-pattern"
	ConflictRejectedProhibited ConflictClass = "rejected-prohibited"
)

// Conflict is a recorded resolution event — trace data, not a log
// line. It binds identity, scope, source, and content revision so
// every identity contest is reconstructable from the trace.
type Conflict struct {
	Class    ConflictClass
	ID       string
	Scope    Scope
	Source   string
	BodyHash string
	Detail   string
}

// ResolutionStatus is the explicit outcome a run result must expose:
// recorded conflicts never look like an indistinguishable ordinary
// success (design §6, Q-L1-6).
type ResolutionStatus string

const (
	StatusCompleted              ResolutionStatus = "completed"
	StatusCompletedWithConflicts ResolutionStatus = "completed_with_conflicts"
	StatusFailedResolution       ResolutionStatus = "failed_resolution"
	StatusFailedIntake           ResolutionStatus = "failed_intake"
)

// Instruction is one rule delivered to the model — never a fact. The
// ID names the instruction slot; the BodyHash names its revision; the
// SourceRef names its provenance.
type Instruction struct {
	ID        string
	Scope     Scope
	Category  Category
	Protected bool
	Body      string // byte-exact as loaded
	SourceRef string // file path, or "inline:<scope>"
	BodyHash  string // SHA-256 of Body, hex
}

func bodyHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}
