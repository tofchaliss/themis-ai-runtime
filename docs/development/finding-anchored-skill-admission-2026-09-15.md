# Finding: anchored skill admission refuses every legitimate submission

Date: 2026-09-15. Deployment `rsys@3`
(`b5f551ab70769e7424f823579715cbd4b2d317ec358ceb1c9574bfe930636b8c`).
Tracked as GitHub issue #1 (`needs-triage`).

This document separates three things that are easy to run together, and
stops at the third:

1. **Observed fact** — what happened, reproducibly.
2. **Control failure** — what the code does, and why it cannot succeed.
3. **Architectural question** — what must be decided, by the owner.

No implementation fix is proposed here. The existence of an obvious
correction must not become an implicit architecture decision.

---

## 1. Observed fact

An envelope was produced by the ordinary L9 path —
`skills.Instantiate(catalog, "remediate-dependency@1", req)` — and
submitted through the production surface `cmd/themis-run` against the
active anchored deployment:

```
task      : rsys-gpt-3
status    :
deployment: b5f551ab7076
refusal   : task-assembly refused: the submitted composition is not the
            composition remediate-dependency@1 registers — a submitter
            selects an anchored skill, never its constituent hashes
exit=1
```

Exit 1 is the refused-submission code. **No task was created and nothing
entered the record plane.** The deployment opened correctly; the
refusal is at assembly.

Both catalog skills — `investigate-cve@1` and
`remediate-dependency@1` — are ACTIVE, and the anchored workflow bundle
in `rsys@3` is exactly `remediate-dependency@1`'s:

```
workflow         c10826f24c2612bb7c3135bec152f40a1b7941ffe09d5f735ebbaac536c731a9
workflow_ceiling 06bf8c0a9f2c06f7297dcdab90f7b12618ad72224ad288e712c391012cbf0f0f
context_contract dfb4ade9c5fe1f2d6f8b285ec9099c407842fa16f7a48f87cea051e9d4ed8f96
```

## 2. Control failure

Two identities are in play, and they belong to different domains:

| Value | What it is |
|---|---|
| `fd59500a97dec812…` | catalog `composition_sha256` — `sha256` of the `skill.json` bytes. The **registered skill manifest identity**. |
| `4ab9302a9ac90e35…` | the envelope's `composition.composition_sha256` — the **instantiated composition seal**. |

Verified independently: `sha256` of `policies/skills/remediate-dependency/skill.json`
equals the catalog entry exactly, and the envelope's
`origin.skill_composition` carries that same `fd59500a…`. **L9 records
the registered identity correctly.**

`orchestration/orchestrator.go`, `verifyAnchoredSkill`:

```go
if env.Composition.Seal != entry.Composition { … refuse … }
```

`Seal` is `sha256` over the canonical serialization of nine members
(`orchestration/envelope.go`, `sealedFields`): workflow,
workflow_ceiling, context_contract, grant_template, spec_template,
input_schema, procedure, **grant**, **spec**.

Two of those — `grant` and `spec` — are produced by instantiation and
are task specific. **The seal therefore varies per task and can never be
a registered constant.** The comparison cannot succeed for any input.

The envelope is not malformed. `checkSeal` passes: the seal is the
correct hash of the identities accompanying it, and every artifact hash
in the commitment matches the manifest's pins. Only the cross-layer
comparison is wrong.

### Why no test caught it

`verifyAnchoredSkill` has one caller and **zero Go tests**. Its only
coverage is Phase C row C17, which supplies a *doctored* composition and
asserts a refusal. Against the current implementation:

```
legitimate composition → REFUSE
doctored composition   → REFUSE
```

so C17 passes. It established `bad input is refused` and never the
required pair, `bad input is refused AND good input is accepted`. **A
control that refuses unconditionally satisfies every asymmetric test of
itself.**

The skill end-to-end tests run unanchored, so the check never executes
there. Every production run before today used a hand-written envelope
with no skill attribution. The defect was unreachable by any existing
test and by every prior live run — it required an anchored deployment
and a genuinely instantiated skill envelope in the same submission,
which had never happened.

This is the same class as the 2026-09-14 mutation pass findings and as
Addendum D's rechecker defect: **a check must not claim more than it
establishes.** Here it appears at the admission layer rather than the
diagnostic one, which is why it matters more.

### Impact

- L9 skills are registered, anchored, and unreachable under G1.
- The governed procedure cannot reach a model in production. Two live
  runs on 2026-09-15 failed at exactly the two points
  `remediate-dependency@1`'s procedure addresses — the report's required
  field shape, and declaring phase completion.
- Phase C row C17 should be treated as **unproven** pending a positive
  counterpart.

## 3. The architectural question (owner-owned)

The current cross-layer comparison **conflates registered skill identity
with instantiated composition identity**. The owner must decide the
correct correspondence between the registered skill manifest and the
instantiated envelope: which members constitute the admission
comparison, and how the per-task members are treated.

That decision spans L9 and L7, with G1 supplying the deployment context,
and it may or may not require an amendment to the locked L9 decisions.
It is not made here, and no fix is implemented ahead of it.

The sequence agreed with the owner:

```
FINDING
   ↓
OWNER DECISION
   ↓
D-L9 / D-G amendment if required
   ↓
implementation
   ↓
positive-path proof
   ↓
mutation tests
   ↓
anchored rsys@3 live run
```

The positive-path test is not optional in that sequence: whatever
correspondence is chosen, an anchored deployment accepting a genuine
skill submission must be asserted, so that "refuse everything" can never
again satisfy the gate.

## What this finding does not establish

- **It does not say the L9 identity model is wrong.** This looks like a
  narrowly scoped cross-layer admission-integrity gap inside an already
  locked design. Whether it reaches further is for the owner decision to
  determine.
- **It does not establish that `gpt-oss:20b` can complete the governed
  task.** Run 1 showed the model drives the workflow far enough to make
  genuine workflow mistakes — it explored, transitioned phases, wrote
  files, and reached the L10 gate. That is useful capability evidence
  and nothing more.
- **It does not change any deployment.** `rsys@3` remains ACTIVE and
  correctly configured; the refusal is not a deployment defect.
