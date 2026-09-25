# Design: Themis v0 — the read door and the decision door

Grill OPEN 2026-09-25 → **CLOSED 2026-09-25: Q-T-1..10 all disposed
(D-T-1, D-T-2, D-T-4..10; Q-T-3 folded into D-T-1), boundaries
B-T-1..3, owner LOCK on each.** §2 holds the locked decisions; §3 the
question table; §4 Gate 0 (implementation whitelist and registers);
§6 the challenge record if the owner opens one. Implementation NOT
STARTED. Proposal:
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

**Implementation note — as recorded (T-M3; RATIFIED AS-RECORDED by the
owner 2026-09-25, with the L5 witness gap kept EXPLICITLY OPEN as the
separate harness amendment `openspec/changes/l5-witness-events/`).**
Owner's rule: T-M3 evidence stands as an as-recorded result; the
absence of L5 writers is a real gap between the constitutional event
model and the implemented record; a proposed event class must not
quietly become an implemented fact source; Themis never invents or
synthesizes L5 witnesses. The harness constitution names
`l5-transition` and `l5-op` as event classes, but no layer writes
them: L5's seal and egress leave no event in the task stream today.
`intake.Resolve` therefore replays the chain over the links that
EXIST — `artifact-bound` (writer l6, one `egress-artifact` reference
that its body names) → object re-hash by the store → lifecycle
`COMPLETED` (writer l6) AFTER the binding → no competing binding →
manifest projection agrees → the egress manifest names the tuple's
task — and refuses link-named on each. The L5 witness is recorded as an
OWED HARNESS AMENDMENT (a record gap, not a Themis mechanism); Themis
never simulates it. T-M4 does not start until that amendment is
classified by the owner as an accepted v0 residual or a required
amendment — it is the one item that changes what Themis is entitled to
claim from the record. When L5 witnessing lands, the two links are
added to the replay and this note is retired.

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

**Implementation note — as recorded (T-M3; RATIFIED AS-RECORDED by the
owner 2026-09-25).** Owner's operational form of D-T-5:

```
L10 verified bytes
        │
        ▼
L5 egress manifest member
        │
        ├── new_hash
        └── content bytes
        │
        ▼
byte/hash identity established
```

The artifact verified is the egress MEMBER of the L5 manifest,
identified by `new_hash` and its content/path; the property that
matters is that the same exact bytes are used in the verification-to-
artifact comparison. "Artifact" is not broadened into the report
itself. The bound `egress-artifact` object is the
L5 EGRESS MANIFEST (`{task_id, changes[{path, type, new_hash, size,
content}], …}`), not the report file. Byte-equality between the raw
verified bytes and the manifest is therefore never true. Check (3) is
implemented as the property it states: the reconstruction's raw bytes
are a MEMBER of the bound manifest — a non-deleted change whose
`new_hash` is the raw bytes' SHA-256 and whose `content` is those
bytes — and the member's path is recorded (`VerifiedPath`). The
verify(A) → egress(B) case refuses exactly as written. The
verification used is the LAST `l10-verification` before the binding
(the fact the harness progressed on); an earlier PASS over other bytes
does not admit. Reconstruction is the seam's exported per-event
function (`seam.ReconstructEvent`, pure: it stores no discrepancy
artifact in a record Themis does not own).

### D-T-6 — Withdrawal versus unavailability (LOCKED 2026-09-25, owner; Q-T-6)

> **Withdrawal changes future admissibility; it does not rewrite or
> invalidate an already-established historical fact. Unavailability or
> corruption prevents re-establishment of that fact and therefore
> fails closed.** If the historical execution and evidence can be
> deterministically reconstructed, withdrawal does not prevent review;
> if reconstruction cannot be established, Themis refuses intake.

| Condition | Themis v0 |
|---|---|
| Task not COMPLETED (FAILED, FAILED_PARTIAL, non-terminal) | refuse — `execution not referencable: not a completed execution` |
| Record verdict TORN / CORRUPT | refuse — no partial acceptance |
| Anchor withdrawn after the execution | proceed — the Position records the anchor hash and its withdrawal state at intake |
| Anchor never registered on Themis's checkout | refuse (D-T-2) |
| Artifact object missing / hash mismatch | refuse (D-T-4) |
| Verification contract withdrawn after the execution | proceed — the reconstruction uses the STORED contract bytes, never today's registry entry; the Position records the contract identity and its state at intake |
| Verification evidence missing / corrupt | refuse (D-T-5) |
| Any Themis-side registry unreadable | refuse — Themis never assumes |

- The registry establishes that the contract was governed; the stored
  contract bytes establish what contract was actually used. Historical
  reproducibility never depends on today's registry entry.
