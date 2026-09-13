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

## The execution-ceiling contract (owner decision, 2026-09-13)

**Deployment-supplied exact bytes, hash-bound by the admitted
anchor.** The execution ceiling carries deployment-specific
configuration (`mirror_root` and friends), so it is NOT a
repo-committed artifact with placeholders — a placeholder would
create a second resolution mechanism (anchor → placeholder →
environment → authority), exactly the hidden deployment input G1
exists to eliminate.

The binding is to the CONCRETE ceiling, never a ceiling "type":

    execution_ceiling: <sha256 of the exact ceiling bytes>

At Open:

    1. resolve the Governance-active Deployment Anchor
    2. obtain the deployment execution ceiling (Config.ExecCeilingPath)
    3. hash its exact bytes
    4. compare against the anchor's execution_ceiling pin
    5. refuse on mismatch (and refuse if it does not LOAD)
    6. freeze anchor + ceiling hash for the Open lifetime

At SubmitTask the envelope's ceiling must BE those bytes: an
anchored task cannot choose its own ceiling. Deployment-supplied
never means submitter-selected.

Two hosts may run the same governed workflows under different
ceilings — each pins different bytes, so each is a different
deployment identity:

    deployment-A: mirror_root=/srv/mirror/A → execution_ceiling=H1
    deployment-B: mirror_root=/opt/mirror/B → execution_ceiling=H2

Consequently a runnable anchor is DEPLOYMENT-INSTANCE-SPECIFIC and
is created where that deployment lives — not committed here. A
reusable "family" of local-dev deployments would be parameterization,
which changes identity semantics and would need its own architecture
decision; it must not be smuggled in as a parameterized anchor.

Scope note: the execution ceiling is deployment-scoped, not
workflow-scoped — it describes where and under what limits THIS
deployment executes. Workflow bundles pin workflow + workflow
ceiling + context contract.
