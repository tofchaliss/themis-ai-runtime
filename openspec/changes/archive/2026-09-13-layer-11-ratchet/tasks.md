# Tasks: Layer 11 — Ratchet

Constitution: design.md §2 (D-L11-1..20, all LOCKED 2026-09-12).
Implementation rule (D-L11-20, strict): anything not in the locked
constitution → stop at the boundary; classify as implementation
detail / residual / genuine gap; reopen the grill only for the last.
The D-L11-20 §2 artifact inventory is the architectural whitelist.

## Gate 0 (owner) — **PASS 2026-09-12**

- [x] Owner Gate 0 judgment: PASS. Implementation authorized from
      M0; no further architectural grill before M0. All four
      flagged choices accepted as implementation choices within the
      constitution. No push implied — local-only/push-after-approval
      boundary unchanged.
- Knowledge rule confirmed at Gate 0: include the knowledge egress
  path when the existing Themis door is available; if unavailable,
  record the proof limitation explicitly — never a substitute
  Harness-side knowledge authority to make M6 pass.
- **Gate 0 execution constraint (carried into M0–M7):** no
  milestone may introduce a component whose architectural role is
  absent from the locked L11 artifact inventory or the
  already-proven underlying mechanisms. If such a component appears
  necessary, implementation STOPS and the item is classified
  (implementation detail / residual / genuine gap) before
  proceeding.
- Owner framing: the question is no longer "what should L11 be?"
  but "does the implementation actually conform to what we have
  already decided L11 is?"

## The four flagged implementation choices (D-L11-20 §4)

Settle each at its milestone, ordinary review, no new architecture:
- [x] (a) Attribution representation: SETTLED v1 as package-level
      plan_ref (ComparisonPackage.PlanRef, object-id-validated) +
      plan-conformance matching; run-envelope attribution awaits
      run-record integration (recorded in plan.go conformance note).
      No L6 amendment needed.
- [x] (b) Fixture/golden housing: SETTLED as registry-plane files,
      registry-relative, hash-pinned (policies/ratchet/ layout).
- [x] (c) v1 comparator: SETTLED — numeric-score-delta@1 (registered
      table, comparator.go).
- [x] (d) Invocation surface: SETTLED — cmd/themis-ratchet CLI,
      synchronous-only, compiled-binary contract tests.

## M0 — Scaffold deletion + docs

- [x] DELETE src/harness/ratchet/{candidates,evaluations,feedback,
      promotion,regression} (D-L11-20 §6: structure follows
      decisions; feedback/ and promotion/ name forbidden
      subsystems).
- [x] Status/code-map docs note the frozen constitution.

## M1 — Registries (D-L11-6/9/13)

- [x] Criterion registry: format + fail-closed read-only loader
      (verification/registry.go pattern: append-only rules,
      CheckAppendOnly, exact ParseRef, two-way identity, withdrawn
      semantics). NO write API (AST wall).
- [x] Criterion schema loader (contract.go pattern): closed schema —
      identity, families, evidence selectors (closed vocabulary,
      established facts only, NO L11 outputs), comparator binding
      (name@version), pinned config (by value, hashed), Δ shape,
      optional ordering/region (per-field direction/thresholds,
      D-L11-7 taxonomy), baseline constraints
      ({requires-current-active}), provenance requirements. Refuse:
      continuation language, security predicates, semantic region
      names (D-L11-6 §5, D-L11-18 walls).
- [x] Regression-set registry: exact K@v member pins; supplied-S@v
      only (D-L11-9 Am. 2 — no latest resolution).
- [x] Registry proofs: loader refusals (doctored artifacts),
      append-only violations, two-way identity, unregistered=data.

## M2 — Comparator + evaluation core (D-L11-4/5/7/17)

- [x] Comparator interface: pure, deterministic, environment-free;
      canonical Δ serialization as registered semantics; semantic
      change = new version (D-L11-17 §2iv).
- [x] v1 comparator (choice (c)).
- [x] Selector resolution over committed L2/L6/L10/benchmark
      records; refusal on unresolvable (closed reason classes,
      D-L11-16 §3).
- [x] Baseline observation: door-registry-hash-grounded admission
      observation; claim/observation mismatch recorded, never
      silently substituted; admission = production precondition
      (refusal, no package).
- [x] Ordering derivations: better-under-K / equal / worse /
      incomparable — stateless, never stored; incomparability per K
      declaration only.
