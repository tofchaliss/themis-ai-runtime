# Design: Themis v0 — the read door and the decision door

Grill OPEN 2026-09-25 (owner-led, one question at a time, facts from
the record before opinions). §2 holds locked decisions; §3 the question
table; §6 the challenge record when the owner opens it. Proposal:
`proposal.md` (scope A, the out-of-scope list, the three boundaries).

## 0. Position in the flow

```
Themis store (Findings, Products, Positions)
   │ read door: ThemisSeam / ThemisReader        governed-record ─► model context
   │                                                                     │
   │                                              model proposes / acts (L1–L11)
   │                                                                     │
   │                                              egress artifact bound (L6) · L10 PASS
   │                                                                     │
   └─ decision door: operator command ◄── referencable execution tuple ──┘
          │  Themis derives + verifies everything from the record plane
          ▼
     HumanDecision (witnessed) ─► Enterprise Position
```

The harness never writes to Themis. The model never touches either
door's authority: the read door delivers under a class the model cannot
change; the decision door is an operator act the model cannot invoke.

## 1. Hard invariants (inherited, not grillable)

- ARCHITECTURE.md: Themis owns security truth, Findings, Enterprise
  Positions, Security Governance. The harness may write to Themis only
  through Themis-owned interfaces; acceptance into security truth
  requires the appropriate human or governed Themis decision; a
  governed automated decision is not AI self-authority.
- G1: a caller-supplied hash IDENTIFIES a deployment; only the anchors
  registry ADMITS it. G2: an L6 object is a fact of kind F iff a
  committed event of F's minting class names it.
- Authority classes are minted at their owning doors only; the model's
  output is `external-untrusted` at every hop (C-L8-18).
- Withdrawal governs new use, never history (L8 D-L8-8, C-L8-14 G; L9
  D-L9-10; G1 Q-G1-8).

## 2. Locked decisions

### D-T-1 — The referencable execution tuple (LOCKED 2026-09-25, owner; Q-T-1, folds Q-T-3)

> A Themis-referencable harness execution is identified by the tuple
> `(deployment_anchor_hash, task_id, artifact_bound_event_seq)`. The
> tuple is the ADMISSIBILITY HANDLE Themis uses to resolve an execution
> act from the harness record plane; Themis derives and verifies
> everything else. The caller supplies the execution tuple and never an
> object id: a caller cannot say "accept object X", only "this governed
> execution act", after which Themis determines what that act bound.

Resolution chain (every step is existing harness code; Themis adds no
identity mechanism):

```
(anchor hash, task id, artifact-bound seq)
   │
   ▼ ReadManifest(task)
   ├── status terminal COMPLETED, verdict VERIFIED
   └── governed_hashes.deployment_anchor == anchor hash
   │
   ▼ VerifyAnchorRecord(recorded hash, anchor bytes from the record, anchors registry)
   └── the anchor is REGISTERED (its current state is irrelevant to history:
       withdrawal after the fact does not unmake the execution)
   │
   ▼ ReadEvents(task) → event at seq
   ├── class artifact-bound
   └── Refs = [{ObjectID, egress-artifact}]  → the artifact identity, DERIVED
   │
   ▼ GetObject(ObjectID)  (re-hashed by the store; corruption is a verdict, not a refusal)
```

Rules folded in:
- **Why a triple, not "anchor + object address":** one content address
  can be bound by two tasks (objects dedup across tasks), so the
  address alone does not name an execution; the binding event does.
  The manifest, not the object, carries `deployment_anchor`; identity
  runs through the manifest.
- **Unanchored tasks are not referencable.** `deployment_anchor =
  "unanchored"` is refused before any further resolution.
- **The anchor in the tuple is not authorization re-established from
  the caller's hash.** Themis performs the G1 two-step: the supplied
  hash identifies the requested deployment; the exact anchor bytes
  come from the record; registration is checked in the anchors
  registry; then the manifest binding is used.
- **No new L6 identity mechanism, no new artifact identity mechanism,
  no caller-supplied ObjectID authority.**

### D-T-2 — Authoritative deployment identity and its resolution (LOCKED 2026-09-25, owner; Q-T-2)

