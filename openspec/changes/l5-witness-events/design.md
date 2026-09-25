# Design record: L5 witness events (D-W-*)

Locked decisions of the `l5-witness-events` amendment, in the order the
owner disposed them. `proposal.md` holds the questions; this file is
the record. PROPOSED until the owner LOCKs.

## D-W-1 — L5 writes its own witnesses (LOCKED 2026-09-25, owner; with the classification)

> `l5-transition` and `l5-op` are written by L5 itself through a narrow
> class-restricted sink. L7 continues to orchestrate the sequence
> (seal → egress → bind → COMPLETED) and never becomes the L5 witness.
> Themis remains a consumer. No new event class.

## D-W-2 — `l5-transition` attests one committed edge of the L5 machine (LOCKED 2026-09-25, owner)

> `l5-transition` attests one committed edge of the L5 environment
> state machine. L5 writes exactly one event for every state
> transition, body `{from, to, reason}` matching the committed `Trace`
> transition, at the transition boundary defined by `transitionLocked`.
> The FULL machine is witnessed, not a selective set of edges.

```
PROVISIONING → ACTIVE → SEALED → EGRESSING → ACKNOWLEDGED → TEARDOWN
                                                              ├──► DESTROYED
                                                              └──► TEARDOWN_ANOMALOUS
```

Ordering semantics (asymmetric, per edge — no blanket rule):
- **Seal: record-after-effect.** The OS-level read-only transition is
  the fact; the witness records the completed effect.
- **EGRESSING: record-before-effect.** The event establishes that the
  governed egress operation is being entered; the read follows.
- Every other edge: defined by whether it represents an already-
  completed environmental fact or announces an operation about to
  occur — decided per edge in the implementation table, never by
  uniformity.

D-T-4 consequence — the first production link becomes precise:

```
l5-transition {from: ACTIVE, to: SEALED, reason: task-complete}
      │
l5-op          egress naming the artifact
      │
artifact-bound
      │
lifecycle → COMPLETED
```

Themis refuses, link-named: no `SEALED` transition · `SEALED` with the
wrong reason · `EGRESSING` without the required preceding transition ·
malformed transition body · inconsistent `from/to` · a transition
sequence inconsistent with the closed L5 machine.

Why full-machine witnessing: a selectively witnessed machine leaves
"was the edge absent because it never executed, was not witnessed, or
was forgotten by the implementation?" unanswerable; the machine is
bounded (at most seven edges) and the complete history is useful for
reconstruction beyond Themis v0.

Class restriction (locked here): the L5 sink accepts only the L5
classes through the L5-owned emission path, and the resulting event
carries the canonical L5 writer identity established BY THAT PATH —
never an arbitrary caller-provided writer string. The MECHANISM that
makes the writer non-forgeable is NOT settled by D-W-2; it is Q-W-6.

Timing: none required (the L4 precedent). Sequence gives causal
order; wall-clock time is trace metadata, not part of the security
fact.

## D-W-6 — Writer attribution is a closed class→writer invariant; L5 emits only through a fixed handle (LOCKED 2026-09-25, owner)

> Event writer attribution is a closed class→writer invariant enforced
> by the record sink, and L5 emission is exposed only through an
> L5-scoped handle whose writer identity and event classes are fixed
> by construction. Two independent walls.

**Wall A — record-plane invariant.** The sink owns the authoritative
mapping (`l5-transition → l5`, `l5-op → l5`, `l4-audit → l4`,
`l8-delegation → l8`, `l10-verification → l10`, L7's classes → `l7`,
…) and refuses any `(class, writer)` pair outside it, typed
`ErrConstitution`. For these classes `writer` is part of the record's
structural validity, no longer metadata.

**Wall B — L5 emission capability.** `TaskRecord.L5Sink()` exposes
`Transition(…)` and `Op(…)` only: no writer string, no class string,
no generic `AppendEvent`. The handle cannot emit any non-L5 class.

