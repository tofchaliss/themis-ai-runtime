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