> The authoritative deployment identity for Themis is the exact
> `deployment_anchor` bytes materialized in the task's own record; their
> SHA-256 must equal `manifest.governed_hashes.deployment_anchor`. The
> anchor's Governance registration is resolved from the `anchors.json`
> registry of the governed checkout in which Themis operates — match by
> artifact hash, two-way name/version agreement, any lifecycle state.
> **Themis MUST NOT establish deployment identity from the record
> alone: the record identifies the anchor; Governance registration in
> the authoritative deployment registry establishes that the anchor was
> governed.**

```
task record ── deployment_anchor object (exact bytes)
            └─ manifest.governed_hashes.deployment_anchor
                     │ hash equality, else REFUSE
                     ▼ parse anchor bytes
          governed checkout / anchors.json
                     │ hash → name/version (two-way)
          ┌──────────┴──────────┐
     registered            never registered
     (any lifecycle)            │
          │                  REFUSE
          ▼
   historical execution
```

- The record is authoritative for EXECUTION identity, not sufficient
  for GOVERNANCE identity: accepting a self-consistent anchor because
  its hash matches the manifest would let an execution introduce its
  own deployment authority into the record — the bootstrap G1 closed.
- Themis never reads a caller-supplied anchor file; the caller
  identifies the execution tuple, Themis reconstructs the anchor from
  the durable record.
- Lifecycle read historically: registered+active → referencable;
  registered+withdrawn → referencable for history; never registered →
  not referencable. Withdrawal affects new opens, never recognition of
  a previously governed execution.
- Deployment coupling: Themis and the harness resolve against the same
  governed checkout, or Themis receives a pinned copy of the
  authoritative registry as a deployment property. A consistency
  requirement, not an identity mechanism.
- No new registry · no Themis write access to deployment governance ·
  no caller-supplied anchor authority · no content-only identity · no
  invalidation of history by later withdrawal.

### D-T-4 — Artifact production is established by causal replay of the record (LOCKED 2026-09-25, owner; Q-T-4)

> Themis establishes that the artifact was PRODUCED by the governed
> execution by replaying the authoritative record and verifying the
> complete production chain from governed L5 egress to artifact
> binding and terminal completion. `artifact-bound` is not the
> production witness by itself; it is the final binding in a
> multi-event witness. Themis obtains the artifact bytes only after
> the replay succeeds.

```
l5-transition  seal task-complete
      │
l5-op          egress naming the artifact address
      │
L6 object      bytes re-hash to the address
      │
artifact-bound Ref{ObjectID, egress-artifact} — the SAME address
      │
lifecycle      → COMPLETED
```

Mandatory checks, all in the tuple's own stream: the sealing
transition precedes the egress; the egress names the address the
binding names; the object re-hashes to it; ordering holds; terminal
COMPLETED follows the binding; no competing `artifact-bound` exists
for the execution; every event's writer is the layer that owns its
class; no model turn and no L8 event can establish the production
fact.

- **Fail-closed, link-named:** any required link absent, duplicated,
  inconsistent, mis-ordered, or attributable to an unauthorized writer
  refuses with a typed reason naming the failed link
  (`artifact-provenance-refused: missing l5-op egress witness for
  artifact <ObjectID>`), never a generic "invalid artifact".
- **No signature, no attestation, no new L6 witness class.** A signed
  egress receipt answers "can an independent party authenticate this
  receipt?"; Themis v0's question is "can I mechanically reconstruct
  whether this artifact was produced by this governed execution?",
  which the record plane already answers (ordered events, durable
  objects, hashes, layer-owned writers, lifecycle, chain verification).
  Adding a signature would be a new L6 mechanism for an undemonstrated
  gap.
- No caller-supplied artifact identity; no trust in `artifact-bound`
  alone. Themis stays a consumer of the harness's authoritative
  evidence, never a competing execution-truth subsystem.

### D-T-5 — The L10 fact is re-established over the exact artifact (LOCKED 2026-09-25, owner; Q-T-5)

