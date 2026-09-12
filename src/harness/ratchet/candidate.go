package ratchet

// The Candidate (D-L11-3): a content-addressed proposal package —
// data, not a runtime artifact, not a source of governance truth.
// Existence confers no authority and creates no evaluation or
// execution path. It has NO lifecycle state (no status field exists
// in this schema, deliberately); proposal disposition is owned by
// the mechanism that receives it. Every Candidate has an
// ATTRIBUTABLE AUTHOR — L11 machinery constructs no Candidates
// (D-L11-14 §4): this type is parsed and stored, never minted here.

import (
	"fmt"
)

// changeRelations is the closed declarative change-relation
// vocabulary (D-L11-3), interpreted only by the receiving door.
var changeRelations = map[string]bool{
	"create-version": true,
	"revise":         true,
	"replace":        true,
}

// BaselineRef is the candidate's CLAIMED baseline (D-L11-5 §6): a
// claim directing comparison, establishing nothing. The comparison
// is conditioned on the door observation, never on this.
type BaselineRef struct {
	Door           string `json:"door"`
	Name           string `json:"name"`
	Version        int    `json:"version"`
	ArtifactSHA256 string `json:"artifact_sha256"`
}

// Candidate is the closed, minimal proposal schema (D-L11-3).
type Candidate struct {
	Artifact string `json:"artifact"` // "l11-candidate"
	Schema   int    `json:"schema"`

	Family         string `json:"family"`
	Target         string `json:"target"`          // target identity at the owning door
	ChangeRelation string `json:"change_relation"` // closed vocabulary

	ContentSHA256 string `json:"content_sha256"` // proposed content commitment

	ClaimedBaseline *BaselineRef `json:"claimed_baseline,omitempty"`

	// Authorship: the attributable act (D-L11-14 §4). AI provenance
	// raises attention, never lowers the bar (D-L9-16).
	Author      string `json:"author"`
	AuthoredVia string `json:"authored_via"` // e.g. "governed-task:<id>", "human"

	// AdvisoryRationale is carried data, explicitly advisory: it
	// establishes nothing and is never quoted as evidence (D-L11-4).
	AdvisoryRationale string `json:"advisory_rationale,omitempty"`

	// EvidenceRefs optionally point at existing comparative-evidence
	// packages BY IDENTITY — references for the door's reading, never
	// inputs to further comparison (L11 output is terminal).
	EvidenceRefs []string `json:"evidence_refs,omitempty"`

	// Supersedes claims lineage by content hash. A claim about
	// origin, never a state change on the referenced candidate
	// (D-L11-11 §5).
	Supersedes []string `json:"supersedes,omitempty"`
}

// ParseCandidate validates candidate bytes fail-closed. Note what is
// ABSENT and must stay absent: status, position, disposition,
// promoted, evaluated — a candidate has no authoritative lifecycle
// state at all (D-L11-3 owner amendment).
func ParseCandidate(raw []byte, origin string) (*Candidate, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrResolve, origin, err)
	}
	dec := jsonDecoder(raw)
	var c Candidate
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrResolve, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrResolve, origin)
	}
	if c.Artifact != "l11-candidate" || c.Schema != 1 {
		return nil, fmt.Errorf("%w: %s: not an l11-candidate@1 artifact", ErrResolve, origin)
	}
	if !Families[c.Family] {
		return nil, fmt.Errorf("%w: %s: unknown candidate family %q", ErrResolve, origin, c.Family)
	}
	if c.Target == "" {
		return nil, fmt.Errorf("%w: %s: target identity required", ErrResolve, origin)
	}
	if !changeRelations[c.ChangeRelation] {
		return nil, fmt.Errorf("%w: %s: unknown change relation %q", ErrResolve, origin, c.ChangeRelation)
	}
	if !shaSyntax.MatchString(c.ContentSHA256) {
		return nil, fmt.Errorf("%w: %s: content_sha256 must be a sha256 hex digest — no identity without bytes", ErrResolve, origin)
	}
	if c.ClaimedBaseline != nil {
		b := c.ClaimedBaseline
		if b.Door == "" || b.Name == "" || b.Version < 1 || !shaSyntax.MatchString(b.ArtifactSHA256) {
			return nil, fmt.Errorf("%w: %s: claimed_baseline must carry door, name, positive version, and artifact hash", ErrResolve, origin)
		}
	}
	if c.Author == "" || c.AuthoredVia == "" {
		return nil, fmt.Errorf("%w: %s: authorship must be attributable (author + authored_via)", ErrResolve, origin)
	}
	for _, ref := range c.EvidenceRefs {
		if !strings2ObjectID(ref) {
			return nil, fmt.Errorf("%w: %s: evidence ref %q is not an object identity", ErrResolve, origin, ref)
		}
	}
	for _, s := range c.Supersedes {
		if !shaSyntax.MatchString(s) {
			return nil, fmt.Errorf("%w: %s: supersedes entry %q is not a content hash", ErrResolve, origin, s)
		}
	}
	return &c, nil
}

func strings2ObjectID(ref string) bool {
	const prefix = "sha256:"
	return len(ref) == len(prefix)+64 && ref[:len(prefix)] == prefix && shaSyntax.MatchString(ref[len(prefix):])
}
