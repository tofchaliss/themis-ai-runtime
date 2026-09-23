# L6 Amendment: the `l8-delegation` event class (L8 M3, 2026-09-23)

Additive, hash-changing change to the archived L6 layer, authorized by
L8 design D-L8-8/17/19 (owner LOCK 2026-09-22) and Gate 0 §5.2.

1. Event class `l8-delegation` (`EvL8Delegation`) added to the closed
   sink vocabulary — the witness that one delegated reasoning
   execution was ESTABLISHED within the parent task. A materially
   different semantic from `model-turn` and `l2-delivery`, so its own
   class even though reuse would be mechanically cheaper (the L9
   event-class lesson). Caller-appendable (written by the L8 seam
   under record-before-effect), not primitive-only. L6 validates the
   envelope, never the content.
2. Body shape and closure are owned by `subagents/delegation`
   (`Event`, `Validate`, `Encode`, `Decode`, `SortEvidence`): identity
   = (task_id, seq), no delegation id (C-L8-20); closed outcome set
   {completed, provider-error, output-over-bound,
   model-identity-mismatch}; evidence references canonical — strictly
   ascending seq, no duplicates (C-L8-6), every seq < parent_call_seq
   (C-L8-8); template bytes referenced (D-L8-17); composition and
   output are `evidence-payload` objects — no new object class. The
   encoding is a pure function of its inputs (Register C:
   `TestEventBodyReproducible`).
3. `ConstitutionHash()` changes. Consequence recorded: every anchor
   pinning `constitution.state` (the proposed `rsys2.json`) is stale
   until the `rsys@4` Governance act re-pins it (L8 M6). No test pins
   the literal hash; `TestL8DelegationEventClass` pins the vocabulary
   by count and membership (grew by exactly one).

Invariants not weakened: object classes unchanged; append-only,
chain, manifest, and recovery disciplines untouched; the L6 suite
re-runs green.
