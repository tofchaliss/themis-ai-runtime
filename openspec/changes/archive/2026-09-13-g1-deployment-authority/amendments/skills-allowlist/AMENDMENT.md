# G1 Amendment: deployment-level Skill allowlist (2026-09-22)

Authorized by: D-SA-9 (LOCKED, owner, 2026-09-22), D-SA-10, D-L10-17.
Additive G1 amendment; the G1 archive stands.

## Amendment definition

1. **Anchor schema (deployment/anchor.go):** optional `skills[]` — the
   exact `name@version` set this deployment may run. Entries exact
   (no floating refs), unique, bounded (64). Absent admits no
   skill-attributed task. Closed schema otherwise unchanged
   (`skill_overrides` or any unknown field still refuses — D-SA-10:
   the anchor selects, never composes).
2. **Assembly gate (orchestration.SubmitTask, anchored):** `skill ∈
   anchor.skills` by exact equality, checked after the instruction
   plane and before the registry/bundle checks and before any catalog
   resolution; refusal names the allowlist ("a skill enters a
   deployment only by Governance act"). `workflows[]` remains the
   independent bundle-level gate.
3. **Deployment adoption:** an allowlist change is an anchor change —
   new anchor + restart; running tasks frozen; history untouched.
   `rsys@4` (Governance act, owner) will carry `skills[]`.

## Affected invariants — none weakened

D-G1-1/1A (identify ≠ admit), frozen authority at Open, the
consumption-pin pattern. The existing `skill_catalog` pin is unchanged
and remains authoritative for composition resolution.

## Evidence

`deployment` TestSkillAllowlistRules; `orchestration`
TestAnchoredSkillAllowlist (absent → refuse; unlisted → refuse; listed →
passes this gate). Commit dc8034c.
