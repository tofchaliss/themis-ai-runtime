# Design: Layer 9 — Skills & Procedures

Grill OPEN 2026-09-07. §2 collects folded decisions as questions close; §3 is the question list.

## 0. Position in the flow

L9 sits *before* submission: it is authoring-and-instantiation machinery. The L7 loop stays the sole executor; L9 produces what the loop consumes. Everything a skill "does" at runtime must already be expressible as the seven governed envelope references — if it isn't, that is a gap in a lower layer, not a license for L9 side channels.

## 1. Hard invariants (inherited, not grillable)

- Model output advisory; deterministic controls enforce (Day-0).
- A skill may instantiate/narrow governed ceilings, never define them (cross-layer principle: L2 Plan⊆Contract · L4 Grant⊆Ceiling⊆Registry · L5 Spec⊆Ceiling · L7 workflow⊆ceiling⊆registry).
- Skills are procedures, never authoritative business truth (ADR-009).
- External content is data; instruction identity and authority are L1-owned; untrusted content reaches the model fenced through L2.
- Governed artifacts: versioned, DisallowUnknownFields, trailing-content refusal, SHA-256 into the record, fail-closed loaders.
- Nothing is defaulted; every absence is a typed, named refusal.
- Approval vocabulary stays reserved fail-closed (Q-L7-9) until its own grill.
- A recorded deterministic state transition identifies the exact governing rule, the selected branch, and the actual causal event — the verifier never guesses (owner principle, L7 final acceptance 2026-09-08).

## 2. Locked decisions (fold target)

*(empty until questions close)*

## 3. Open questions — Q-L9-1..16 (owner list 2026-09-08, absorbing the draft 10)

**Boundary (owner-proposed, under grill):** L9 defines, resolves, and validates governed Skills and Procedures. It provides reusable domain execution patterns to L7 but owns no authorization, security truth, workflow state, or AI authority. Claude counter pending owner ruling: L9 never executes at runtime — instantiation produces an ordinary governed envelope; L7 alone executes (second-orchestrator prevention by reachability, not discipline).

**Starting principle (owner, locked):** a Skill/Procedure may describe or perform a governed method of work, but it cannot grant authority, create security truth, alter workflow semantics, or bypass the deterministic controls of L1–L7.

1. **Q-L9-1 — What is a Skill?** Instructions, a reusable procedure, a capability bundle, a workflow fragment, or something else?
2. **Q-L9-2 — What is a Procedure?** How does it differ from a Skill?
3. **Q-L9-3 — Who owns a Skill?** Harness, Themis domain, a repository, or an external provider?
4. **Q-L9-4 — Can a Skill contain executable logic?** If yes, where does it execute; if no, what exactly does a Skill produce?
5. **Q-L9-5 — Can a Skill select tools?** L4 already owns authorization.
6. **Q-L9-6 — Can a Skill change the workflow?** Owner initial position: no — L7 remains sole owner of workflow transition.
7. **Q-L9-7 — Can the model invent a Skill/Procedure?** Hard distinction: model-generated reasoning vs governed reusable procedure vs executable capability.
8. **Q-L9-8 — What is the authority of Skill output?** Cannot be security truth (constitution).
9. **Q-L9-9 — How is a Skill versioned and pinned?**
10. **Q-L9-10 — What happens when a Skill changes while a task is running?**
11. **Q-L9-11 — Can Skills call other Skills?** Recursion, depth, authority propagation, cycle controls.
12. **Q-L9-12 — How does L9 interact with L7?** The most important architectural question.
13. **Q-L9-13 — The input contract.** Typed caller inputs, validation, injection posture, caps, no defaulting.
14. **Q-L9-14 — Verification steps.** Per-skill Verification (Build=PASS etc.) — phases, tools, or an L10 seam; who owns the verdict.
15. **Q-L9-15 — Approval requirements.** Skills declare approval points; no approval channel exists — v1 posture.
16. **Q-L9-16 — Proof gate.** Registers + the P0 slice skill and what its live proof demonstrates.

## 4. Test plan

*(drafted after the fold)*
