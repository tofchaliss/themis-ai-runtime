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

- [x] Workflow-schema gate vocabulary: Edge.Gate (opaque token +
      outcome), exact-pin syntax, closed outcome vocabulary, gate-ladder
      w/ mandatory ungated fallback, counter-free gated edges; five
      verification events in the constitution (ConstitutionHash changed,
      deliberate); declaration-gated exposure + reachability totality
      enforced at assembly (registry-aware); L6 gains the additive
      l10-verification event class; L4 ToolDef gains verifier_eligible
- [x] δ gate-aware edge selection + verifState latest-per-token walk
      state (CallState pattern); fired branch recorded in the transition
      body w/ gated edge identity "<phase>/<on>#g<idx>". Register-R
      replay proofs over gate-bearing walks deferred to M6 (need M4's
      capability + scripted walks) — recorded in the amendment
- [x] Every new loader/assembly rule pinned by a doctored definition
      (verification_seam_test.go: 9 refusals + ladder determinism +
      conditional totality + pre-amendment invariance)
- [x] Archived L7 suite re-run GREEN (105.9s full suite incl. fault
      sweeps + kill recovery + skill registers); tools/ + state/ green;
      formal amendment record created (archive/2026-09-07-layer-07-
      orchestration/amendments/L10-verification-seam/AMENDMENT.md); the
      two API-closure allowlist extensions were guard-caught first, then
      deliberately recorded; go list -deps proves L7 verification-blind
- [ ] Scoped architecture review of the amendment — batched into the
      L10 close reviews (M7), scope-tagged to the amendment record

## 4. L10-M4 — First verifier capability (Class 3)

- [x] First verifier capability: **verify_report@v4** — in-process,
      read-only raw capture (executor = read_file discipline), closed
      typed parameters (path + exact contract ref), result domain
      {report_valid, report_invalid} (total canonicalization — every
      input maps; out-of-domain unreachable), pure environment-free
      canonicalization registered in the seam (already-governed
      mechanism per D-L10-12 amendment).
      **SCOPE FINDING (recorded, owner attention):** run_go_build/
      run_go_tests-class verifiers require process execution, and L5's
      exec surface is deliberately git-only — a process-exec capability
      is an L5 architectural amendment with its own gate. NEW RESIDUAL:
      "L5 process-execution amendment for external-tool verifiers." The
      M6 slice gates on verify_report (real deterministic verification
      through the full seam) instead
- [x] L4 registration artifacts PROPOSED: policies/tools/
      registry-v4.proposed.json (registry-v3 + verify_report,
      verifier_eligible: true) — owner registers
- [x] Verification Contract PROPOSED: policies/verification/
      report-valid/contract.json + contracts.proposed.json (report-valid@1
      binds verify_report @ registry-v4 hash) — owner registers
- [x] Registration-review obligations applied: result domain total and
      honest; completeness = single required slot; freshness = read-at-
      call from current workspace state (inherently current; noted);
      sequencing = M6 lattice review obligation; canonicalization purity
      proven by test; latest-per-contract understood
- [x] Seam implementation (verification/seam): EvaluateCall composes
      atomic resolution → canonicalization → pure evaluation; eligibility
      = verifier_eligible AND pinned-registry-hash == registry-in-force;
      pre-instance refusals typed (no contract, unregistered, floating,
      capability mismatch, registry drift, withdrawn); no-reopen contract
      bytes carried from resolution; proofs green
- [ ] Security review (Class 3) — batched with M2/M3 into the close
      reviews (recorded deviation: single seam review covers evaluator +
      L7 amendment + capability together)

## 5. L10-M5 — Reconstruction + observability views (Class 2/3)

- [x] Reconstruction: pure core (verification/reconstruct.go — supplied
      bytes only, no I/O) + seam wrapper over the live L6 root
      (ReconstructTask): two-source checks, canon(raw) recomputation,
      mapping recomputation as the authoritative legitimacy check; typed
      missing-input failures; discrepancy artifact stored via the root
      ObjectStore as evidence-payload OUTSIDE all task streams; original
      events proven byte-identical after reconstruction
- [x] L6 vocabulary check RESULT: evidence-payload class + existing
      store primitive carry the discrepancy artifact with no
      authority-semantics change and no task-stream append — no new L6
      object class needed. RECORDED ADG FOLLOW-UP: audit-scope objects
      are unanchored under L6's future reachability GC (no deleter
      exists today; "no probabilistic GC") — an anchoring decision must
      precede any GC implementation
- [x] Record-derived views: pure VerificationHistory w/ latest-per-
      contract projection + derived-from provenance (event range + view
      version); purity + downgrade-ordering proven. Views carry tokens/
      outcomes/seqs only (no evidence content — minimal sensitivity
      class); no L6 write reachability by construction (read seam only);
      no view→model path exists
- [x] Register P proofs: consistent-reconstructs; six single-element
      swaps fail (foreign contract bytes, claimed outcome, swapped
      raw/canonical, claimed identity, claimed capability); four typed
      missing-input cases; machinery-outcome records skip the result
      chain; end-to-end forged-outcome discrepancy over a real root

## 6. L10-M6 — remediate-dependency@1 + live proof (Class 2/3)

