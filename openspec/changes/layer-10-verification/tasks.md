# Tasks: Layer 10 — Verification & Observability

Grill closed 2026-09-11 (Q-L10-1..21 all CLOSED or RESIDUAL; design.md §3 is
the record, §2 the folded decisions D-L10-1..18 + D-L10-1a, §4 the proposed
proof registers). Execution starts only after owner Gate 0. Three-state
verdicts per milestone: architecture-conformant · test-evidenced ·
operationally-proven. An editing command or commit message is never
evidence. The L9 mutation-testing lesson applies from day one.

## 0. Gate

- [x] Grill session held; boundary + all questions answered and recorded
      (owner-led sequence Q-L10-1..18 + merged Q-L10-19..21 + carved
      Q-L10-10a; Q-L10-15 → residual)
- [x] Decisions folded into design.md §2 (D-L10-1..18, D-L10-1a); proof
      registers proposed (§4)
- [ ] Gate 0: design + proof plan accepted by owner

## 1. L10-M1 — Contract registry + schema (Class 3)

- [ ] Verification Contract artifact + fail-closed loader: closed schema,
      DisallowUnknownFields, trailing-content refusal, exact name@version,
      capability pin, evidence input spec (kinds + constraints), pinned
      config, result mapping (PASS/FAIL/INCONCLUSIVE only), failure
      mapping, provenance requirements, SHA-256 commitment (D-L10-2)
- [ ] L10 Contract Registry: append-only, two-way identity agreement,
      rebind refusal, active|withdrawn, atomic resolve snapshot (D-L10-2,
      D-L9-10 pattern); no harness/model write capability (AST audit +
      decoded registry scan — the no-catalog-verb wall)
- [ ] Eligibility enforcement: binding a capability not registered
      deterministic-verifier-eligible refused fail-closed (D-L10-3)
- [ ] Register A registry/schema proofs
- [ ] Security review (Class 3)

## 2. L10-M2 — Evaluator + evaluation record (Class 3)

- [ ] Bounded evaluator: mechanical-validity checks + declarative mapping;
      machinery-reserved UNAVAILABLE/INVALID with closed per-class reason
      vocabulary; evaluator invariant-failure mints NO outcome (D-L10-4/6/8)
- [ ] Evidence admissibility validation against contract input spec
      (kind, authority class, task binding); INVALID w/ execution skipped
      recorded; unresolvable object → UNAVAILABLE (D-L10-7/8 amendment)
- [ ] Evaluation record: references + L10-computed facts per D-L10-10;
      contract-bytes (and externalized config) stored as L6
      evidence-payloads in the no-reopen window; raw + canonical results
      both durable; recorded through the existing L7→L6 discipline,
      record-before-event (D-L10-6 stage 5)
- [ ] Hostile-verifier proofs (vocabulary-string outputs, out-of-domain,
      garbage) + Register T evaluator entries
- [ ] Security review (Class 3)

## 3. L10-M3 — L7 amendment: verification seam (Class 3, archived-layer amendment per D-L10-17)

- [ ] Workflow-schema gate vocabulary: (opaque contract token, required
      outcome), exact-pin only; five verification events join the control
      vocabulary; declaration-gated verification exposure as load refusal;
      reachability-based totality (D-L10-13, both owner amendments)
- [ ] δ latest-per-token walk state (CallState pattern); Register D
      replayer extended to re-derive it
- [ ] Every new loader rule pinned by a doctored definition
- [ ] Archived L7 suite re-run green; formal amendment record created at
      openspec/changes/archive/2026-09-07-layer-07-orchestration/
      amendments/L10-verification-seam/ (definition, affected invariants,
      loader changes, new proofs, suite result, scoped review, hashes)
- [ ] Scoped architecture review of the amendment (not a re-grill)

## 4. L10-M4 — First verifier capability (Class 3)

- [ ] One closed verifier capability (run_go_build@v1 or run_go_tests@v1):
      pinned invocation + toolchain identity, closed typed parameter
      surface, result domain, environment-independent canonicalization via
      already-governed mechanism, L5 spec class (D-L10-3/12/16)
- [ ] L4 registration artifacts authored as PROPOSED (registration is a
      Governance/owner act — machinery never self-registers)
- [ ] Matching Verification Contract authored as PROPOSED
- [ ] Registration-review obligations checklist applied (result-domain
      honesty, completeness, freshness, sequencing, latest-per-contract,
      canonicalization reproducibility)
- [ ] Security review (Class 3)

## 5. L10-M5 — Reconstruction + observability views (Class 2/3)

- [ ] Reconstruction tooling: read-only, all D-L10-10 two-source checks,
      canon(raw) check, mapping recomputation; typed failure on missing
      input; discrepancy artifact as L6 evidence-payload in audit scope
      outside task streams; original record untouched (D-L10-12)
- [ ] L6 vocabulary check (OPEN from D-L10-12): confirm evidence-payload
      carries the discrepancy artifact without authority-semantics change;
      stop at the Architecture Decision Gate if it does not
- [ ] Record-derived views: pure functions w/ derived-from provenance;
      sensitivity inheritance + classification enforcement, fail-closed on
      absent required classification; no L6 write reachability (AST audit);
      no view→model path (D-L10-5/14)
- [ ] Register P proofs (mutation-coupled)

## 6. L10-M6 — remediate-dependency@1 + live proof (Class 2/3)

- [ ] remediate-dependency@1 Skill bundle authored (ANALYZE read-only →
      REMEDIATE write → VERIFY gated on contract PASS → COMPLETE);
      L9 composition-coherence check covers gate/contract pin agreement
      (D-L10-13)
- [ ] Governance registration of capability + contract + skill (OWNER ACTS)
- [ ] Register E live proof vs local model through the unmodified L7 loop;
      negative live proofs (no-proposal exhaustion, genuine FAIL re-entry,
      equivalence)
- [ ] Register R replay/multiplicity proofs incl. downgrade scenarios

## 7. Close

- [ ] Three Class-3 reviews (security, test, architecture) against the
      final implementation; every CRITICAL/HIGH remediated with
      mutation-verified regressions
- [ ] Egress anti-smuggling schema proofs (D-L10-18)
- [ ] Traceability at archive: Q-L10 decision → mechanism → register →
      test → evidence → disposition
- [ ] Owner acceptance of three-state verdicts; code map + status doc
      updates; push/archive on owner approval

## 8. Deferred (recorded residuals — never silently promoted)

- Live operational telemetry (Q-L10-15 → dedicated future grill)
- Detect-and-report-only classification consistency checker (D-L10-14)
- Egress/inbox machine-typing contract — opens only on a concrete Themis
  ingestion requirement, at the egress boundary (D-L10-18)
- Probabilistic/model-assisted verification band (D-L10-3)
- Generic/parameterized command runner — unregistrable in v1; reopening is
  a dedicated architecture decision (D-L10-16)
- Workflow-defined evaluation-selection policy (D-L10-9)
- Verification-outcome reason codes reaching δ (D-L10-8)
