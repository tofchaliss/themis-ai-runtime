# Design record: L11 ↔ Governance promotion boundary (D-R-*)

Locked decisions in the order the owner disposed them. Nothing here
changes D-L11-1..20 or D-L9-16.

## D-R-1 — Promotion is not a new Themis-owned state; the two existing doors remain the acts (LOCKED 2026-09-26, owner)

> Promotion is not a new Themis-owned state. The existing registration
> and reliance doors remain the authoritative governance/deployment
> acts; Row 12 adds provenance linking those acts to their
> authenticated human actor and the L11 evidence considered.

```
L11 evaluation ─ quality evidence ─▶ human governance decision
                                          ├─▶ Registration (catalog door): this reviewed revision is EXECUTABLE
                                          └─▶ Reliance (anchor door):     this deployment RELIES on that revision
```

- **Two decisions, kept distinct (D-L9-16):** registration = executable;
  reliance = relied upon. The L11 comparison package informs both; it
  never becomes the act. "Promotion" is not one atomic act.
- **Themis does not own runtime-method promotion.** Q-C-2 already has
  Themis record `method = skill name@version + composition_sha256` in
  the commission without validating runtime registries; making Themis
  the promotion owner would contradict that and force Themis to track
  runtime governance artifacts it was designed not to own.

| Runtime owns | Themis owns |
|---|---|
| executable method identity · registration · deployment reliance | commission · proposal admissibility · security Governance · final Position |

- **The actual gap is provenance of the human act:** the doors exist;
  what is missing is *who made the decision and which L11 evidence they
  relied upon*. Q-R-2 attaches a witness to the existing doors; it does
  not invent a new door or an approved-methods registry.

**Invariant carried into Q-R-2:** Themis may record and govern security
decisions about work performed under a method, but it does not become
the authority for whether a runtime method is executable or relied
upon. Runtime registration and deployment reliance remain runtime-owned
governance/deployment acts; L11 supplies evidence to those acts, and
Row 12 establishes their provenance.

## D-R-2 — The door witness: one content-addressed decision record, two-way bound (LOCKED 2026-09-26, owner)

> Every registration, withdrawal, and reliance door act is mechanically
> traceable to one immutable, content-addressed decision record whose
> target and hash are two-way bound to the governed entry; the record
> identifies the actor with no stronger authentication claim than the
> door actually possesses and references, rather than interprets, L11
> evidence.

1. **One decision record per door act:** `policies/decisions/<id>.json`
   — `id`, `kind ∈ {registration, withdrawal, reliance}`, `target`
   (exact governed identity incl. composition or artifact hash),
   `evidence[]` (criterion `K@v`, comparison package object id, record
   root identity), `rationale`, `decided_at`, `actor`. Provenance for an
   existing act, not a new state machine.
2. **Two-way binding, enforced by the loaders:** door entry →
   `decision_ref` → record → `target` → the same entry, hashes agreeing.
   Missing record, wrong target, or wrong hash → refuse to load → Open
   refuses. Catalog and anchors registry go to `version: 2` with
   `decision_ref` required on EVERY entry; existing entries receive
   records written from the ratifications already in the archive — no
   grandfathering.
3. **`actor` never overclaims:** `commit:<author>` = asserted identity
   (a git door cannot authenticate a human); `key:<KeyID>` =
   authenticated Themis principal, where a door is exercised under one.
   No signing mechanism introduced; a cryptographic author witness is
   its own future decision.
4. **L11 evidence referenced, never interpreted:** the record names
   packages; the runtime keeps them; cold replay from the object id. The
   loader validates the reference, not the semantic conclusion — the
   registration loader must never become an L11 evaluator (D-L11-18).
5. **Empty evidence is an explicit fact:** `evidence: []` with a
   rationale ("initial registration; no L11 comparison") is materially
   different from silent absence (D-L11-2 respected).

**Consequence (recorded):** the decision record is part of the
governed tree and participates in the same pinning discipline —
decision record → catalog/anchor entry → governed commit → anchor pin →
runtime deployment identity. A stale or modified record cannot quietly
accompany an otherwise valid entry. First version bumps land with
`rsys@6`.

**Distinction from a promotion registry (wording preserved):**
`policies/decisions` records WHY an existing runtime governance door
was exercised; it does not determine whether a method is executable or
relied upon. The catalog determines registration; the anchor
determines reliance; the record establishes provenance; L11 supplies
evidence; Themis does not become the runtime registry.

## D-R-3 — Commission validity is independent of runtime registration and reliance (LOCKED 2026-09-26, owner)

> Commission validity is independent of runtime method registration or
> deployment reliance. Runtime admission alone determines whether the
> commissioned method can execute; Themis never consults those runtime
> registries. A commission that cannot currently execute remains a
> valid authority record but produces no proposal-eligible execution
> until runtime admission succeeds.

A commission identifies what was AUTHORIZED; runtime admission
determines whether that method can actually EXECUTE (registered +
active; composition matches; in the anchor's `skills[]` — all enforced
fail-closed before any model runs). An invalid commissioned method
cannot silently produce a valid execution: refusal → no completed
execution → no resolvable tuple → no proposal. The commission stays a
legitimate Governance fact.

A commission created before registration is not invalid; when the
doors later admit the method, it becomes usable with no Themis
mutation — commissioning is a pure authority record, never a cached
snapshot of runtime registry state. What happens when a commissioned
method is later withdrawn stays under D-C-3 (historical commission,
current runtime admission, no retrospective rewriting; proactive
withdrawal is governance practice).

| Layer | Question | Owner |
|---|---|---|
| Commission | Was this method authorized for this Finding? | Themis |
| Admission | Can this method execute in this deployment? | Runtime |
| Proposal admission | Does this execution correspond to the commission and satisfy evidence requirements? | Themis |

## D-R-4 — No automation path from L11 to a door exists or is created (LOCKED 2026-09-26, owner)

> No L11 or ratchet execution path may write, select, register,
> withdraw, or prepare a runtime skill/deployment door. L11 produces
> evidence only. Runtime doors remain human-governed acts. Decision
> records may reference L11 evidence, but L11 artifacts never reference
> or mutate the doors.

```
L11 ─ evidence ─▶ Human ─ decision ─▶ Decision Record ─ provenance ─▶ existing runtime door
L11 ────X────▶ runtime door   (prohibited)
```

1. **The decision-record loader is read-only** (beside the catalog and
   registry loaders); a filesystem-writer AST wall covers `skills`,
   `deployment`, and the decision loader.
2. **L11 cannot reach the doors:** `ratchet` and `themis-ratchet` import
   only `confine` and `state`; the closed-importer test is extended to
   name `skills` and `deployment` as forbidden, so the first half of an
   automation path cannot appear by convenience import.
3. **Evidence reference is one-way:** decision → L11 package; never L11
   package → decision/catalog/anchor. A no-side-effect test: producing
   L11 evidence leaves the governed tree unchanged (D-L11-8 restated).
4. **No "preparation" automation:** no helper turns an L11 result into a
   draft catalog entry — that would be an automated promotion path with
   a human merely clicking commit. The human decision record is the
   stopping point.

## Closure (grill CLOSED 2026-09-26, D-R-1..4)

L11 informs human runtime governance decisions but cannot promote or
mutate runtime doors; runtime owns registration and reliance; Themis
owns commissioning and downstream security governance; neither becomes
the other's registry. With Rows 4, 11, and 12 decided, no 🔴 row
remains; implementation proceeds with W-M1.