- **Position lifecycle boundary:** once an Enterprise Position exists,
  later withdrawal of an artifact, anchor, or contract it references
  does not invalidate or rewrite it. A Position is a historical
  Governance act; a later decision creates a subsequent state of
  Governance, it does not mutate the historical act. Supersession,
  conflict, and multiplicity are Q-T-7's.
- Reviewer-visible property: withdrawal is not retroactive erasure, and
  missing evidence is never silently tolerated.

### D-T-7 — What a Position is, and the one act that creates it (LOCKED 2026-09-25, owner; Q-T-7)

> A Position is an append-only Themis record about exactly ONE EXISTING
> Finding, expressing a disposition from the closed v0 vocabulary
> (`not-applicable | mitigated | accepted-risk | remediation-planned`)
> and citing exactly one referencable harness execution as its
> evidence basis. It REFERENCES, never copies: the Finding identity,
> the execution tuple, the derived egress artifact object id, the
> verification contract identity and reconstruction verdict, the
> anchor hash with its lifecycle state at intake, the human decision
> witness (D-T-8), and its own hash. The report bytes never become the
> Position; the harness record remains the evidence.
>
> **Only Themis can create a Position, and `themis-decide` is the only
> v0 creation mechanism**: run by a human; resolves the tuple through
> D-T-1..6 and refuses on any failure before showing anything; renders
> the three facts separately (model turns, artifact bytes, L10
> outcome); takes the disposition and a rationale as explicit
> arguments; appends `positions/<finding-id>/<n>.json` with `n` DERIVED
> by Themis as the next unused sequence. No update, no delete, no
> edit, no model-selected disposition, no object-id argument, no
> report-path argument.

- **The Finding must exist** in the authoritative Themis store,
  resolved through Themis's own read door — the Finding id is not
  evidence of existence. Refusal: `position-refused: finding-not-found`.
  This closes the v0 loop: Finding read → model work → governed
  execution → verified artifact → human decision → Position on the
  SAME Finding.
- **Supersession by sequence:** every Position for a Finding remains a
  historical Governance act; the CURRENT Position is a deterministic
  projection (highest valid sequence), never a mutable flag.
- **Anchor state is intake metadata, not disposition:**
  `anchor.state_at_intake` records what a reviewer should know ("made
  from an execution governed by anchor X, withdrawn by intake time");
  it never changes the Position's meaning.

```
Themis Finding → governed read → model reasoning → governed execution
    → verified artifact → human decision → Position (same Finding)
```

### D-T-8 — The decision witness (LOCKED 2026-09-25, owner; Q-T-8)

> The Position record itself is the durable decision witness. Its
> `decision` block is OBSERVED by `themis-decide` from its own process,
> never asserted by an argument: observed OS username, uid, host; UTC
> time; the disposition and rationale exactly as the human supplied
> them; the identities of every evidence item RENDERED to the decider
> (artifact object id, verification record id, the model-turn object
> ids shown, anchor hash); the governed Themis checkout commit; and
> `decider_authentication: observed-not-authenticated`. Any
> `decision.*` key supplied in arguments is refused (the `stampOrigin`
> rule). Decision identity is accountability metadata, not authority:
> the authority is that only `themis-decide` writes Positions and only
> a human runs it.

- **Legitimate human inputs:** disposition, rationale. **Not inputs:**
  the human's identity, the evidence identities (derived from the
  resolved execution and from what was actually rendered).
- **Why the evidence hashes:** the Position records the evidence VIEW
  at the moment of decision, so a reviewer reconstructs Finding →
  tuple → artifact → L10 → model turns shown → anchor → commit →
  human-entered decision. The model's reasoning stays evidence; the
  decision stays a distinct Governance act.
- **`observed-not-authenticated`, retained verbatim:** it tells the
  reviewer the system observed and recorded the OS identity and did
  not cryptographically authenticate the human. Authentication is the
  system-wide residual it already is on every signoff; upgrading it is
  a Governance/security decision for the whole system, never a
  Themis-only mechanism.
- No identity provider · no signing key · no Themis-specific
  authentication root · no caller-asserted identity · no model
  authority.

### D-T-9 — The read door (LOCKED 2026-09-25, owner; Q-T-9)

