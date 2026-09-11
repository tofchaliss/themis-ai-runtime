# L6 Amendment: L10 verification seam (2026-09-11)

Additive change to the archived L6 layer, authorized by D-L10-6 stage 5
and Gate 0; full context in the L7 amendment record
(../../2026-09-07-layer-07-orchestration/amendments/
L10-verification-seam/AMENDMENT.md, item 8).

1. Event class `l10-verification` (EvVerification) added to the closed
   sink vocabulary — the committed L10 evaluation instance {contract
   token, outcome, record object id}. A new semantic gets its own
   class rather than a third meaning on an existing one (the L9
   event-class lesson). L6 validates the envelope, never the content;
   the class is caller-appendable (not primitive-only), written by the
   L7 loop under record-before-event.

Recorded ADG follow-up (not part of this amendment): audit-scope
objects (reconstruction-discrepancy artifacts stored via the root
object store outside task streams) are unanchored under the FUTURE
reachability GC; an anchoring decision must precede any GC
implementation. No deleter exists in L6 today.

Invariants not weakened: object classes unchanged; append-only,
chain, and manifest disciplines untouched; the L6 suite re-runs green
(close evidence, L10 tasks.md).
