# Design record: Themis integration (D-I-*)

Locked decisions, in the order the owner disposed them. `proposal.md`
holds the questions; this file is the record. PROPOSED until LOCKed.

## D-I-1 — Intake topology (LOCKED 2026-09-25, owner; deployment fact: same host)

> `cmd/themis-intake` is a Themis-owned CLI running on the same
> enterprise VM as the harness. It receives the execution TUPLE (never
> a record path or an ObjectID), resolves it through the locked
> D-T-1/D-T-2 machinery against the local, read-only harness record
> plane (manifest, event stream, object store), renders the evidence
> view, and raises a Decision Proposal over the authenticated
> Governance API. An authenticated human decider accepts or rejects;
> acceptance establishes the Position.

```
Human ─ tuple ─▶ cmd/themis-intake ─ local read-only ─▶ harness record plane
                        │
                        ▼ intake.Resolve → Evidence View
                        │ authenticated Governance API
                        ▼
                 Themis Governance ─▶ Decision Proposal ─▶ authenticated human ─▶ Position
```

Ownership boundary (locked):
- Harness owns execution records, evidence, provenance, verification
  reconstruction.
- `themis-intake` owns translating an already-existing harness
  execution into a Governance Proposal.
- Governance owns the proposal lifecycle and the authenticated decision.
- The human decider owns accept/reject.
- The harness does not initiate a Governance act. Governance does not
  read the harness record plane. Filesystem access to the record is
  confined to the Themis-owned intake CLI.

No second Position mechanism: the harness's output is the evidence
basis for the EXISTING Proposal → authenticated acceptance → Position
flow. The CLI boundary is `tuple → Resolve → Evidence View → Proposal`,
never `file path → read files → manufacture proposal`.
