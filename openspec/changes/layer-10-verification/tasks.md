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
- [x] Gate 0: design + proof plan accepted by owner (2026-09-11, "PASS /
      IMPLEMENTATION AUTHORIZED"; all five registers judged sufficient;
      milestones approved as written; **Gate-0 condition: implementation
      must never convert a PROPOSED Governance registration into an active
      registration merely because the machinery exists — registration
      authority remains outside the machinery, surviving implementation
      and the proof suite**; M6 closure demonstration = the full
      PROPOSED→registered→instantiate→walk→verify→reconstruct trace with
      negative Governance-boundary paths)

## 1. L10-M1 — Contract registry + schema (Class 3)

- [x] Verification Contract artifact + fail-closed loader
      (src/harness/verification/contract.go): closed schema,
      DisallowUnknownFields, trailing-content refusal, exact name@version,
      capability pin (name + registry hash), evidence input spec (closed
      kind vocabulary, task_bound mandatory true), config by value inside
      the canonical representation (64KiB bound — no external config store
      needed in v1, D-L10-10 #3 trivially satisfied), result mapping
      (PASS/FAIL/INCONCLUSIVE only; machinery statuses and Governance
      vocabulary unrepresentable), provenance completeness not
      contract-relaxable, SHA-256 commitment. NOTE: the D-L10-2
      failure-semantics mapping is realized as the INCONCLUSIVE entries of
      the result mapping + the machinery-reserved statuses — no separate
      free-form field (recorded for owner review)
- [x] L10 Contract Registry (registry.go): append-only + CheckAppendOnly
      (deletion/rebind/un-withdrawal detected), two-way identity
      agreement, rebind refusal at load (duplicate = refusal),
      active|withdrawn, atomic single-load resolve, confine.ResolvePath +
      Lstat on contract paths; no write API — pinned by AST audit
      (alias-resistant, the L9 wall)
- [x] Eligibility enforcement fail-closed: nil checker, checker error, or
      ineligible → refusal (EligibilityChecker seam; L4 wiring lands with
      M3/M4 under the archived-layer amendment discipline)
- [x] Register A registry/schema proofs (verification_test.go) + 4
      mutation probes killed (mapping wall, nil-checker, two-way
      identity, rebind detection); gofmt/vet/tests green
- [x] Security review (Class 3, 2026-09-11): H-1 (write wall was
      string-matching, PoC-bypassed by alias/newline/function-value
      binding) → remediated with a true AST SelectorExpr walk flagging
      any REFERENCE to os writers, extended writer set, forbidden
      indirect-import check, and an audit-of-the-audit test; probe
      re-run confirms the doctored binding is flagged. M-1 (duplicate
      JSON keys last-wins) → token-level duplicate-key refusal in both
      loaders. L-1..L-6 all hardened (object-only config, canonical
      version syntax, size bounds, name length cap, required-slot rule,
      symlink-escape test). Suite green post-remediation.
- [ ] FOLLOW-UP (archived-layer hardening, from review): the L9
      catalog/manifest loaders share the duplicate-key and
      version-syntax latencies — fix under the D-L10-17 amendment
      discipline as a separate small change, not silently

## 2. L10-M2 — Evaluator + evaluation record (Class 3)

- [x] Bounded PURE evaluator (verification/evaluate.go): stage-indexed
      mechanical-validity checks + declarative mapping; closed per-class
      reason vocabulary with class-integrity checker; evaluator
      invariant-failure returns error and mints NO outcome (the sixth
      terminal, D-L10-8)
- [x] Evidence admissibility: closed-world slot filling, task binding,
      object-identity syntax → INVALID; structurally-valid-but-unreadable
      → UNAVAILABLE (owner amendment honored); deterministic facts
      supplied by the governed caller, never the model
- [x] Evaluation record type: references + L10-computed facts (contract
      identity computed from loaded bytes, execution by reference,
      matched-mapping recorded as non-authoritative convenience).
      NOTE: the L6 stores (contract bytes, raw+canonical objects) and the
      record-before-event write are the L7 seam's half of stage 5 —
      implemented in M3 where the seam forms; the evaluator itself is
      deliberately I/O-free
- [x] Hostile-verifier proofs: PASS/FAIL/NOT_AFFECTED/SECURE/garbage as
      canonical results are inert domain data → INVALID; a contract
      mapping domain string "PASS"→FAIL proves the mapping mints, not
      the spelling. 3 evaluator mutation probes killed (nearest-match
      laundering, config-check removal, task-binding drop)
- [ ] Security review (Class 3) — batched with M3 (the seam review needs
      both sides; recorded deviation, evaluator is pure and unwired)

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
