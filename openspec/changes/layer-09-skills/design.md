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

## 2. Locked decisions (fold target)

*(empty until questions close)*

## 3. Open questions — Q-L9-1..10

1. **Q-L9-1 — What is a skill, structurally?** New executable artifact kind, or a governed *bundle* pinning existing artifact kinds (workflow def + context contract + grant template + spec template + input schema + procedure text) by hash?
2. **Q-L9-2 — The instruction channel.** Skill procedure text is instructions. Which channel delivers it — a registered L1 scope (closing the Q-L1-1 skill-source IOU) with an activation contract, per-phase L2 slots, or both? What authority class can skill text ever earn?
3. **Q-L9-3 — Catalog and registration.** Where do skills live, who may register one, what makes it *reviewed* (vs merely present), and how does an unreviewed skill fail to execute structurally?
4. **Q-L9-4 — Instantiation.** How does {skill, caller inputs} become an envelope? What may the caller supply (inputs only?) and what is skill-fixed? Fourth application of instantiate-never-define.
5. **Q-L9-5 — The input contract.** Typed inputs, validation, injection posture (inputs are external-untrusted forever?), size caps, no defaulting.
6. **Q-L9-6 — Verification steps.** The architecture's per-skill Verification (Build=PASS etc.) — deterministic checks as workflow phases, as tools, or as an L10 seam? Who owns the verdict?
7. **Q-L9-7 — Approval requirements.** Skills declare approval points, but no approval channel exists. v1 posture: approval-requiring skills refuse to load, or degrade to fail-terminal?
8. **Q-L9-8 — Versioning and evolution.** Version identity in the record, upgrade semantics, retry across skill versions, deprecation without deletion.
9. **Q-L9-9 — Failure conditions.** Skill-declared failure conditions vs the L7 failure taxonomy — mapping or duplication?
10. **Q-L9-10 — Proof gate.** Registers + the P0 slice: which of the six architecture skills ships first, and what does its live proof have to demonstrate?

## 4. Test plan

*(drafted after the fold)*
