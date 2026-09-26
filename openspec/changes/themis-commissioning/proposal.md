# Proposal: workflow commissioning — where the Finding → work relationship becomes authoritative

Status: **OPEN 2026-09-26 — grill open (owner-led, one question per
turn).** Completion-matrix row 4 (`themis-integration/completion-matrix.md`).
Discipline (owner): **do not design this around the current demo
workflow; design the general commissioning contract the demo can later
instantiate.** The architecture determines the demo, never the reverse.

## The gap (facts from code, 2026-09-26)

**Themis side.** A Finding's lifecycle is Identified → Under
Investigation → Position Established → Monitoring → Resolved → Archived
(EDR-GOVERNANCE-01 D7); every transition is a domain operation emitting
a completed-fact event. Under Investigation is entered by
`RaiseProposal` and by the governed reopen. Governance events are
FindingOpened/Resolved/Reopened/Archived, ProposalRaised/Accepted/Rejected,
PositionEstablished/Revised. **There is no concept of work commissioned
against a Finding**: no work item, assignment, or "investigation
started by principal P using method M" record, in Governance or
anywhere else in Themis.

**Runtime side.** L9 instantiation takes caller inputs (validated
against the skill's closed input schema; serialized as the payload and
hashed into the spec) and an `Origin` map — OPAQUE governed attribution
recorded verbatim by L7 as `origin:<key>` governed hashes in the CREATED
event and "interpreted by nobody" (D-L9-13). The record therefore holds
the Finding id as a caller input and any attribution the caller chose
to state, but nothing that says anyone with authority sanctioned work on
that Finding.

**Integration side (decided).** The proposal's `harness-execution/v1`
evidence (D-I-5) will name the tuple and the Finding bytes the execution
read. Business Verification vouches that the evidence refs (component
PURL, CVE) belong to the Finding — it does not say the work was
commissioned. So today the Finding → task relationship exists first in
the harness envelope and reaches Themis only at proposal time, after the
fact. That is the gap row 4 resolves.

## The relationship to establish (owner)

```
Finding
   │  commissioned by Governance          ← where does this become authoritative?
   ▼
Work / Task  →  Execution  →  Evidence  →  Proposal  →  Position
```

## Grill — question table (dependency order)

| # | Question | Recommendation (facts-first; PROPOSED) | State |
|---|---|---|---|
| Q-C-1 | Is commissioning a durable Themis Governance act, or a runtime task binding? | **LOCKED 2026-09-26 → D-C-1:** a Themis-owned, authenticated, pre-execution, durable, append-only Governance act; runtime carries the id verbatim (`Origin`), never mints it; Themis verifies correspondence at proposal time; commissioning = authority, binding = claim; invariant: never retrospective. | LOCKED |
| Q-C-2 | Minimum identity and immutable content | **LOCKED 2026-09-26 → D-C-2:** commission_id (UUID v4), finding_id, method (name@version + composition hash), deployment (name@version + artifact hash), commissioned_by (`key:<KeyID>`), premise (descriptive), rationale, raised_at (informational); existence-before-reference ordering; no task id, inputs, outcome, or stage change; runtime registries never validated by Themis. | LOCKED |
| Q-C-3 | Commission lifecycle | **LOCKED 2026-09-26 → D-C-3:** `open` → `withdrawn` forward-only with its own witness; many executions per commission; withdrawal evaluated at proposal time on Themis's sequence, never retroactive; no expiry; no uniqueness beyond the id. | LOCKED |
| Q-C-4 | Effect on the Finding's investigation stage | none by itself (Under Investigation stays "proposals in flight"); a stage change would be a D7 amendment and needs its own decision | OPEN |
| Q-C-5 | How the runtime carries it and how Themis verifies it two-way | envelope `Origin["commission"] = <id>` (D-L9-13, no L7/L9 semantic change); recorded as `origin:commission` in L6; `themis-intake` puts it in the evidence; Governance checks anchor/skill/finding of the evidence against the commission at proposal time | OPEN |
| Q-C-6 | Who may commission; separation from proposer and decider | product-scoped write key; commissioner may equal the proposer; decider must differ from both (operational, per D-I-6) | OPEN |

## Not in scope

No harness runtime change beyond carrying an opaque id · no automatic
commissioning by Themis · no scheduling or assignment system · no change
to L7/L9 semantics · no demo-specific fields.