Why both: the handle alone proves L5 can emit only L5 events, not that
nobody else can; the mapping alone proves the accepted pair is valid,
but a caller could still pass `("l5-transition", "l5")` directly. The
combination closes both directions.

**AST wall (precision, owner):** the proof is architectural, not a
global search for the literal `"l5"`: only the state/emission
implementation owns the L5 writer constant; the handle is the only API
exposed to L5; other packages cannot invoke the L5-specific emission
primitive; sink validation independently rejects forged pairs.
Structural AND runtime proof.

**Constitution/anchor consequence — NOT absorbed here.** Adding
`eventWriters` to the constitution moves `constitution.state` and the
constitution hash; Q-W-4 must deal explicitly with the resulting
anchor re-pin. The implementation must not update the constitution and
keep running under the old anchor.

**What this does not prove:** process authentication. It proves the
governed code paths cannot legitimately misattribute an event under
the defined architecture; G1 binds execution to the authorized
deployment artifact.

## D-W-3 — `l5-op` has exactly two governed forms (LOCKED 2026-09-25, owner)

> `l5-op` attests L5 EXECUTION FACTS, never generic telemetry. Two
> forms only.

**1. Subprocess operation** — one event per governed subprocess
invocation, body `{phase, argv, exit, outcome}` exactly as the
`OpRecord` holds it, record-after-effect at the existing `record()`
site. All four paths are witnessed: refused · budget-exhausted ·
start-failed · completed — a complete governed subprocess history, not
only successes. `MaxRSSByte` stays in the Trace (an observation, not a
bound). The sink's body secret scan applies: `argv` must never become
a way to introduce secret material into the durable record.

**2. Egress operation** — one event, body `{op: "egress", outcome,
artifact_address, observed_total_bytes, observed_file_count}`.
Sequence:

```
L5 EGRESSING → artifact store acknowledges address
             → l5-op egress outcome=acknowledged address=A   (L5's acknowledgement)
             → L7 stores object → artifact-bound address=A   (L6's binding)
             → COMPLETED
```

**Tightening (owner):** a refused/failed egress IS witnessed (typed
outcome, no address) but CANNOT satisfy production. Only
`outcome = acknowledged` is a candidate production link; a failed
egress witness never satisfies the presence requirement of the second
D-T-4 link.

D-T-4 replay, precise:

```
l5-transition ACTIVE→SEALED reason=task-complete
   → l5-transition SEALED→EGRESSING
   → l5-op op=egress outcome=acknowledged address=A
   → artifact-bound address=A
   → object bytes/hash verified
   → COMPLETED
```

Themis refuses, naming the first failed link: egress witness absent ·
outcome not `acknowledged` · ordering wrong · more than one successful
egress witness · bound address differs · durable object does not
correspond to the acknowledged address · chain otherwise broken.

Why full subprocess witnessing: "did not run" and "ran but was not
recorded" must be distinguishable; the ceiling bounds the op count;
future reconstruction gets a complete L5 history without interpreting
`Trace`.

**Trace durability is outside D-W-3.** The events are the durable
witnesses; whether the in-memory `Trace` is additionally materialized
is a separate durability question, not raised here.

## D-W-4 — Constitution hash and anchor re-pin (LOCKED 2026-09-25, owner)

> `eventWriters` MUST be folded into `state.ConstitutionHash()` as
> deterministic `writer:<class>><writer>` components: the writer
> invariant is part of the governed constitution, not an
> implementation detail, and the hash changes exactly once because the
> constitution actually changed. `orchestration.ConstitutionHash()`
> is unchanged (the sink changes how L5 emits, not L7's verbs,
> transition events, or terminal mapping) — asserted by an explicit
> old == new test, never by inspection.

**One host mint: `rsys@6`**, carrying the amended L6 constitution
hash, the `themis_store` pin, and the catalog/skill state T-M5
requires. No intermediate anchor solely for the amendment (a
deployable-but-not-demoable state nobody intends). `rsys@5` is
withdrawn only after `rsys@6` is ACTIVE and independently validated;
the registry stays append-only.

