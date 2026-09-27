# Proposal: L5 witness events — who writes `l5-transition` / `l5-op`, and what they attest

Status: **CLASSIFIED 2026-09-25 (owner): REQUIRED AMENDMENT, pre-T-M4.**
Themis v0 T-M4 remains BLOCKED until the seven exit conditions below
are met. **Grill CLOSED 2026-09-25: Q-W-1..6 all LOCKED (`design.md` D-W-1..6).**
Implementation plan: `tasks.md` W-M1..W-M4; T-M4 unblocks after W-M3.

## Owner classification (2026-09-25, verbatim in substance)

> L5 witness events are a required pre-T-M4 harness amendment. The
> current T-M3 evidence remains valid as historical/as-recorded
> evidence, but `intake.Resolve` must not claim the full D-T-4
> causal-production proof for new Position intake until L5-owned
> witnesses exist.

Why not a residual (owner): (1) it is an OWNERSHIP gap, not missing
metadata — the classes exist constitutionally and lack their
authoritative writer; treating L6-written representations as
substitutes turns "L5 witnessed its own execution" into "L6 recorded
what another layer says L5 did", the cross-layer authority ambiguity
the architecture has consistently refused; (2) Themis v0's
differentiator is independent reconstruction, and the demo must
withstand "how do you know the artifact came from the governed
execution?" at exactly this point; (3) the amendment is small and
already shaped — L5 owns `Trace` (seal reason, transitions, ops,
artifact address, egress outcome); a narrow class-restricted sink
gives L5 a controlled route to its own classes; L7 keeps orchestrating
and does not become the L5 witness; Themis stays a consumer; no new
event class.

```
Current v0 evidence                    Architectural claim (D-T-4)

L6-written record                      L5-owned execution
    ├── seal-related link                  ├── environment transition
    ├── egress-related link                └── governed egress operation
    └── artifact binding                            │
             ▼                                      ▼
       artifact identity                        L6 record
                                                    ▼
                                              artifact binding
```

Nuance (owner): the four T-M3 artifacts are NOT invalidated and the
earlier implementation is not "wrong". Historical T-M3 evidence →
valid as-recorded; new Themis Position intake → blocked until the
D-T-4 causal witness is complete. Implemented as
`intake.Resolution.ProductionWitness = "l6-record-only"`, rendered in
the evidence view, so the claim Themis makes is exactly what the
record supports.

## Exit conditions (owner) — T-M4 unblocks when ALL hold

1. L5 owns the `l5-transition` and `l5-op` witness writes.
2. Writer restrictions are defined and tested.
3. Constitution/registry pin implications are resolved.
4. Pre-amendment records have an explicit compatibility rule.
5. `intake.Resolve` replays the five-link chain.
6. Positive and negative proofs cover the new witness boundary.
7. Mutation probes cover attempts to forge / omit / substitute L5
   witnesses.

Raised by: `openspec/changes/themis-v0/design.md` D-T-4 implementation
note; `openspec/changes/themis-v0/RESUME-HERE.md` Gate 1 — T-M3.

## The gap

The L6 constitution (`src/harness/state/constitution.go`) names two
event classes, `l5-transition` and `l5-op`, in its closed vocabulary.
No harness layer writes either: there is no `AppendEvent(state.EvL5Transition, …)`
or `AppendEvent(state.EvL5Op, …)` anywhere in non-test code. The
classes exist in the constitutional event model and are absent from
every implemented record.

D-T-4 (Themis v0, LOCKED) describes production as a chain

```
l5-transition  seal task-complete
l5-op          egress naming the artifact address
L6 object      bytes re-hash to the address
artifact-bound Ref{ObjectID, egress-artifact}
lifecycle      → COMPLETED
```

whose first two links do not exist in the record. `intake.Resolve`
replays the three that do, link-named on refusal, and the owner
ratified that as an as-recorded result. Themis does not and must not
synthesize the missing two.

## What the record holds today (facts from code)

- **L5 keeps its own trace, in memory, never in the task stream.**
  `execution.Env` carries a `Trace` with `SealReason`, `Ops`,
  `ArtifactAddress`, `EgressOutcome`, workspace/provider identities
  (`execution/lifecycle.go`, `execution/egress.go`). It is L5-owned
  evidence that never reaches L6.
- **L5 does not import `state`.** `execution` imports `confine` and
  `internal/strictjson` only. It has no handle on a task record and
  cannot append to one as written.
- **L7 drives the completion path and writes what the record has.**
  `orchestration/loop.go` `complete()`: `l5.Seal(SealTaskComplete)` →
  `l5.Egress(ceiling, spec, store)` (returns the artifact-store
  address) → readback → `task.StoreObject(egress-artifact, bytes)` →
  `task.BindArtifact(objID)` (L6 writes `artifact-bound`, writer l6) →
  `l5.Teardown()` → `task.Transition(COMPLETED)` (L6 writes
  `lifecycle`, writer l6). L7 records nothing about the seal or the
  egress op itself.
- **Seal is a mechanism with a typed reason** (`task-complete`,
  `deadline`, `fatal-breach`, `caller-abort`) and makes the workspace
  OS-level read-only; egress is reachable only from a clean seal
  (`beginEgress`). The seal reason is exactly the fact D-T-4's first
  link wants witnessed, and it exists — in L5's memory.
