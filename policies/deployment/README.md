# policies/deployment — Deployment Anchors (G1)

An anchor is the Governance-owned, content-addressed declaration of
the exact artifact set under which a deployment may execute
(openspec/changes/g1-deployment-authority/design.md). Admission is a
two-step: caller-supplied bytes/hash IDENTIFY a requested
deployment; only `anchors.json` ADMITS it.

## Current state

The withdrawn anchor's bytes are retained as
`local-dev.withdrawn.json` — the subject of the registration, kept
so the record stays interpretable. It is pre-bundle-schema and will
not parse under the current loader; that is correct, not a defect.

**`local-dev@1`: WITHDRAWN 2026-09-13.** Recorded reason: the anchor
pinned `policies/skills/remediate-dependency/spec-template.json`
where an execution ceiling belongs, and no execution-ceiling
artifact exists in this repository (`policies/execution/` is empty).
An ACTIVE anchor that cannot instantiate a runnable deployment is
not acceptable (owner disposition, finding 5), and the loader must
never be weakened to reinterpret a spec template as a ceiling.

Withdrawal — not deletion — is the correct act: the registration
stays permanently interpretable, and the withdrawn state refuses new
opens while past records remain explicable (D-G1-1A / Q-G1-8).

**No anchor is ACTIVE.** A runnable `local-dev@2` requires:
1. a real execution-ceiling artifact (its `mirror_root` is
   host-specific, so it is deployment configuration rather than a
   repo-committed artifact — this is the open question);
2. the per-workflow bundle form (workflow + its ceilings + its
   contract as one indivisible unit);
3. the constitution pins for the binary that will run it.

Until then, orchestrators run only in the explicitly declared
`Unanchored` test-harness caller role, and production wiring stays
blocked (owner gate).
