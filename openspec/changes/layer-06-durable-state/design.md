# Design: Layer 6 — Durable State

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §9 (layer doc), archived L1–L5 designs (their trace-sink/timing/retention IOUs land here), `ARCHITECTURE.md`, the shipped RunRecord envelope pattern (`internal/service` / benchmarks — manifest-first, options retained).
**Decision IDs** `D-L6-n`; **open questions** `Q-L6-n`.

## 0. Position in the flow

```
L1..L5 typed events (in-memory today)          governed artifacts (hashed)
        │                                              │
        ▼                                              ▼
┌─────────────────────────────────────────────────────────────┐
│ L6 Durable State                                            │
│  task record (envelope)  ·  trace sink (append-only)        │
│  artifact addresses      ·  timing joins here               │
└───────────────────────────────┬─────────────────────────────┘
                                │ operational record
                                ▼
                 Themis governance (system of record)
                 — reads the harness record; is never replaced by it
```

## 1. Hard invariants (inherited, not grillable)

- **Themis is the security system of record.** The harness trace is the operational record of harness behavior; it feeds governance and never establishes competing security truth.
- **No secret in durable state, ever** (3.5; Q-L5-5 contamination handling precedes persistence; L1 secret scan precedes instruction loading).
- **Records are data about decisions, never inputs to them:** nothing in L1–L5's deterministic paths reads the trace sink back to decide anything.
- **Fail closed on the record:** if the durable record cannot be written at a boundary that requires it, that is a typed failure — never a silent best-effort.
- **Deterministic layers stay deterministic:** timestamps live in the sink's envelope, not in event content that any replay/verification hashes.
- **Model output is advisory** — an L6 record of a model's claim is a record of a claim.

## 2. Draft decisions (grill targets)

### D-L6-1 — Task record as the durable envelope
One record per task: task identity, every governed-artifact hash that judged it (EIS, contract, management policy, registry, grant, ceiling, spec, registration), status (the L1 four-status vocabulary generalized), terminal outcome, artifact addresses. RunRecord discipline: manifest-first write so a crashed task keeps attribution.

### D-L6-2 — Append-only trace sink, typed events
The existing event types (Conflict, delivery record, selection trace, AuditEvent, Transition/OpRecord) persist as appended, length-framed, hash-chained entries bound to the task record. No new event vocabulary invented at L6: the layers own their event types; L6 owns durability.

### D-L6-3 — Persistence boundaries
Events persist at defined boundaries (per-call for L4 audit, per-transition for L5, at compose for L2) — the grill decides sync-per-event vs staged, and what "the boundary requires the write" means for each class.

### D-L6-4 — Timing at the sink
Wall-clock timestamps and executor identity attach in the sink envelope at append time (the recorded L4 deferral). Determinism of decision content is untouched.

### D-L6-5 — Crash semantics: typed partial, no resume (v1)
A crash leaves the manifest + whatever the sink holds: a typed, attributable partial record. No checkpoint/resume in v1 — re-execution is an orchestration decision against a fresh environment (locked at Q-L5-11).

### D-L6-6 — Retention: explicit, governed, not yet clever
Retain-all stays the v1 posture, now stated in a governed artifact rather than by omission; deletion/GC vocabulary defined but unimplemented (seam-now pattern, like the credential broker).

### D-L6-7 — Storage substrate: local files first
Same posture as every layer: local-first (JSON/JSONL under a state root), content-addressing reused where immutability is the property; no database dependency in v1.

## 3. Open questions for the grill (Q-L6-n)

1. **Q-L6-1 — Record-plane ownership boundary:** what exactly may the harness's durable record be *used for*? (Reconstruction/audit/governance-feed yes; but can L7 read task records for orchestration decisions? Can L11 evaluate from them? Where is the line that keeps L6 from becoming a second source of truth?)
2. **Q-L6-2 — The unit of record:** task, execution, or turn — and identity/versioning when a task is re-run (new environment identity per Q-L5-12; does the task record link attempts?).
3. **Q-L6-3 — Integrity model:** hash-chained sink entries vs content-addressed batches vs plain append — what tamper-evidence is claimed, against whom, and what is honestly NOT claimed (local-operator model, per the store precedent)?
4. **Q-L6-4 — Persistence-boundary strength per event class:** which events are write-before-proceed (L4 audit before result delivery?) vs write-behind; what fails closed when the sink is unavailable mid-task?
5. **Q-L6-5 — Reconstruction contract:** what exactly does "every model-visible byte reconstructable" require the sink to store vs reference (payload hashes exist — do payloads themselves persist, and where does evidence-verbatim meet retention)?
6. **Q-L6-6 — Timing without lying:** monotonic vs wall clocks, clock-skew honesty, and what timing is model-visible (nothing?) vs trace-only.
7. **Q-L6-7 — Retention/GC vocabulary:** what the governed retention artifact declares; interaction with the ArtifactStore's write-once claim; what deletion even means for hash-chained records.
8. **Q-L6-8 — Themis hand-off:** how governance consumes the record (export? query seam? files?) without L6 growing a query API that becomes load-bearing security surface.
9. **Q-L6-9 — Operational proof gate:** proposed — a live L1→L5 task run persisted end-to-end; process killed mid-task in a second run; both records read back cold and verified (complete vs typed-partial).

## 4. Test plan (three-state discipline)

Fail-closed loaders for the state-root/retention artifacts; manifest-first crash-window tests; sink append/read-back byte-equality; hash-chain verification incl. tamper detection; boundary-strength tests per event class; cold reconstruction of a full task; typed-partial after kill; live proof per Q-L6-9.
