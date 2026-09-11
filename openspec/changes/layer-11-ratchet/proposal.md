# Proposal: Layer 11 — Ratchet

Status: GRILL OPEN 2026-09-11. Implementation does not start before
Gate 0. The last layer of the locked Phase-C sequence.

## Why now

L10 shipped and archived; the sequence ends L11: "generalize
compare/gate/variants — no second evaluation subsystem" (locked plan).
Three forces converge:

1. **The raw material is proven and waiting.** The benchmark plane
   already runs the full loop the Ratchet generalizes: run → evaluate →
   validate → report → compare → gate, prompt variants (A/B), and — as
   of this session — digest-bound verdict artifacts whose admission is
   enforced at consumption (the gate/router work). L10 deliberately did
   not absorb any of it; that line was drawn for L11.
2. **Two residuals were assigned here by name.** D-L9-16 deferred the
   formal proposal/candidate lifecycle to "the future L11 grill," and
   the evaluation-vs-operational purpose attribution gap was recorded
   as L11/L6.
3. **The inherited constitution is already strict.** D-L9-16 locked the
   two-gates-never-collapsed rule, no auto-promotion (no harness path
   from evaluation results to registration/withdrawal/selection),
   human-only promotion, evaluation as ordinary governed tasks with no
   special authority, and no reduced review for AI-authored candidates.
   L11's job is to give that constitution its machinery — not to
   renegotiate it.

## What the Ratchet is (architecture baseline)

"What validated improvements should be incorporated so future work is
better and regression-resistant?" — a cross-cutting feedback layer:
knowledge improvement, skill improvement, evaluation improvement,
instruction improvement, feeding future tasks. Purpose: make the system
progressively better while preventing unvalidated AI behavior from
becoming permanent truth (00-p0-architecture-v2 §14).

## What this change delivers (subject to the grill)

- The candidate/evidence/promotion vocabulary: what may be proposed,
  how comparative evidence is produced under governed evaluation, and
  what each promotion act reuses (the existing governance doors — skill
  catalog, contract registry, instruction revision — never a new one).
- The regression discipline generalized from the benchmark gate:
  scores may only ratchet up; failures become regression material;
  admission is enforced at consumption (the verdict pattern).
- The purpose-attribution answer (evaluation vs operational) for the
  record plane.
- A reconciliation of the benchmark plane: what generalizes, what
  stays, single-home — no second evaluation subsystem, no fork.

## Constraints inherited (not grillable)

Model output advisory; the ratchet informs, humans promote (D-L9-16);
author ≠ governance ≠ machinery (D-L9-3/8); registration gates precede
any execution (no evaluation backdoor); PASS/score ≠ better ≠ promoted
(D-L10-15 discipline extended); four-proposition separation at every
boundary; fail closed; L6 owns the record; Themis owns meaning.
