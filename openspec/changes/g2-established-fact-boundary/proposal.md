# Proposal: G2 — The Established-Fact Boundary

Status: GRILL OPEN 2026-09-13. Source: L1–L11 integration audit
synthesis G2 (owner-classified genuine cross-layer gap,
grill-second). The D1/D4 defect remediation is BLOCKED on this
grill by owner order: G2 defines who can establish a fact;
implementation then enforces that boundary — never the reverse.

## The gap (owner formulation)

L11 has a vocabulary of fact references, but the architecture has
not established a sufficiently authoritative mapping between an L6
object and the fact class L11 may treat as established evidence.
Durable storage does not make an established fact; an ObjectID
proves identity of bytes, not epistemic authority of bytes. The
laundering path: model assertion → L6 object → criterion selector
→ GroundFacts → Δ.

PROHIBITED solution shape (owner): an `established_fact: true`
boolean (or any per-object classification field) on L6 objects —
that merely moves the trust problem. Authority must come from the
mechanism that mints or establishes the fact, with provenance
sufficient for L11 to verify eligibility.

## Grill questions

| Q | Question |
|---|---|
| Q-G2-1 | What establishes a fact — the principle? |
| Q-G2-2 | The fact taxonomy (executor-minted, model-authored, L10 outcomes, L6 execution records, external, derived, L11-produced) and each kind's establishing mechanism? |
| Q-G2-3 | Where does the fact-kind → establishing-mechanism mapping live, and who owns it? |
| Q-G2-4 | How does L11 mechanically verify eligibility at grounding (schema consequences)? |
| Q-G2-5 | How does cold reconstruction re-verify establishment? |
| Q-G2-6 | The prohibited shapes, enumerated? |

## Constraints inherited (not grillable)

Model output advisory never authority; L6 is the sole record plane
and validates envelopes never content; L11 consumes only
established facts and its output is terminal; refusal-not-weaker-
package; parameters never programs; no per-object trust booleans
(owner, above); R2 as ratified (event-carried classification is
the direction the ADG must restore); Day-0.
