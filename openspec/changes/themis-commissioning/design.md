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

## D-C-2 — Minimum identity and immutable content (LOCKED 2026-09-26, owner)

> **A Commission is an authority record, not a runtime execution
> record.** It is an immutable, pre-execution Governance fact
> identifying one Finding, one commissioned method, one commissioned
> deployment, and one commissioning principal, together with a
> descriptive premise and rationale. It contains no execution state,
> outcome, task identity, or runtime registry interpretation.

| Field | Content | Disposition |
|---|---|---|
| `commission_id` | Themis-minted UUID v4 | LOCK — Themis identity and the pre-execution reference |
| `finding_id` | exactly one Finding | LOCK |
| `method` | skill `name@version` + `composition_sha256`, opaque | LOCK |
| `deployment` | anchor `name@version` + `artifact_sha256`, opaque | LOCK |
| `commissioned_by` | server-derived `key:<KeyID>` (D10) | LOCK |
| `premise` | Finding stage + current Position version at commission | LOCK — descriptive snapshot, never authority, never a lock |
| `rationale` | optional free text | LOCK — human intent only |
| `raised_at` | UTC | LOCK — informational, never ordering or security evidence |

**Ordering, stated precisely (owner):** the runtime cannot truthfully
reference a commission before Themis has minted it, because the
identifier does not exist before the Governance act; the runtime's
CREATED event then records that identifier in its immutable,
hash-chained history. This proves EXISTENCE-BEFORE-REFERENCE. UUID v4
randomness serves identity, NOT temporal ordering; wall clocks are not
consulted.

```
Themis mints commission_id → commission exists durably → runtime receives it
   → runtime CREATED event records it → execution proceeds
```

**Premise is descriptive, not a concurrency lock:**
`premise.position_version = 7` means "Position 7 existed when this
commission was created", never "Themis guarantees Position 7 stays
current throughout execution". What a moved premise means belongs to
proposal validation / governance semantics.

**Deliberate omissions (locked):** no task id (the task does not exist
yet; Commission → Execution, the execution cites the commission, never
the reverse — a later runtime object never becomes the source of
authority); no skill inputs (runtime execution data; two competing
descriptions of what executed would result); no outcome (a commission
never mutates into a result; Q-C-3 owns lifecycle); no Finding stage
transition (Q-C-4).

**Registries are not validated at commission time (locked):** Themis
records the IDENTITY of what was commissioned; the runtime validates
method and deployment at execution time; Themis equality-checks the
commissioned identity against the execution evidence at proposal time.
Validating runtime registries in Themis would reverse the ownership
model (Governance depending on the runtime's skill and deployment
registries). Themis need not understand what `remediate-dependency@4`
means; it establishes that the execution used exactly the method and
deployment that were commissioned.

## D-C-3 — Commission lifecycle (LOCKED 2026-09-26, owner)

> **A commission is reusable authority, not consumable authorization.**
> Multiple executions may cite it. Withdrawal is a forward-only
> Governance transition that affects future proposal admissibility and
> never rewrites or invalidates historical runtime evidence.

1. **`open` → `withdrawn`, forward-only.** Withdrawal is itself a
   Governance fact with its own authenticated witness and optional
   rationale; the original commission stays immutable history.
   `withdrawn` means "not admissible for future Governance use", never
   "the authority never existed" — essential for reconstruction.
2. **One commission may authorize many executions** (retries, `RetryOf`,
   re-runs). The commission is the one authority fact; executions stay
   independent runtime facts. Governance does not participate in
   ordinary runtime retry behaviour.
3. **Withdrawal is evaluated at PROPOSAL time, on Themis's own
   sequence.** Themis can establish "C existed, then E cited C"; it
   cannot establish `withdraw(C) < task-start(E)` — no shared sequence —
   so retroactive invalidation would be a false temporal claim. Rule:
   execution evidence remains historical; at proposal admission,
   commission `open` → eligible, `withdrawn` → refused
   (`proposal-refused: commission withdrawn`). An execution run while
   the commission was open remains evidence even after withdrawal; a
   later proposal cannot use the withdrawn commission.
4. **No expiry.** Expiry would introduce a clock-based security
   condition where the architecture deliberately uses none; withdrawal
   is the explicit mechanism. Withdrawing because the premise moved is
   governance practice, not mechanism.

**No uniqueness beyond the id:** `C1 = A → F → M → D` and
`C2 = B → F → M → D` are distinct legitimate authority facts; the
execution chooses which it operates under and the proposal names it;
Themis never infers or merges.

| Concern | Owner |
|---|---|
| Commission exists / identity / withdrawal | Themis |
| Execution under commission; retry / re-run | Runtime |
| Historical execution evidence | Runtime / L6 |
| Whether an execution may support a new proposal | Themis |
| Decision about the resulting security posture | Themis |

## D-C-4 — No Finding-stage effect (LOCKED 2026-09-26, owner)

> Commissioning does not constitute investigation progress. A
> commission may exist at any non-terminal Finding stage, and only the
> existing proposal/reopen domain operations may change the Finding's
> investigation stage. Withdrawal has no stage consequence; D7 is not
> amended.

- Commissioning = "authorized to perform governed work"; Under
  Investigation = "a proposal is in flight / flagged for review".
  Different facts, kept different (otherwise authorization would read
  as investigation progress, collapsing D-C-1).
- **Archived is the only refusal** (`commission-refused: finding
  archived`). Identified, Under Investigation, Position Established,
  Monitoring, Resolved: all allowed. Re-examining a Resolved Finding
  uses the existing governed reopen: Resolved → (commission, no stage
  change) → execution → proposal raised → Under Investigation.
- **A commission is a Finding-scoped fact, not a transition:**
  `commissions[]` in the Finding read view; thin Governance-internal
  events `FindingCommissioned` / `CommissionWithdrawn` on the outbox
  (D8); Communication acquires no dependency.
- **`premise.stage` is observational:** the stage observed when the
  commission was created, never the stage resulting from it — a
  historical snapshot, not a hidden lifecycle operation.