- [x] Proofs: determinism (bit-identical reruns), direction
      symmetry, refusal neutrality, no-ordering ⇒ no better-claim.

## M3 — Packages, persistence, reconstruction (D-L11-4/9/11/17)

- [x] Comparative-evidence package: full conditioning tuple
      (D-L11-17 §1); content-hash identity; L6 StoreObject
      persistence; refusal facts and discrepancy facts as instance
      artifacts.
- [x] Regression-evidence package: complete-under-S only (one
      constituent per member; no partial packages, no coverage
      fields); resistant-under-S as stateless derivation.
- [x] Reconstruct (L10 pattern): CONFIRMED /
      UNREPRODUCIBLE-FOR-MISSING-INPUTS / DISCREPANCY; never
      repairs; cold — package + L6 + registries only.
- [x] Proofs: cold reconstruction (delete-cache test), tamper
      detection at read boundary, no-discard-after-accepted-
      invocation (D-L11-14 §3).

## M4 — Candidate, Evaluation Plan, invocation (D-L11-3/10/14)

- [x] Candidate schema: closed contents (D-L11-3), content-hash
      identity, no lifecycle state, lineage as hash claims;
      families incl. criterion-revision + regression-set-revision
      (D-L11-19 §2).
- [x] Evaluation Plan schema: declarative allowed-list only
      (D-L11-10); inert; never authority; plan-conformance =
      record-to-plan matching.
- [x] Attribution (choice (a)): run → plan reference as provenance;
      must not alter L7/L10/L4/L5 behavior.
- [x] Invocation surface (choice (d)): synchronous-only entry;
      exact pins required (K@v, S@v, B identity); no defaults.
- [x] Proofs: plan cannot execute; candidate confers nothing;
      machinery authors no candidates.

## M5 — Walls, guards, registers (D-L11-11/12/14/15/16/18)

- [x] AST walls: no registry write APIs; no
      timers/tickers/watchers/pollers/queues/background goroutines;
      L11 package imports audited (terminal output — no path from
      package production to any initiation API).
- [x] Negative-state guards: no status columns, no caches with
      authority, no champion/current pointers, no feedback
      artifacts, no outcome enum, no L10 tokens as L11 states, no
      security predicates in schemas.
- [x] API-closure allowlist extensions (deliberate, commented).
- [x] Registers: proposition boundary (six claims refused at every
      surface), consumption classes (P1→K2→P2 refused; Δ not a
      selection input), recursion (L11-about-L11 criterion
      unregistrable), adversarial register (weight-flattering,
      baseline-flattering, selector cherry-picking → visible;
      corpus self-modification impossible).

## M6 — Proof slice (live) (D-L11-20 §3)

- [x] End-to-end: author candidate (model, governed task) →
      registration via door (*.proposed.* → owner act) → governed
      evaluation runs (ordinary tasks, plan-attributed) → comparison
      request (exact pins) → package → derived views → human reads
      evidence at door → promotion act → regression admission.
- [x] Knowledge-family egress path per the Gate 0 rule (or recorded
      residual — no substitute door).
- [x] Live model proof (ollama) for the authored-candidate arc.

## M7 — Close reviews + archive

- [x] Three Class-3 reviews run 2026-09-13 (architecture, security,
      test/verification) against b16fc87..fca10e0. Convergent core
      finding: invocation boundary trusted its caller (sec C-1 =
      arch H-1; sec H-1 = arch H-2). ALL CRITICAL/HIGH remediated:
      door resolution (ObserveAdmission/ReverifyAdmission), evidence
      grounding (GroundFacts + params application), durable
      unregistered/withdrawn refusals, registry binding + pin,
      missing-inputs durability, set hash binding, strict CLI
      inputs, reconstruction parity, ~19 mutants killed, wall
      upgrades, CLI contract suite, *.proposed.* handoff arc.
      Recorded classifications awaiting owner ratification:
      no-discard scope, consumption-pin-instead-of-persisted-prior,
      sensitivity residual (see traceability.md).
- [x] Traceability completed (traceability.md, incl. remediation
      record).
- [x] Owner closure judgment 2026-09-13: CLOSED, all three states
      PASS (operationally-proven machine-local); three
      classifications RATIFIED; findings = implementation
      corrections, no decision reopened (design.md §5).
- [x] Governance activation: approved for Governance action,
      executed as the owner-directed act (separate commit);
      bench-score-delta@1 + core-regression@1 ACTIVE.
- [x] Push approved (no history rewrite); archive approved with
      activation recorded separately.

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