- **The egress manifest already names the task** (`task_id`,
  `spec_hash`, `ceiling_hash`, `binary_digest`, `provider`, `changes[]`),
  and the bound object IS that manifest. The second link's content
  (the egress op naming the artifact address) is derivable from the
  binding today; what is missing is the L5-attributed witness that
  the egress OPERATION happened, in order, before the binding.

- **Writer attribution is caller-asserted today.** `AppendEvent(class,
  writer, …)` records whatever writer string the caller passes; the
  constitution restricts only `primitiveOnlyEvents` (recovery,
  verdict, artifact-bound, lifecycle) to L6's own primitives. There is
  no per-class writer rule for `l4-audit`, `l10-verification`,
  `l8-delegation`, or the two L5 classes — "every event's writer is the
  layer that owns its class" is a convention of the emitting code, not
  a sink-enforced invariant. Exit condition 2 (writer restrictions
  defined and tested) therefore reaches beyond L5.
- **`Trace` already holds the transition list**: `Transitions
  []Transition{From, To, Reason}` appended at every `transitionLocked`
  (PROVISIONING → ACTIVE → SEALED → EGRESSING → ACKNOWLEDGED → TEARDOWN
  → DESTROYED | TEARDOWN_ANOMALOUS), and `Ops []OpRecord{Phase, Argv,
  Exit, Outcome, MaxRSSByte}`. The witness content exists; only its
  route to the record is missing.

## What the amendment must decide (grill, owner-led)

| # | Question | Recommendation (facts-first; PROPOSED) |
|---|---|---|
| Q-W-1 | Who is the WRITER of `l5-transition`/`l5-op`? | **LOCKED 2026-09-25 (owner, with the classification):** L5 itself, through a narrow class-restricted sink; L7 keeps orchestrating and is never the L5 witness. Was: L5 itself, through a narrow sink interface (append-only, class-restricted to the two L5 classes) handed in at provision — "every event's writer is the layer that owns its class" (D-T-4) is the rule the constitution already states for every other class; L7 writing on L5's behalf would make L7 the fact source for L5 facts. |
| Q-W-2 | What does `l5-transition` attest, and when is it written? | **LOCKED 2026-09-25 → D-W-2:** one event per committed edge of the full L5 machine, body `{from, to, reason}` = the `Trace` transition, at `transitionLocked`; seal record-after-effect, EGRESSING record-before-effect, other edges decided per edge; no timing; class restriction locked, writer non-forgeability deferred to Q-W-6. |
| Q-W-3 | What does `l5-op` attest? | **LOCKED 2026-09-25 → D-W-3:** two forms — one event per governed subprocess op `{phase, argv, exit, outcome}` on all four paths, record-after-effect, secret scan applies, `MaxRSSByte` excluded; one egress event `{op: egress, outcome, artifact_address, observed_total_bytes, observed_file_count}` after the store acknowledges and before L7 stores/binds; only `acknowledged` can satisfy production; Trace durability out of scope. |
| Q-W-4 | Constitution hash and anchor re-pin? | **LOCKED 2026-09-25 → D-W-4:** fold `eventWriters` into the L6 hash (`writer:<class>><writer>`); L7 hash unchanged, asserted by test; ONE host mint `rsys@6` carrying the new constitution + `themis_store` + T-M5 catalog/skills; `rsys@5` withdrawn only after `rsys@6` ACTIVE and validated; history never mutated; four exit proofs incl. Addendum G; host binary's compiled hash must equal the stored pin. |
| Q-W-5 | What does Themis do when it lands; pre-amendment records? | **LOCKED 2026-09-25 → D-W-5:** closed witnessing-constitution table (compiled hash must be in it); branch on the RECORD's `constitution_hash`; witnessing → five-link chain mandatory, `l5-witnessed`, Position-eligible; historical → evidence that existed, `l6-record-only`, inspectable, `themis-decide` refuses `production-witness-incomplete`; a current record never falls back to historical. |
| Q-W-6 | How is the L5 writer identity made NON-FORGEABLE? | **LOCKED 2026-09-25 → D-W-6:** Wall A — sink-enforced closed class→writer mapping for every class (`ErrConstitution`); Wall B — `TaskRecord.L5Sink()` with `Transition`/`Op` only, writer and classes fixed by construction; architectural AST proof; constitution hash / anchor re-pin explicitly deferred to Q-W-4; not process authentication (G1's). |

## Classification — made (see status above)

- **Accepted v0 residual:** the demo and Themis v0 proceed on the
  three-link replay; the Position records that L5 witnessing was not
  available in the record it cites; the amendment is scheduled after
  v0.
- **Required amendment:** the amendment lands (harness change,
  constitution check, VM proof) before T-M4, and `intake.Resolve`
  replays five links from the start.

Recommendation (PROPOSED): **accepted v0 residual**, on the facts
above — the three implemented links are all L6-written, ordered,
hash-verified, and the egress manifest is task-named; the missing
witnesses add L5 attribution and ordering evidence, not a new
identity. The Position must NAME the residual so the claim Themis
makes is exactly what the record supports.

## Not in scope

No new event class · no signature or attestation · no Themis-side
synthesis of L5 events · no change to D-T-4's locked intent.
