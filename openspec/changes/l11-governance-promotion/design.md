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