- [x] remediate-dependency@1 Skill bundle authored PROPOSED
      (policies/skills/remediate-dependency/: ANALYZE read-only →
      REMEDIATE write + verify_report + PASS-gated completion; procedure,
      schema, templates, ceiling, manifest); catalog.proposed.json v2
      resolves both skills through the real L9 machinery. Contract pins
      are realized as the pinned workflow's gate tokens (exact immutable
      pins inside a pinned artifact — no L9 manifest change needed;
      recorded reading of D-L10-2/13)
- [ ] Governance registration of capability + contract + skill (OWNER
      ACTS: registry-v4.proposed.json → registry-v4.json;
      contracts.proposed.json → contracts.json;
      catalog.proposed.json → catalog.json)
- [x] Register E: **LIVE PASS vs qwen2.5:7b through the unmodified
      production loop with the REAL seam** — write report → verify under
      report-valid@1 → PASS → gate → COMPLETED → cold reconstruction
      consistent (14.6s). Scripted closure demonstration + FAIL-closes-
      gate negative with the real contract. Negative paths deterministic
      via scripted walks (no-proposal exhaustion, refusal-is-not-outcome);
      a live mis-tooled walk (write_file w/ unknown field, prose stall)
      failed closed through the reviewed exhaustion path — recorded as
      incidental live negative evidence
- [x] Register R: gate-walk proofs through the production loop —
      FAIL→PASS remediation (both instances durable, causal replay fires
      the gated edge #g0), no-verification non-completion, PASS→
      UNAVAILABLE downgrade closes the gate, refusal produces no event,
      both assembly refusals; the Register D replayer extended with the
      gate ladder + latest-per-token re-derivation, all archived replay
      proofs green

## 7. Close

- [x] Three Class-3 reviews run against the final implementation
      (2026-09-11), all findings dispositioned:
      **Security** (verdict: walls verified holding): H-1 record
      selection forgeable → REMEDIATED (event body names the record
      object; explicit lookup; PoC path dead). M-1 missing cross-checks
      → REMEDIATED (event-body-vs-record + execution_ref→L4-audit
      capability/registry checks in reconstruction). M-2 append-only
      dead in live path → REMEDIATED (per-evaluator prior-state
      CheckAppendOnly; violation = machinery error → invariant path).
      M-3 no disjointness wall → REMEDIATED (Evaluator.CheckDisjoint,
      exercised in e2e; deployment obligation documented). L-1 auth-
      registry drift → REMEDIATED (authorizing registry hash crosses
      the seam; drift refuses). L-2 symlink race in readGoverned →
      REMEDIATED (single-handle open/stat/limited-read). L-3
      control+verifier_eligible → REMEDIATED (load refusal).
      **Architecture** (verdict: CONDITIONAL PASS): H-1 = security H-1,
      remediated. M-2 config-check tautology → degeneracy recorded in
      code + hard obligation at the L5 process-exec amendment. M-4
      missing per-archive records → REMEDIATED (L4 + L6 amendment
      stubs cross-referencing the L7 record). L-1 unanchored refusal
      object → REMEDIATED (removed). L-2 multi-slot contracts →
      REMEDIATED (typed seam refusal + test). L-3 resolvability
      conflation → REMEDIATED (empty capture grades through
      canonicalization; test renamed). L-4 stale scaffolds → left for
      owner disposition at archive (see owner list).
      **Test** (verdict: strong core, gaps closed): survived ladder
      mutation → KILLED (TestTwoGateLadderInProduction, production δ,
      two satisfiable gates, #g0 wins). Fault-window deferral → CLOSED
      (3 fault points + TestVerificationFaultSweep + sync guard;
      recorded equivalent mutant: verifState-before-commit is
      unobservable since commit failure is always fatal pre-gate).
      Register T #8 cross-task replay → CLOSED
      (TestCrossTaskReplayCannotSatisfyGate). T #5 event tamper →
      CLOSED (TestEventTamperDetectedAtReadBoundary). INVALID/
      INCONCLUSIVE production walks → CLOSED. CheckAppendOnly wired
      (security M-2). Seeder-drift risk noted: the consistent
      reconstruction case is production-driven via e2e; the seeder
      remains only for the forgery case.
- [x] Full module suite green post-remediation (16 pkgs + context
      green in isolation — the documented pre-existing L2 contention
      flake, unrelated); live Register E re-passed in-suite
- [x] Traceability drafted (traceability.md); egress anti-smuggling is
      structural in v1 (no new egress surface; schema walls at the
      contract/gate layers proven)
- [x] OWNER ACTS + DECISIONS (2026-09-11, closure judgment PASS):
      three-state verdicts accepted (conformant / evidenced / proven
      for the verify_report slice); Governance registrations performed
      as directed owner acts (registry-v4.json, verification/
      contracts.json, skills/catalog.json ACTIVE; tests retargeted);
      D-L10-6 realized stage order BLESSED as v1 clarification;
      Register E ACCEPTED via verify_report (run_go_* residual intact);
      contract-pins-as-gate-tokens SIGNED OFF without a duplicate pin
      set; stale scaffolds REMOVED (verification/{build,evidence,lint,
      security,test}, observability/* — no artifacts lost, .gitkeep
      only); status/code-map updates + push + archive APPROVED

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
