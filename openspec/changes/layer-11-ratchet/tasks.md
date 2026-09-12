# Tasks: Layer 11 — Ratchet

Constitution: design.md §2 (D-L11-1..20, all LOCKED 2026-09-12).
Implementation rule (D-L11-20, strict): anything not in the locked
constitution → stop at the boundary; classify as implementation
detail / residual / genuine gap; reopen the grill only for the last.
The D-L11-20 §2 artifact inventory is the architectural whitelist.

## Gate 0 (owner) — PENDING

- [ ] Owner Gate 0 judgment on this plan.
- Knowledge rule already DECIDED at D-L11-20 lock: the v1 proof
  slice includes one knowledge-family egress path IF the existing
  Themis ingestion door is available; otherwise explicitly
  unproved/residual — no substitute L11 knowledge authority.

## The four flagged implementation choices (D-L11-20 §4)

Settle each at its milestone, ordinary review, no new architecture:
- [ ] (a) Attribution representation: existing task-envelope field
      vs narrow additive L6 amendment (M4).
- [ ] (b) Fixture/golden physical housing: registry-plane files vs
      L6 objects by size (M1).
- [ ] (c) v1 comparator set: ≥1 concrete family for the proof slice
      (proposed: numeric-score-delta over benchmark/L10 facts) (M2).
- [ ] (d) Invocation surface: CLI subcommand vs service endpoint —
      Class-3 review either way (M4).

## M0 — Scaffold deletion + docs

- [ ] DELETE src/harness/ratchet/{candidates,evaluations,feedback,
      promotion,regression} (D-L11-20 §6: structure follows
      decisions; feedback/ and promotion/ name forbidden
      subsystems).
- [ ] Status/code-map docs note the frozen constitution.

## M1 — Registries (D-L11-6/9/13)

- [ ] Criterion registry: format + fail-closed read-only loader
      (verification/registry.go pattern: append-only rules,
      CheckAppendOnly, exact ParseRef, two-way identity, withdrawn
      semantics). NO write API (AST wall).
- [ ] Criterion schema loader (contract.go pattern): closed schema —
      identity, families, evidence selectors (closed vocabulary,
      established facts only, NO L11 outputs), comparator binding
      (name@version), pinned config (by value, hashed), Δ shape,
      optional ordering/region (per-field direction/thresholds,
      D-L11-7 taxonomy), baseline constraints
      ({requires-current-active}), provenance requirements. Refuse:
      continuation language, security predicates, semantic region
      names (D-L11-6 §5, D-L11-18 walls).
- [ ] Regression-set registry: exact K@v member pins; supplied-S@v
      only (D-L11-9 Am. 2 — no latest resolution).
- [ ] Registry proofs: loader refusals (doctored artifacts),
      append-only violations, two-way identity, unregistered=data.

## M2 — Comparator + evaluation core (D-L11-4/5/7/17)

- [ ] Comparator interface: pure, deterministic, environment-free;
      canonical Δ serialization as registered semantics; semantic
      change = new version (D-L11-17 §2iv).
- [ ] v1 comparator (choice (c)).
- [ ] Selector resolution over committed L2/L6/L10/benchmark
      records; refusal on unresolvable (closed reason classes,
      D-L11-16 §3).
- [ ] Baseline observation: door-registry-hash-grounded admission
      observation; claim/observation mismatch recorded, never
      silently substituted; admission = production precondition
      (refusal, no package).
- [ ] Ordering derivations: better-under-K / equal / worse /
      incomparable — stateless, never stored; incomparability per K
      declaration only.
- [ ] Proofs: determinism (bit-identical reruns), direction
      symmetry, refusal neutrality, no-ordering ⇒ no better-claim.

## M3 — Packages, persistence, reconstruction (D-L11-4/9/11/17)

- [ ] Comparative-evidence package: full conditioning tuple
      (D-L11-17 §1); content-hash identity; L6 StoreObject
      persistence; refusal facts and discrepancy facts as instance
      artifacts.