> The Themis store is a governed, anchor-pinned, read-only registry
> family under `policies/themis/`: `findings.json` and `products.json`
> (append-only registries of immutable records: `id`, fields, `state`
> active|withdrawn, `steward`), plus `positions/` (D-T-7, written only
> by `themis-decide`). A new anchor pin, `themis_store`, hashes the
> Findings and Products registries; Positions are EXCLUDED from the
> model-visible pin. The sole v0 read door is the existing L4 seam: a
> Themis package implements `ThemisSeam.Read(kind, id)` for `finding`
> and `product`, returning the exact record bytes hash-verified at
> load; withdrawn records refuse typed (history, not servable current
> context); unknown ids are `seam-unavailable`. The authority class is
> minted by the L4 registry's `trust: governed-record` on the
> capability, never by the store — the store carries no class field, so
> a malformed record cannot self-declare. The L2 `ThemisReader` seam
> stays unwired in v0.

```
model → get_finding / get_product → L4: grant.themis_scope · quota ·
anchor-pinned store · governed-record classification → Themis record
```

- **Why L4, not L2 composition:** composition-time Themis context
  would need a contract slot, a source registration, and a decision
  about which Findings a task sees before it runs — relevance,
  count, sensitivity, capacity, task-time selection — a real L2/L3
  design question the demo does not need. The tool path already
  carries the controls: scope prefix, quotas, framed result with a
  hash, and the loop D-T-7 requires (a Finding fetched by id → a
  Position on that Finding).
- **Why Positions are unpinned and unserved:** they are Governance
  outputs created after the anchor; pinning them would ask when a
  changing Position registry enters a running deployment's
  model-visible state — not solved for v0.
- **Product, locked minimal:** a name-and-version referential record a
  Finding may identify (`id`, `name`, `version`, `state`, `steward`).
  **`get_product` is referential context only; a Product has no
  independent security disposition in v0.** Not a second
  security-truth model, not a component/SBOM hierarchy, not an
  authorization object, not a remediation state machine.

### D-T-10 — Where Themis lives, and the walls (LOCKED 2026-09-25, owner; Q-T-10)

> Themis v0 is its own Go module, `src/themis` (`module
> github.com/tofchaliss/themis-app`), in the workspace. Themis depends
> on the harness module for READ-ONLY record-plane, deployment, and
> verification contracts (`state`, `deployment`, `verification`,
> `verification/seam`) and on nothing else in it. The harness depends
> on Themis nowhere. `cmd/themis-run` may construct the read seam
> (`store`) and must never import `intake`. The harness executes
> governed work on behalf of Themis; it is not the authority that owns
> Themis state.

