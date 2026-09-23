# L7 Amendment: anchored skill admission sequence (2026-09-22)

Authorized by: D-SA-1..10 (LOCKED, owner, 2026-09-22), D-L10-17. The
L7 archive stands; this record is additive and corrects the
2026-09-15 defect at `verifyAnchoredSkill`.

## Amendment definition (orchestration/orchestrator.go, envelope.go)

Anchored, skill-attributed `SubmitTask` now runs, in order:

```
seal self-consistency (LoadEnvelope, ErrInvariant)
  → instruction plane + constitution pins
  → skill ∈ anchor.skills                       (D-SA-9, ErrAssembly)
  → tool registry pin
  → workflow bundle ∈ anchor.workflows + C15    (G1)
  → D-SA-3: unattributed skill_procedure_path → ErrAssembly
  → catalog hash vs anchor.skill_catalog
  → Resolve(skill): registered, ACTIVE, manifest two-way   (D-SA-8)
  → seven-member equality vs manifest pins       (D-SA-2, ErrAssembly)
  → model ∈ anchor.models
  → Claim 1: materialized artifacts vs commitment (ErrInvariant)
  → grant read; Claim 1 grant; Instantiates(grant, manifest grant_template)
    + SpecInstantiates(spec, manifest spec_template)   (D-SA-4)
  → grantWithinCeiling → registry → provision → EIS
  → record: governed["skill"], skill_catalog + skill_manifest bytes
```

- `verifyAnchoredSkill` selects by the `skill` field (D-SA-5), never
  `origin`; returns the resolved catalog + manifest.
- `grantAuthorityDigest` includes `template_scope` (set digest).
- No δ change, no control vocabulary change, no envelope path added.

## Affected invariants — none weakened

D-L7-2/3/5/6 (δ untouched), D-L7-10 (three-verb seam unchanged),
D-L7-11, D-L7-12. L7 remains hash-comparing: it reads pins, applies
the L4/L5 relations, and interprets nothing.

## Evidence

`orchestration/skill_admission_test.go`; walls
`TestActivateSkillSourceSingleCallSite`, `TestSealHasNoGovernanceConsumer`
(type-checked). Live anchored walk PASS. Commit dc8034c.
