# Proposal: skill identity vs instantiated composition identity (L9/L7 admission)

Status: **GRILL NOT STARTED.** Decision questions set, nothing locked.
No implementation before the grill closes and Gate 0 passes — the
existing implementation is provably wrong, and an obvious correction
must not become an architecture decision by being written.

Raised by: `docs/development/finding-anchored-skill-admission-2026-09-15.md`,
GitHub issue #1 (`needs-triage`).

## The defect that raised it

Under an anchored deployment — the production configuration —
`verifyAnchoredSkill` compares the envelope's **composition seal**
against the catalog's **registered `composition_sha256`**. Those are
different identity domains, and two of the seal's nine members (`grant`,
`spec`) are produced per task, so the seal can never be a registered
constant. The comparison succeeds for no input.

Both catalog skills are ACTIVE and unreachable. The finding document
holds the evidence; this proposal holds only the question.

## The two identity domains

```
Governance catalog
        │  registered skill manifest
        ▼
skill.json ──► composition identity = sha256(skill.json bytes)
 ├── workflow                 ┐
 ├── workflow_ceiling         │
 ├── context_contract         ├─ FIXED by the skill
 ├── grant_template           │
 ├── spec_template            │
 ├── input_schema             │
 └── procedure                ┘
        │
        │  instantiate (L9)
        ▼
task envelope
 ├── fixed members      ──────────► must correspond to the manifest — HOW is the question
 ├── instantiated grant ──────────► task-specific
 ├── instantiated spec  ──────────► task-specific
 └── composition seal   ──────────► integrity of THIS instance
```

The seal's job is visibly the third one: `checkSeal` proves the
identities accompanying it were not altered after sealing. Whether it
has any further role is part of what must be decided.

## The tension at the centre

`origin` is declared opaque — *"recorded verbatim, interpreted by
nobody"* (D-L9-13), and `themis-run` *"exercises zero semantics on it"*.
Yet `verifyAnchoredSkill` reads `origin["skill"]` and makes an
**admission decision** from it.

So the fact that determines *which skill a task claims* already rides in
a field the architecture calls opaque, and it is already load-bearing.
That is not incidental to this defect; it is the shape of the problem.
Any correspondence rule has to say where the claimed identity legitimately
comes from.

## Decision questions (draft; the owner sets the final set and IDs)

### Ownership and boundary

| Q | Question |
|---|---|
| Q-SA-1 | Which layer owns the correspondence check — L7 at assembly (as today), L9 at instantiation, or both at different strengths? |
| Q-SA-2 | Is L7 reading `origin["skill"]` an accepted exception to opacity, or a defect in its own right? If accepted, does the exception extend to other `origin` keys, or is it exactly one? |
| Q-SA-3 | Must the claimed skill identity travel in a **load-bearing** envelope field rather than in attribution? If so, what becomes of `origin`'s skill keys — duplicate, or attribution only? |

### The correspondence itself

| Q | Question |
|---|---|
| Q-SA-4 | Which members are FIXED by the skill and which are INSTANTIATED? The split above is a reading of the current artifacts, not a decision. |
| Q-SA-5 | Must every fixed member be compared individually against the manifest's pins, or does resolving the manifest and comparing one derived value suffice? What does each choice fail to catch? |
| Q-SA-6 | What, if anything, constrains the instantiated members at admission beyond the existing grant ⊆ ceiling and spec ⊆ ceiling checks? Specifically: must L7 verify the effective grant is a **narrowing** of the skill's `grant_template`, or is that solely L9's at instantiation? |
| Q-SA-7 | What role does the seal retain? Is instance integrity its whole job, or does it participate in admission? |

### Scope

| Q | Question |
|---|---|
| Q-SA-8 | Should the same correspondence be enforced for **unanchored** deployments? Today `verifyAnchoredSkill` runs only when anchored, so a skill-attributed task under an unanchored deployment receives no catalog check at all. Is that asymmetry intended? |
| Q-SA-9 | A task claiming a **withdrawn** skill: refuse new submission while past records stay interpretable, by analogy with anchors? Or something else? |
| Q-SA-10 | Does a deployment anchor pin the skill *catalog* only, or also which skills within it a deployment may run? (`rsys@3` pins `skill_catalog`; the allowlist concept exists for models and not for skills.) |

### Governance

| Q | Question |
|---|---|
| Q-SA-11 | Is this an implementation defect inside locked decisions, or does the chosen correspondence require a D-L9 and/or D-G1 amendment? |
| Q-SA-12 | What becomes of Phase C row C17? It asserts a doctored composition is refused, which an unconditional refusal satisfies. What is its positive counterpart, and where does it live? |

## Constraints inherited (not grillable)

Model output advisory, never authority. A submitter selects an anchored
skill, never its constituent hashes (owner finding 1). Governance owns
registration; L9 instantiates, it does not register. L7 stays
hash-comparing and learns nothing about skills. Fail closed; nothing
defaulted. Author ≠ governance ≠ machinery. L6 owns the record. Past
records stay interpretable across supersession. Day-0 prohibitions.

## Gate 0 criteria (proposed)

- [ ] Every Q-SA-* answered and locked as a decision, or recorded as a
      residual with its own gate.
- [ ] The chosen correspondence stated as a **rule a reviewer can check
      against the implementation**, not as a description of the code.
- [ ] Where the claimed skill identity legitimately comes from is stated
      explicitly, and the `origin` opacity question is settled either way
      rather than left implicit.
- [ ] **A positive-path test is mandatory**: an anchored deployment
      accepting a genuine L9-instantiated skill submission. Not optional
      under any chosen correspondence. Its absence is what let this
      defect exist.
- [ ] C17 retained, plus its positive counterpart, so that
      "refuse everything" can never again satisfy the gate.
- [ ] Mutation-verified both ways: suppressing the check must fail the
      **negative** test, and a comparison that refuses unconditionally
      must fail the **positive** one.
- [ ] A live anchored run on `rsys@3` submitting `remediate-dependency@1`
      through `themis-run`.

## What this proposal does not do

It answers no question above. It does not propose which members to
compare, and deliberately does not repeat the correction that is obvious
from the defect — the whole point of the sequence agreed on 2026-09-15 is
that the finding precedes the decision, and the decision precedes the
implementation.

It also does not reopen L9. This looks like a narrowly scoped
cross-layer admission-integrity gap inside an already locked design.
Whether it reaches further is for the grill to determine, not for this
document to assume.