Packages and binaries: `themis/store` (anchor-pinned read-only
registries, `ThemisSeam.Read`, D-T-9) · `themis/intake` (D-T-1..8
resolution, the evidence view, the Position append) ·
`cmd/themis-decide` (the human's command, D-T-7/8) · `cmd/themis-run`
gains only the read seam.

```
                     THEMIS
          ┌────────────────────────┐
          │ store        intake    │
          │  READ         WRITE    │
          └─────┬────────────▲─────┘
                │ read seam  │ themis-decide (human)
          ┌─────▼────────────┴─────┐
          │        HARNESS         │
          │   L1 ─────────── L11   │
          │   NO Position writer   │
          │   NO intake dependency │
          └────────────────────────┘
```

**The write operation does not exist in the harness authority
surface**: the Position writer is not "not exposed as a tool"; it is
unreachable from the model execution path by the dependency graph.
Stronger than `model → write_position → refused`.

Four walls, each a test:
1. **Dependency wall** — the ENTIRE harness module's dependency graph
   (`go list -deps` over every harness package, not only direct
   imports) contains no `themis-app/intake`; an indirect path
   `harness → A → B → intake` is a violation.
2. **Binary wall** — `cmd/themis-run` imports `themis-app/store` only,
   never `intake`.
3. **Writer wall** — `intake` has exactly one write site
   (`positions/<id>/<n>.json`, `O_EXCL` so a sequence number can never
   be reused), no other `os` writer, and no import of `tools`,
   `orchestration`, `execution`, or `runtime/model`. The store package
   has no `os` writer at all — a Themis read-store immutability wall
   (its own boundary, not an L9 rule). Both tested by AST against the
   intended boundary, never by package naming alone.
   **Scope of Wall 3 (owner, 2026-09-25, documented — no
   architecture change):** Wall 3 is a DIRECT-import wall. `intake` has
   a direct dependency on the read-only verification reconstruction
   seam (`verification/seam`); the seam's transitive implementation
   dependencies (orchestration, tools, runtime/model) are not
   prohibited by Wall 3. Nobody may read the wall as "intake cannot
   reach orchestration/tools/model"; the wall's claim is that `intake`
   itself imports none of them and holds no call site into them.
4. **Capability wall** — the L4 registry contains no Position-writing
   capability, verb, target class, executor, or error vocabulary.

Core invariant: *the harness may consume Themis-governed records
through the read seam, but the harness dependency graph contains no
path to the Themis Position writer. Position creation is exclusively a
Themis-owned human command.*

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
| Q-T-6 | Withdrawn / unavailable task, anchor, artifact, verification record? | withdrawn → proceed (recorded); unavailable/corrupt → refuse; Positions never rewritten | LOCKED → D-T-6 |
| Q-T-7 | What exact act creates the Enterprise Position? | `themis-decide` only; append-only numbered record about an existing Finding; closed dispositions; references never copies | LOCKED → D-T-7 |
| Q-T-8 | How is the human decision witnessed? | the Position record; decision block observed never asserted; evidence view recorded; `observed-not-authenticated` | LOCKED → D-T-8 |
| Q-T-9 | The read door: what, which class, pinned how? | L4 `ThemisSeam` only; findings + products anchor-pinned (`themis_store`); L4 mints the class; withdrawn unservable; Positions unpinned; Product minimal | LOCKED → D-T-9 |
| Q-T-10 | Package, store, command; the walls | own module `src/themis`; Themis → harness one-way; store/intake/themis-decide; four walls incl. whole-graph dependency | LOCKED → D-T-10 |

## 4. Gate 0 — implementation whitelist (LOCKED with the grill, 2026-09-25)

Anything not listed here stops implementation and is classified first.

**Themis-owned (`src/themis`, module `themis-app`)**
- `store`: loaders for `policies/themis/findings.json` and
  `products.json` (append-only registries, strict keys, immutable
  records, `state`, `steward`; withdrawn refuses on read), the
  `ThemisSeam` implementation, the `themis_store` hash.
- `intake`: `Resolve(tuple)` implementing D-T-1..6 with link-named
  refusals; `EvidenceView` (model turns shown, artifact bytes, L10
  reconstruction) with the identities of everything rendered; `Append`
  writing `positions/<finding>/<n>.json` under `O_EXCL`; `Current`
  (highest sequence projection); the Position record type (D-T-7/8).
- `cmd/themis-decide`: the human command (tuple, `--disposition`,
  `--rationale`; refuses any `decision.*`; observes identity).
- `cmd/themis-inspect` (read-only): renders a Finding's Positions and
  re-verifies one Position against the record (reconstruction).

**Existing-layer amendments**
- G1 anchor: pin `themis_store` (sha or `"absent"`); Open verifies it
  against `Config.ThemisStorePath` the way it verifies the delegation
  registry; `rsys@6` proposed.
- `cmd/themis-run`: constructs `store` and passes it as the L4
  `ThemisSeam`; never imports `intake`.
- `themis-status` / `themis-preflight`: print the pin; verify the two
  registries load.
- Skill: `remediate-dependency@3` whose grant template grants
  `get_finding` (scope `FIND-`) and `get_product` (scope `PROD-`), and
  whose procedure tells the model to read the Finding first;
  `investigate-cve` untouched. The P0 Finding `FIND-2026-0001` and
  Product `PROD-demo-vuln-app` authored as PROPOSED registrations.
- G2 fact table: row `themis_position` — established by a human
  through `themis-decide`, witnessed by the Position record.

**Explicitly absent**: L2 `ThemisReader` wiring · any harness import of
`intake` · a Position capability/verb/executor · automated acceptance ·
Position edit/delete · identity provider or signatures · feeds, KB,
enrichment, SBOM, component hierarchy · Positions in the model-visible
pin.

**Proof registers**
- **A — Admission/authority**: store loaders (closed schema, withdrawn,
  key wall), intake refusals (each D-T-1..6 link, link-named), the
  four walls (D-T-10), `decision.*` refused.
- **B — Positive path FIRST**: a real record from the harness
  (anchored `remediate-dependency@3` walk that read `FIND-2026-0001`
  through `get_finding`, egressed, L10 PASS) → `themis-decide` creates
  Position 1 → `themis-inspect` re-verifies it CONFIRMED; then each
  negative twin (verify-then-mutate, PASS on other bytes, FAILED task,
  unregistered anchor, missing object, unknown Finding, withdrawn
  Finding read, second Position supersedes, `O_EXCL` collision).
- **C — Record**: the Position's evidence view re-derives from the
  harness record; a Position survives withdrawal of its anchor,
  contract, and Finding; corruption → refused, never partial.
- **D — Laundering**: the Finding enters the model as
  `governed-record`; the model's restatement re-enters at the floor
  (record projection); nothing the model wrote appears in the Position
  except by the human's rationale.
- **E — Live**: the demo walk (VM): live model reads the Finding, runs
  the workflow to COMPLETED/PASS under `rsys@6`; the operator creates
  the Position; `themis-inspect` shows Finding → execution → artifact
  → verification → decision.
