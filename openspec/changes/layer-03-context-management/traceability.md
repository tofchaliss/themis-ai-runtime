# Layer 3 Traceability — grill invariants → tests

All tests in `src/harness/context`. Invariants from `design.md` §3 (owner-locked closures).

| Invariant | Evidence |
| --- | --- |
| Deterministic policy execution, zero runtime judgment (Q-L3-1) | `TestManageDeterministic` (deep-equal across runs); policy-declared rank keys only (`TestManagementPolicyFailsClosed/illegal rank key` — "relevance" rejected) |
| Type-system boundary: rank reads ItemRef only (Q-L3-1) | Structural: `rankForDrop` signature consumes `[]ItemRef`; `ItemRef.Size` is the only size input; no evidence bytes reachable |
| Capacity is never authority (Q-L3-2) | `TestManageDropOrderContractGate` (required, undroppable-optional, withheld, unknown slots all rejected in drop order); `TestContractDroppableValidation` (required+droppable and withheld+droppable are contract contradictions) |
| Four-way governed-slot disposition, fail-closed default | Structural: `Slot.Droppable` JSON-absent = false = undroppable; validation forbids contradictory dispositions |
| Unmeetable budget fails closed — no silent degradation | `TestManageBudgetFailsClosed` (nil managed, nil trace, ErrBudget) |
| `omitted_for_capacity` fifth state; state-only model marker; counts trace-only (Q-L3-3) | `TestManagePressureDropsAndMarker` (partial: delivered + flag marker, no counts in payload), `TestManageFullSlotOmission` (full slot renders the fifth state) |
| No-pressure passthrough byte-identical (Q-L3-9) | `TestManageNoPressurePassthrough` (items deep-equal + composed view identical to un-managed composition) |
| Drop order + within-slot ranking deterministic | `TestManagePressureDropsAndMarker` (size_asc drops the large item first, exactly one drop decision), `TestManageDeterministic` |
| Duplicate-hash occurrence accounting | `TestManageFullSlotOmission` (two byte-identical items both dropped); `TestManagePhantomDropBudget` (security HIGH-1 regression: dedup+duplicates cannot falsify the budget; refs bijective with delivered items) |
| Dedup removes delivery redundancy, never provenance multiplicity (Q-L3-8) | `TestManageDedup` (within-class collapse retains every supplier's source/kind/version in `Collapsed`; cross-class byte-identical never collapses) |
| Rank ≠ render (Q-L3-4) | Structural: Compose renders contract slot order unchanged; `TestManageNoPressurePassthrough` composed-view identity |
| No truncation | Structural: Manage moves whole items only; no byte-slicing code path exists |
| ManagementPolicy artifact posture (Q-L3-7) | `TestManagementPolicyFailsClosed` (8 fail-closed cases); `PolicyHash` + estimator recorded in trace (`TestManageNoPressurePassthrough`) |
| Compressor seam only, no activation possible (Q-L3-6) | Structural: `CompressorRegistration` has no vocabulary field — activation impossible by construction |
| Probabilistic selection excluded (Q-L3-5) | Structural: no embedding/reranker/semantic code exists; rank keys are a closed legal set |
| Retrieval as classified L2 sources (M3) | `TestSearchSource` + `TestSearchSymlinkEscapeRefused` (security HIGH-2 regression: file symlinks hard-refuse) + `TestSearchOverCapRefuses` (MEDIUM-1: no silent truncation — over-cap refuses deterministically) |
| Live pressure proof (Q-L3-9) | `TestLivePressureProof` — PASS 2026-09-06: drops applied (111/300 tokens), omission marker delivered, live model cited surviving evidence |
