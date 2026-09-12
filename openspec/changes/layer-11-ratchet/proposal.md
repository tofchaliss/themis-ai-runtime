# Proposal: Layer 11 — Ratchet

Status: **GRILL CLOSED 2026-09-12 — D-L11-1..20 ALL LOCKED (20/20).**
Awaiting Gate 0; implementation does not start before it. The last
layer of the locked Phase-C sequence. The authoritative constitution
is design.md §2; tasks.md holds the milestone plan; the artifact
inventory in D-L11-20 §2 is the implementation whitelist.

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

## What this change delivers (as locked by the grill)

- **Two Governance registries** (criterion K, regression set S) in
  L11 format with fail-closed read-only loaders — Governance owns
  every registration act (D-L11-6/9).
- **The comparative proposition machinery:** conditioned comparative
  fact as the only L11 atom; deterministic registered comparators
  (code identity, option A); packages with complete conditioning
  tuples; stateless derivations (better-under-K, resistant-under-S)
  never stored (D-L11-4/7/17).
- **Baseline/admission discipline:** doors own baselines; admission
  observations, never L11 admission facts; refusal (not weaker
  packages) when preconditions fail (D-L11-5/16).
- **Candidate + Evaluation Plan** as inert content-addressed instance
  artifacts; describe-and-consume execution model — L11 runs,
  selects, and schedules nothing (D-L11-3/10/11).
- **Selection ≠ promotion** formalized (five conditions; the router
  legitimized as the Class-2 instance) and the six-class
  consumption-of-evidence rule; L11 output terminal (D-L11-8/15).
- **Automation boundary:** complete-never-initiate; no
  self-continuation; no candidate generation by machinery; no-discard
  after accepted invocation (D-L11-14).
- **No feedback subsystem; corpus = registered criteria + pinned
  parameters; no outcome vocabulary; no security meaning; recursion
  terminates in Governance — no L12 (D-L11-12/13/16/18/19).

## Constraints inherited (not grillable)

Model output advisory; the ratchet informs, humans promote (D-L9-16);
author ≠ governance ≠ machinery (D-L9-3/8); registration gates precede
any execution (no evaluation backdoor); PASS/score ≠ better ≠ promoted
(D-L10-15 discipline extended); four-proposition separation at every
boundary; fail closed; L6 owns the record; Themis owns meaning.
