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

### D-L9-0 — Runtime position: L9 never executes at runtime (LOCKED 2026-09-08)
L9 is a pre-submission plane: define → resolve → validate → instantiate. Instantiation produces an ordinary governed envelope; L7 alone executes the resulting walk. There is no L9 runtime walker to constrain — second-orchestrator prevention by reachability, not discipline. The older architecture wording "L9 executes procedure" is superseded: execution responsibility was absorbed into shipped L7.

### D-L9-3 — Ownership: author ≠ governance ≠ machinery (LOCKED 2026-09-08, Q-L9-3)
Three axes, never blended: authorship (any permitted source — confers no trust or authority), catalog/registration/review (Themis governance — registering the reviewed composition hash is the owning act), machinery (harness — load/validate/resolve/instantiate, mechanical enforcement only, never content). An unregistered skill-shaped artifact is data and structurally unexecutable — presence is not authority (the repository-instructions trust model). **v1 prohibition:** repository-sourced or externally sourced Skills cannot become executable through discovery; they require a future explicit activation architecture + review gate (a Skill carries workflow + grant potential — more authority than repository prose). Steward ≠ authority: catalog metadata for accountability/review routing only; execution authority comes from the registered composition. **Review evidence v1:** registration in the governed catalog IS the evidence; the entry binds the exact composition hash + governance metadata; an asserted reviewed=true or author metadata is never evidence. No signature machinery in v1 — signatures solve actor attestation, not governance admission; reviewer attribution/separation-of-duties/external distribution would be a separate decision. Authority chain: author creates → governance reviews/registers → L9 mechanically validates/resolves → L7 executes; no step silently transfers ownership.

### D-L9-2 — Procedure: a composition concept, never a second runtime entity (LOCKED 2026-09-08, Q-L9-2)
A Procedure is the reusable method-of-work composed of one pinned workflow definition (deterministic governing skeleton, executed by δ) and one pinned procedure artifact (advisory technique, zero authority). A Skill is the atomic registration/review/versioning/instantiation unit packaging that Procedure with its resource envelope (context contract, grant template, spec template, input schema). Procedure identity = the identity of its pinned workflow + procedure-artifact composition; it is not independently registered, executable, or mutable — standalone Procedure registration would recreate the second-runtime lifecycle just eliminated (D-L9-0). Reuse: Procedure artifacts are immutable/versioned; Skills independently pin their identities — a later procedure@P2 modifies no existing Skill; a new Skill revision must explicitly choose P2. The authority split is total: the control half governs only sequence; the advisory half only suggests (prose inert at δ, proven in L7). Deferred to later questions, not assumed: phase binding of procedure sections (validation semantics), executable-logic boundary (Q-L9-4).

### D-L9-1 — A Skill is a governed, hash-pinned atomic composition (LOCKED 2026-09-08, Q-L9-1)
A Skill is a governed manifest pinning by SHA-256 one complete set of the artifact kinds L1–L7 already execute — workflow definition, workflow ceiling, context contract, grant template, spec template — plus a typed input schema and a **pinned procedure artifact reference** (not embedded: L1 owns instruction identity; embedding would mint a second instruction-identity mechanism inside L9), under a name and version. It is inert data, not an execution engine. **Atomic composition invariant:** the pinned artifacts are resolved as one governed set and cannot be independently mixed or substituted during instantiation (skill A's workflow with grant B is structurally impossible); the composition hash is the identity of the reviewed combination and lands in the L6 record. **Registry NOT pinned:** the capability vocabulary and authorization contracts stay L4-owned; a Skill references required capabilities through its grant template (skill grant ⊆ workflow ceiling ⊆ L4 registry, with L7 phase narrowing at execution) but cannot define or authorize capabilities. If capability-contract pinning is ever needed for reproducibility, that is a specific L4 contract identity, never L9 ownership of the registry. Skill = composition, not capability.

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