- [ ] Regression-evidence package: complete-under-S only (one
      constituent per member; no partial packages, no coverage
      fields); resistant-under-S as stateless derivation.
- [ ] Reconstruct (L10 pattern): CONFIRMED /
      UNREPRODUCIBLE-FOR-MISSING-INPUTS / DISCREPANCY; never
      repairs; cold — package + L6 + registries only.
- [ ] Proofs: cold reconstruction (delete-cache test), tamper
      detection at read boundary, no-discard-after-accepted-
      invocation (D-L11-14 §3).

## M4 — Candidate, Evaluation Plan, invocation (D-L11-3/10/14)

- [ ] Candidate schema: closed contents (D-L11-3), content-hash
      identity, no lifecycle state, lineage as hash claims;
      families incl. criterion-revision + regression-set-revision
      (D-L11-19 §2).
- [ ] Evaluation Plan schema: declarative allowed-list only
      (D-L11-10); inert; never authority; plan-conformance =
      record-to-plan matching.
- [ ] Attribution (choice (a)): run → plan reference as provenance;
      must not alter L7/L10/L4/L5 behavior.
- [ ] Invocation surface (choice (d)): synchronous-only entry;
      exact pins required (K@v, S@v, B identity); no defaults.
- [ ] Proofs: plan cannot execute; candidate confers nothing;
      machinery authors no candidates.

## M5 — Walls, guards, registers (D-L11-11/12/14/15/16/18)

- [ ] AST walls: no registry write APIs; no
      timers/tickers/watchers/pollers/queues/background goroutines;
      L11 package imports audited (terminal output — no path from
      package production to any initiation API).
- [ ] Negative-state guards: no status columns, no caches with
      authority, no champion/current pointers, no feedback
      artifacts, no outcome enum, no L10 tokens as L11 states, no
      security predicates in schemas.
- [ ] API-closure allowlist extensions (deliberate, commented).
- [ ] Registers: proposition boundary (six claims refused at every
      surface), consumption classes (P1→K2→P2 refused; Δ not a
      selection input), recursion (L11-about-L11 criterion
      unregistrable), adversarial register (weight-flattering,
      baseline-flattering, selector cherry-picking → visible;
      corpus self-modification impossible).

## M6 — Proof slice (live) (D-L11-20 §3)

- [ ] End-to-end: author candidate (model, governed task) →
      registration via door (*.proposed.* → owner act) → governed
      evaluation runs (ordinary tasks, plan-attributed) → comparison
      request (exact pins) → package → derived views → human reads
      evidence at door → promotion act → regression admission.
- [ ] Knowledge-family egress path per the Gate 0 rule (or recorded
      residual — no substitute door).
- [ ] Live model proof (ollama) for the authored-candidate arc.

## M7 — Close reviews + archive

- [ ] Three Class-3 reviews (architecture, security,
      test/verification) against the locked constitution;
      remediation; re-runs.
- [ ] Traceability completed (traceability.md).
- [ ] Owner closure judgment; Governance activation of any
      *.proposed.* registrations; push (owner-approved,
      THEMIS_PUSH_APPROVED=1); archive.

## Residuals (recorded, never silently promoted)

Pre-L11 standing: L5 process-exec amendment; live telemetry grill;
L6 GC-anchoring ADG; L9 loader hardening; C3 attestation; L8
delegation; equivalent-mutant record.
L11-minted (each = fresh architecture decision if ever wanted):
meta-comparison; Δ in runtime selection; multi-baseline criteria;
benchmark-plane physical unification; knowledge-family proof if the
Themis door is unavailable.

## Forbidden components (D-L11-20 §6 — implementation tripwires)

feedback subsystem; candidate registry/status store; champion
registry; L11 event stream; authoritative cache;
scheduler/timer/watcher/queue; L11 executor; baseline manager;
version-selection authority; coverage quantification; outcome enum;
security predicates; meta-comparison; self-evaluation; optimization
objective.
