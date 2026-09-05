package instructions

import (
	"errors"
	"fmt"
	"sort"
)

// ErrIntake marks failures caused by what the caller handed in — the
// untrusted task payload or its exemption records — as opposed to
// failures of the trusted instruction environment. StatusOf maps it
// to StatusFailedIntake.
var ErrIntake = errors.New("task intake rejected")

// SourceRef records which root or payload fed a resolution — trace
// provenance, not content.
type SourceRef struct {
	Kind Scope
	Root string // directory root, or "inline:<kind>" for payloads
}

// EffectiveSet is the immutable value produced by one resolution —
// the complete trace-metadata shape Layer 6 will store:
// {Hash (EIS), SourceHashes, Sources, Conflicts, PolicyHash,
// ExemptionsApplied, Status}. Same sources and config ⇒ same Hash,
// regardless of Source argument order. There is no mutation
// operation: instruction change is a new resolution (design §6,
// Q-L1-5).
type EffectiveSet struct {
	Instructions      []Instruction     // resolved order: (scope, id)
	Conflicts         []Conflict        // recorded rejections, deterministic order
	Hash              string            // canonical set hash — the version
	SourceHashes      map[string]string // id -> body hash (admitted instructions)
	Sources           []SourceRef       // which roots/payloads fed the set
	PolicyHash        string            // pattern policy that judged the load
	ExemptionsApplied []Exemption       // suppressed detections, deterministic order
	Status            ResolutionStatus  // completed | completed_with_conflicts
}

// Resolve is the single entry point: Load → order → hash. Atomic per
// Q-L1-6: it yields exactly the configured EffectiveSet or nil with
// an error — never a partial, degraded, or fallback environment.
func Resolve(cfg Config, sources ...Source) (*EffectiveSet, error) {
	// An empty EIS is not a valid delivered environment: no role, no
	// safety rules. Fail closed rather than deliver a bare header.
	if len(sources) == 0 {
		return nil, fmt.Errorf("%w: no instruction sources registered", ErrSourceUnavailable)
	}
	res, err := Load(cfg, sources...)
	if err != nil {
		return nil, err
	}
	set := &EffectiveSet{
		Instructions:      res.Instructions,
		Conflicts:         res.Conflicts,
		SourceHashes:      map[string]string{},
		PolicyHash:        res.PolicyHash,
		ExemptionsApplied: res.ExemptionsApplied,
		Status:            StatusCompleted,
	}
	sort.SliceStable(set.Instructions, func(i, j int) bool {
		a, b := set.Instructions[i], set.Instructions[j]
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.ID < b.ID
	})
	sort.SliceStable(set.Conflicts, func(i, j int) bool {
		a, b := set.Conflicts[i], set.Conflicts[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Class != b.Class {
			return a.Class < b.Class
		}
		return a.BodyHash < b.BodyHash
	})
	sort.SliceStable(set.ExemptionsApplied, func(i, j int) bool {
		a, b := set.ExemptionsApplied[i], set.ExemptionsApplied[j]
		if a.PatternID != b.PatternID {
			return a.PatternID < b.PatternID
		}
		return a.InstructionID < b.InstructionID
	})
	for _, inst := range set.Instructions {
		set.SourceHashes[inst.ID] = inst.BodyHash
	}
	for _, src := range sources {
		ref := SourceRef{Kind: src.Kind, Root: src.Root}
		if src.Inline != nil {
			ref.Root = fmt.Sprintf("inline:%s", src.Kind)
		}
		set.Sources = append(set.Sources, ref)
	}
	sort.SliceStable(set.Sources, func(i, j int) bool {
		a, b := set.Sources[i], set.Sources[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Root < b.Root
	})
	set.Hash = eisHash(set.Instructions)
	if len(set.Conflicts) > 0 {
		set.Status = StatusCompletedWithConflicts
	}
	return set, nil
}

// StatusOf maps a Resolve error to the explicit run-result status
// (design §6, Q-L1-6): conflicts never look like ordinary success,
// and failures name whose problem they are.
func StatusOf(err error) ResolutionStatus {
	switch {
	case err == nil:
		return StatusCompleted
	case errors.Is(err, ErrIntake):
		return StatusFailedIntake
	default:
		return StatusFailedResolution
	}
}