> Themis never reads the outcome from the `l10-verification` event
> body as a claim. The PASS used for intake must be a REPRODUCIBLE PASS
> for the EXACT egress artifact identified by D-T-4: Themis runs the
> existing L10 reconstruction over the task's verification events and
> requires (1) a `Consistent` reconstruction with outcome PASS under a
> contract registered in Themis's governed checkout (`contracts.json`,
> any lifecycle state, two-way identity); (2) its `execution_ref`
> resolving to an authorized `l4-audit` of a verifier-eligible
> capability in the same stream, whose `ResultHash` equals the hash of
> the reconstruction's raw bytes; (3) the raw bytes BYTE-IDENTICAL to
> the egress artifact bytes obtained under D-T-4; (4) the verification
> event's seq preceding the `artifact-bound` seq. The L7 gate that fired
> on the outcome is not consulted.

```
D-T-4 egress artifact ── exact bytes ──┐
                                        ▼
L10 reconstruction: registered contract · Consistent · PASS ·
authorized verifier audit (ResultHash = raw hash) ·
raw bytes == artifact bytes · verification seq < binding seq
                                        │
                                        ▼
                          eligible for human intake
```

- **Why (3):** without it, `verify_report(A) → PASS` then `egress(B)`
  is internally consistent and would accept B on A's PASS. The
  verifier can operate on workspace content mid-walk; the artifact is
  produced at completion; the equality makes the PASS about THIS
  artifact.
- **Consequence, accepted:** verify → mutate → egress yields no
  verified artifact for v0. A workflow needing that pattern needs
  another governed verification after the mutation, producing a new
  fact for the final artifact — never a weaker intake contract.
- **Why not the L7 gate:** the gate proves what the harness required
  for its own progression; Themis independently establishes whether
  the evidence satisfies the INTAKE contract. Otherwise Themis would
  trust another layer's conclusion about its own conclusion.
- Named refusals: `no reproducible PASS` · `contract not registered` ·
  `verifier execution not authorized` · `verification evidence
  unavailable` · `verified bytes are not the bound artifact` ·
  `verification does not correspond to artifact production`.
- No new verifier · no new verification event type · no trust in the
  recorded outcome alone · no trust in the L7 gate · no PASS over
  different bytes.

### Boundaries locked with D-T-1 (owner, 2026-09-25)

- **B-T-1 — Human decision only.** The decision door is structurally
  `HumanDecision`. A governed automated decision is a separate
  mechanism with its own grill; it must not be introduced as an
  extension of the human door's proof.
- **B-T-2 — L10 PASS is admissibility, not correctness.** No PASS, no
  intake; but PASS establishes only that the artifact satisfies the
  registered contract. The acceptance view exposes model turns,
  artifact bytes, and the L10 outcome as three independent things; the
  human decision is the semantic acceptance act.
- **B-T-3 — The write boundary is proven structurally.** The harness
  holds no Position capability, executor, or seam, and an import/AST
  wall proves harness packages never reference the Themis write path.
  Capability absence is a consequence of ownership; the wall is the
  evidence.

## 3. Grill — question table

| # | Question | Recommendation | Disposition |
|---|---|---|---|
| Q-T-1 | What constitutes a Themis-referencable harness execution? | the triple, derived artifact, unanchored refused | LOCKED → D-T-1 |
| Q-T-2 | Which deployment identity is authoritative, how resolved? | record bytes identify; governed registry establishes; any lifecycle | LOCKED → D-T-2 |
| Q-T-3 | Which L6 object/event identifies the artifact? | — | folded into D-T-1 |
| Q-T-4 | How does Themis verify the artifact was produced by that execution? | causal replay l5-transition → l5-op → object → artifact-bound → COMPLETED; fail-closed, link-named | LOCKED → D-T-4 |
| Q-T-5 | How does Themis verify the L10 result? | reproducible PASS, registered contract, authorized audit, raw bytes == artifact, verification precedes binding; L7 gate not consulted | LOCKED → D-T-5 |
| Q-T-6 | Withdrawn / unavailable task, anchor, artifact, verification record? | — | open |
| Q-T-7 | What exact act creates the Enterprise Position? | — | open |
| Q-T-8 | How is the human decision witnessed? | — | open |
| Q-T-9 | The read door: what, which class, pinned how? | — | open |
| Q-T-10 | Package, store, command; the walls | — | open |
