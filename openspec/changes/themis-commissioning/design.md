# Design record: workflow commissioning (D-C-*)

Locked decisions in the order the owner disposed them. `proposal.md`
holds the questions; this file is the record. PROPOSED until LOCKed.
Discipline: the general commissioning contract, never the current demo
workflow.

## D-C-1 — Commissioning is a durable, append-only Themis Governance act made before execution (LOCKED 2026-09-26, owner)

> Commissioning is a Themis-owned, authenticated, pre-execution,
> durable Governance act. The runtime carries the commission identity
> without reinterpretation (the L7 `Origin` principle: recorded
> verbatim, interpreted never) and records its execution against it;
> Themis verifies the correspondence at proposal time. The runtime
> never mints or manufactures a commission.

**The distinction (explicit):** commissioning establishes AUTHORITY to
perform governed work; runtime binding establishes that an execution
CLAIMS to have operated under that authority. Neither replaces the
other. Three independently meaningful facts result:

1. Commissioning fact — Themis says the work was authorized.
2. Execution fact — the runtime says what actually happened.
3. Governance fact — Themis later decides what the work means
   (proposal → decision).

This replaces retrospective inference ("execution mentions Finding X,
the proposal cites the execution, therefore work on X was authorized").

```
BEFORE EXECUTION   Themis Governance commissioning act {Finding, principal, governed method, deployment, commission id}
                          │ commission id
                   themis-ai-runtime {L7 Origin, Finding scope, skill/anchor, execution, verification}
                          │ execution evidence
                   Themis Governance proposal validation: commission ↔ execution consistency check
```

Why (architectural, not demo-driven): the runtime can carry and
reproduce attribution (Finding id + `Origin`) but that does not make it
authoritative; Themis has no durable fact expressing "this work was
authorized against this Finding using this governed method and
deployment"; D-I-5's evidence proves what execution occurred and what
Finding was read, which is different from proving the work was
sanctioned beforehand. A runtime-only binding would establish
execution attribution, not commissioning authority.

**Settled here — the category only.** NOT decided here: exact record
schema, Finding-lifecycle effect, who may commission, API/CLI
transport, commission lifecycle, expiration/revocation, exact matching
rules (Q-C-2..6).

**Invariant carried into Q-C-2 (owner):** no execution may establish
commissioning retrospectively; commissioning must exist as an
authoritative Themis fact before the governed work begins.
