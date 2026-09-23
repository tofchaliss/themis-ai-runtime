package tools

// The `delegate` capability — L4's half of Layer 8 (registry-v5; the L8
// amendment to the archived L4 layer, openspec/changes/l8-subagents
// §5.2, D-L8-4, D-L8-15, C-L8-5, C-L8-11, C-L8-13). L4 owns: the target
// class and its exact-scope gate (authorize.go), the closed delegation
// error classes, the evidence-reference SHAPE, and the executor whose
// "execution" is the deterministic INSTANTIATION of the delegation —
// resolve the template, re-establish every evidence reference against
// the parent's record, compose in memory, check bounds — producing an
// instantiation capture (identities only) as its audited evidence.
// The composition object, the model call, the output, and the
// `l8-delegation` witness are the post-hook's (L7/L8 seam), which
// re-derives the composition without reading the capture (C-L8-13).
// L4 executes no model and owns no composition semantics: the
// instantiation half is injected, exactly as the L10 evaluator is.

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// delegationRefSyntax: an exact delegation-template reference.
var delegationRefSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*@[1-9][0-9]*$`)

// evidenceRefSyntax: one reference into the parent's own stream,
// "<seq>:<objectID>" (C-L8-5) — the declaring event's sequence and the
// content address it references. A bare object id is not a reference.
var evidenceRefSyntax = regexp.MustCompile(`^([1-9][0-9]*):(sha256:[0-9a-f]{64})$`)

// EvidenceRef is one parsed reference. Nothing here is verified: the
// seam re-establishes existence, reachability, integrity, and class
// from the record.
type EvidenceRef struct {
	Seq      int64
	ObjectID string
}

// MaxEvidenceRefs bounds the reference count structurally; the L4
// argument cap already implies a smaller bound (~218 references in
// 16 KiB), this is the explicit closed form (C-L8-7).
const MaxEvidenceRefs = 256

// ParseEvidenceRefs parses the `evidence` argument: comma-separated
// references, no spaces, empty = none. Shape only.
func ParseEvidenceRefs(s string) ([]EvidenceRef, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) > MaxEvidenceRefs {
		return nil, fmt.Errorf("%d references exceed the cap %d", len(parts), MaxEvidenceRefs)
	}
	out := make([]EvidenceRef, 0, len(parts))
	for _, p := range parts {
		m := evidenceRefSyntax.FindStringSubmatch(p)
		if m == nil {
			return nil, fmt.Errorf("%q is not a <seq>:<objectID> reference", bound(p))
		}
		seq, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%q: bad sequence", bound(p))
		}
		out = append(out, EvidenceRef{Seq: seq, ObjectID: m[2]})
	}
	return out, nil
}

// Closed delegation error classes (C-L8-11). A refusal to INSTANTIATE
// (stage B, D-L8-15) is "delegation-refused:<reason>" with the reason
// drawn from the closed set below; the three post-instance outcome
// classes name the seam's typed failures re-entering the parent.
const (
	ErrDelegationProviderError         ErrorClass = "delegation-provider-error"
	ErrDelegationOutputOverBound       ErrorClass = "delegation-output-over-bound"
	ErrDelegationModelIdentityMismatch ErrorClass = "delegation-model-identity-mismatch"

	delegationRefusedPrefix = "delegation-refused:"
)

// DelegationRefusalReason is the closed reason vocabulary for stage B.
type DelegationRefusalReason string

const (
	RefusalTemplateUnresolvable  DelegationRefusalReason = "template-unresolvable"
	RefusalTemplateWithdrawn     DelegationRefusalReason = "template-withdrawn"
	RefusalTemplateHashMismatch  DelegationRefusalReason = "template-hash-mismatch"
	RefusalRegistryUnreadable    DelegationRefusalReason = "registry-unreadable"
	RefusalEvidenceUnreachable   DelegationRefusalReason = "evidence-unreachable"
	RefusalEvidenceNotPrior      DelegationRefusalReason = "evidence-not-prior"
	RefusalEvidenceDuplicate     DelegationRefusalReason = "evidence-duplicate"
	RefusalEvidenceRegistryDrift DelegationRefusalReason = "evidence-registry-drift"
	RefusalEvidenceSlotAmbiguous DelegationRefusalReason = "evidence-slot-ambiguous"
	RefusalResolveFailed         DelegationRefusalReason = "resolve-failed"
	RefusalComposeRefused        DelegationRefusalReason = "compose-refused"
	RefusalBriefOverBound        DelegationRefusalReason = "brief-over-bound"
	RefusalSeamUnavailable       DelegationRefusalReason = "seam-unavailable"
)

var delegationRefusalReasons = map[DelegationRefusalReason]bool{
	RefusalTemplateUnresolvable: true, RefusalTemplateWithdrawn: true,
	RefusalTemplateHashMismatch: true, RefusalRegistryUnreadable: true,
	RefusalEvidenceUnreachable: true, RefusalEvidenceNotPrior: true,
	RefusalEvidenceDuplicate: true, RefusalEvidenceRegistryDrift: true,
	RefusalEvidenceSlotAmbiguous: true, RefusalResolveFailed: true,
	RefusalComposeRefused: true, RefusalBriefOverBound: true,
	RefusalSeamUnavailable: true,
}

// DelegationRefused builds the typed stage-B error class. An unknown
// reason yields the seam-unavailable class rather than an open-ended
// string: the vocabulary is closed at the constructor.
func DelegationRefused(reason DelegationRefusalReason) ErrorClass {
	if !delegationRefusalReasons[reason] {
		reason = RefusalSeamUnavailable
	}
	return ErrorClass(delegationRefusedPrefix + string(reason))
}

// KnownErrorClass reports membership in the closed executor error
// vocabulary — the pre-L8 classes plus the delegation families.
func KnownErrorClass(c ErrorClass) bool {
	switch c {
	case ErrFileUnreadable, ErrSeamUnavailable, ErrOversized, ErrTimeout, ErrWriteRefused,
		ErrDelegationProviderError, ErrDelegationOutputOverBound, ErrDelegationModelIdentityMismatch:
		return true
	}
	if strings.HasPrefix(string(c), delegationRefusedPrefix) {
		return delegationRefusalReasons[DelegationRefusalReason(strings.TrimPrefix(string(c), delegationRefusedPrefix))]
	}
	return false
}

// ErrDelegationRefusal is the typed error an instantiator returns for a
// stage-B refusal; Class is the closed reason.
type ErrDelegationRefusal struct {
	Reason DelegationRefusalReason
	Detail string // record-side detail (audit), never model-visible
}

func (e *ErrDelegationRefusal) Error() string {
	return string(DelegationRefused(e.Reason)) + ": " + e.Detail
}

// DelegationInstantiator is the injected compose half of the L8 seam
// (C-L8-13): given the authorized selection, it resolves the template,
// re-establishes the evidence against the parent's record, composes in
// memory under the L2 caps, and returns the deterministic instantiation
// CAPTURE — identities only (template hash, evidence triples, EIS hash,
// contract hash, payload hash), never bytes. A *ErrDelegationRefusal
// is the stage-B refusal; any other error is machinery failure.
// Bound per task by the orchestrator at assembly; L4 holds no state
// root, no registry path, and no model.
type DelegationInstantiator interface {
	Instantiate(template string, evidence []EvidenceRef, brief string) (capture []byte, err error)
}

// execDelegate is the `delegate` executor: pure reads under the
// registry timeout, no model call. A nil instantiator is fail-closed
// (assembly refuses a phase exposing `delegate` without a delegator;
// reaching here is the seam-unavailable class, never a silent pass).
func execDelegate(inst DelegationInstantiator) Executor {
	return func(entry *GrantEntry, args map[string]any, target string) Outcome {
		if inst == nil {
			return Outcome{ErrClass: DelegationRefused(RefusalSeamUnavailable)}
		}
		ev, _ := args["evidence"].(string)
		refs, err := ParseEvidenceRefs(ev)
		if err != nil {
			// Authorize already refused this shape; a second look is
			// the executor's own fail-closed posture.
			return Outcome{ErrClass: DelegationRefused(RefusalEvidenceUnreachable)}
		}
		brief, _ := args["brief"].(string)
		capture, err := inst.Instantiate(target, refs, brief)
		if err != nil {
			var r *ErrDelegationRefusal
			if errors.As(err, &r) {
				return Outcome{ErrClass: DelegationRefused(r.Reason)}
			}
			return Outcome{ErrClass: ErrSeamUnavailable}
		}
		if len(capture) == 0 || len(capture) > maxToolEvidence {
			return Outcome{ErrClass: ErrOversized}
		}
		return Outcome{Evidence: capture}
	}
}
