package instructions

// Skill instruction source — the Layer-9 activation of the ScopeSkill
// slot the L1 grill pre-provisioned ("awaits Layer 9 registered skill
// identity"). Mirrors the repository activation posture (D-L5-8):
// a skill procedure becomes an instruction source ONLY through this
// constructor, which demands bytes matching the pin it was given —
// hand-built ScopeSkill sources are structurally unrecognizable.
// Activation establishes byte integrity ONLY: whether the composition
// was governance-registered is a catalog property this path cannot
// observe and never asserts. The procedure text is advisory technique with zero
// authority (D-L9-2): it enters the untrusted-source tier, so
// directive-pattern violations are rejected-and-recorded, never
// silently delivered, and the L1 constitution cannot be shadowed by
// it (namespace ownership).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// skillProcedureID is the fixed instruction identity for a task's
// activated procedure text. Fixed — not derived from the skill name —
// so the consumer (L7) interprets nothing about skills; the skill's
// identity travels in task attribution, never in instruction ids.
const skillProcedureID = "skill.procedure"

var procedureSHASyntax = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ActivateSkillSource turns verified procedure bytes into the one
// legitimate ScopeSkill source. The caller supplies the bytes it
// resolved and the pinned hash from the governed skill composition;
// a mismatch is a refusal, never a best-effort delivery (D-L9-11:
// re-reads re-verify; mismatch means the reviewed composition can no
// longer be affirmed).
func ActivateSkillSource(procedure []byte, pinnedSHA256 string) (Source, error) {
	if len(procedure) == 0 {
		return Source{}, fmt.Errorf("%w: empty procedure artifact", ErrSourceUnavailable)
	}
	if !procedureSHASyntax.MatchString(pinnedSHA256) {
		return Source{}, fmt.Errorf("%w: skill procedure pin must be a sha256 hex digest", ErrSourceUnavailable)
	}
	sum := sha256.Sum256(procedure)
	if hex.EncodeToString(sum[:]) != pinnedSHA256 {
		return Source{}, fmt.Errorf("%w: skill procedure bytes do not match the pinned hash", ErrSourceUnavailable)
	}
	inst := Instruction{
		ID:       skillProcedureID,
		Scope:    ScopeSkill,
		Category: CategorySkill,
		Body:     string(procedure),
	}
	return Source{Kind: ScopeSkill, Inline: []Instruction{inst}, activated: true}, nil
}