**Invariant:** a constitution hash identifies the exact execution
constitution under which a task was created; an anchor identifies the
governed deployment that executed it. Changing the hash never mutates
historical records — it creates a clean boundary:

```
Pre-D-W-4 task   → old constitution hash → old anchor → historical interpretation (Q-W-5)
Post-D-W-4 task  → new constitution hash → rsys@6 → l5-transition + l5-op witnesses
                                                    → complete five-link production chain
```

`VerifyAnchorRecord` resolving withdrawn anchors establishes
historical ADDRESSABILITY, not compatibility with the five-link proof
— that is Q-W-5's to define.

**Exit proofs (all four required):**

| # | Proof | Required result |
|---|---|---|
| 1 | L6 constitution test | new hash includes `eventWriters` and has the expected new value |
| 2 | L7 constitution test | hash unchanged |
| 3 | Host validation | `rsys@6` ACTIVE with the new constitution and the Themis pin |
| 4 | Governance record | Addendum G records the re-pin and activation |

Plus, before the amendment is called complete: the `constitution.state`
stored in `rsys@6` must equal the compiled `state.ConstitutionHash()`
of the binary ACTUALLY RUNNING on the host (the rsys@4 host-tree
lesson, closed at the constitution too).

## D-W-5 — Constitution-keyed production compatibility (LOCKED 2026-09-25, owner)

> Proof obligations are VERSIONED BY CONSTITUTION; the historical
> record stays immutable. Themis maintains a closed witnessing-
> constitution table, seeded with the amended L6 hash; the compiled
> harness constitution hash must be present in it (build-time guard
> against Themis silently falling behind a constitution change).

**The constitution hash determines what evidence is OWED; the event
stream determines whether that evidence EXISTS.** The branch is taken
from the task's RECORDED `constitution_hash`, never from the running
binary's — replay determinism.

```
read constitution_hash
        ├── witnessing constitution → require the D-W-2 + D-W-3 chain
        └── historical constitution → reconstruct only the evidence that existed
```

| Record | Constitution | L5 witnesses | Result |
|---|---|---|---|
| Historical | old | impossible / absent | inspectable, NOT Position-eligible |
| Current | witnessing | present and valid | Position-eligible (`ProductionWitness = l5-witnessed`) |
| Current | witnessing | missing / invalid | REFUSED `artifact-provenance-refused`, link-named |

Witnessing constitution — the mandatory chain: ACTIVE→SEALED
(task-complete) → SEALED→EGRESSING → `l5-op` egress acknowledged
address A → `artifact-bound` A → COMPLETED; any missing, reordered,
duplicated, unacknowledged, or mismatched witness refuses with the
failing link named. A current record can NEVER fall back to
historical treatment because its witnesses are missing.

Historical constitution — `intake.Resolve` still reconstructs D-T-1,
D-T-2, the three L6 links, and D-T-5, renders `ProductionWitness =
l6-record-only` with the actual historical constitution hash; it is
inspectable; `themis-decide` refuses `position-refused:
production-witness-incomplete`. Historical records are valid
historical evidence and cannot be promoted into a new Themis security
decision without the current provenance requirement — the residual
rejected at T-M3 is not relocated into the Position schema.

Demo consequence: `rsys@6` → new constitution → five-link replay → L10
verification → Position eligibility; the four T-M3 fixtures become
deliberate historical negatives.

## Closure

Grill CLOSED 2026-09-25 with D-W-1..6 locked. The chain: D-W-2
witnesses state transitions · D-W-3 witnesses governed operations and
egress · D-W-6 makes writer attribution non-forgeable · D-W-4 makes it
constitutional and re-anchors · D-W-5 makes the proof obligation
constitution-versioned. The T-M3 boundary is unchanged.
