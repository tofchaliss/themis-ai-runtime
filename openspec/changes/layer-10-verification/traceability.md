# Layer 10 Traceability — grill decisions → mechanisms → registers → tests

Tests in `src/harness/verification`, `src/harness/verification/seam`, and
`src/harness/orchestration`. Decisions from `design.md` §2 (D-L10-1..18 +
D-L10-1a), registers from §4. Review record in tasks.md M1–M7. (Review
dispositions filled at close.)

| Decision / invariant | Mechanism | Register | Evidence |
| --- | --- | --- | --- |
| D-L10-1 — boundary/authority: L10 evaluates registered contracts + derives views; never security meaning, never a record plane, never an orchestrator | package layout: `verification` (pure evaluator, loaders, views) + `verification/seam` (composition); no L6 write API in verification (AST wall); evaluator invoked only in the L7 result path | A | `TestNoRegistryWriteCapability` (AST SelectorExpr walk + audit-of-the-audit), `go list -deps ./orchestration` = 0 verification hits |
| D-L10-1a — the ladder; X′ narrower than X | positions mechanized: model claims inert (L7 proven), evidence as input only, outcomes minted solely by the evaluator, no governance state representable | A, T | `TestHostileVerifierOutputIsInertDomainData`, egress vocabulary walls (schema refusals) |
| D-L10-2 — contracts: governed, registered, closed schema, anti-smuggling | `contract.go` fail-closed loader; `registry.go` append-only registry, no write API, exact pins, two-way identity | A | `TestContractClosedSchema` (incl. NOT_AFFECTED unrepresentable), `TestRegistryRefusals`, `TestAppendOnly`, `TestPathEscapeRefused` (incl. symlink), `TestHardeningRefusals` (dup keys, non-object config, non-canonical versions, size/name bounds) |
| D-L10-3 — determinism as registration property; canonicalization registered; model never a verifier | `verifier_eligible` L4 registration property; `EligibilityChecker` fail-closed (nil/error/false refuse); canonicalizers registered in code per capability | A | `TestEligibilityFailsClosed`, seam `TestPreInstanceRefusals/registry-version mismatch`, `TestCanonicalizationIsPure`; M1 mutation probes (nil-checker) killed |
| D-L10-4 — five-value vocabulary; machinery statuses unmappable; typed L7 events without collapse | `Outcome` + `contractMappable`; five constitution events; per-class `Reason` vocabulary + `CheckReasonClass` | A | loader refusals (INVALID/UNAVAILABLE in mapping), `TestReasonClassIntegrity`; M2 mutation probe (nearest-match laundering) killed |
| D-L10-5 — observability = derivation; L6 records → L10 derives; no second history | pure `VerificationHistory` view w/ derived-from provenance; reconstruction observability-class; no collection anywhere | P | `TestVerificationHistoryView` (purity, downgrade ordering), `TestReconstructTaskDiscrepancy` (stream byte-identical after) |
| D-L10-6 — five-stage execution; verifier never mints; record-before-event; model-proposed only | seam stages 1+4 around the ordinary L4→L5 call; `loop.evaluateVerification` stores → EvVerification commit → δ; no auto-invoke path exists | T, R, E | `TestHostileVerifierOutputIsInertDomainData`, `TestNoVerificationNoCompletion` (no-bypass invariant live), `TestGateWalkRemediation` |
| D-L10-7 — model trust boundary: two apertures; contract-identity-bound gates; substitution unrepresentable | proposal args carry contract ref; gates match opaque tokens; verification events originate only from committed evaluations | T, R | `TestVerificationRefusalIsNotAnOutcome`, gate-ladder foreign-token test (`TestGateLadderDeterminism`), `TestPreInstanceRefusals` |
| D-L10-8 — stage-indexed failure semantics; instance-creation boundary; result-domain rule; evaluator failure mints nothing | pure `Evaluate` fixed stage order; refusals pre-instance in the seam; `Evaluate` error path returns no outcome | A, T | `TestStageIndexedFailures` (all reasons), `TestEvaluatorFailureMintsNoOutcome`, `TestEmptyEvidenceGradesUnavailable`, `TestOutcomesAcrossReportShapes` |
| D-L10-9 — multiplicity; latest-per-contract gates; stateless-at-δ; downgrade asymmetry | `verifState` walk state (CallState pattern); gate ladder in `step()`; replayer re-derivation | R | `TestGateWalkRemediation` (both instances durable, [FAIL PASS]), `TestDowngradeClosesGate`, `TestGateLadderDeterminism`, extended `replayAndVerify` green on all walks |
| D-L10-10 — provenance: references + computed facts; no identity without bytes; two-source checks; outcome is historical fact | `Evaluation` record shape; contract bytes carried on `Contract.Raw` (no-reopen); stores in the loop before the event | P | `TestProposedBundleIsConsistent` (computed identities, no-reopen hash), `TestReconstructSingleElementSwapsFail` (six swaps), `TestReconstructMissingInputsTyped` |
| D-L10-11 — Register T tamper model | walls across loaders, registry, evaluator, seam, L6 integrity | T | M1/M2 mutation probes (7 killed), `TestRegistryRefusals/tampered`, `TestReconstructTaskDiscrepancy` (forged outcome detected), write-capability walls |
| D-L10-12 — reconstruction vs re-evaluation; discrepancy artifacts; no retroactive mutation | pure `Reconstruct` + seam `ReconstructTask`; artifacts via root store outside task streams; no replay authority anywhere | P | `TestReconstructConsistent` (purity), `TestReconstructTaskDiscrepancy` (artifact stored, stream byte-identical), `TestReconstructMachineryOutcomesSkipResultChain` |
| D-L10-13 — contract-blind L7; minimal payload; gate vocabulary; declaration-gated exposure; reachability totality | opaque tokens in `GateCond`/events; loader + assembly rules; injected evaluator seam | A, R | `TestVerificationSeamLoaderRefusals` (9 doctored), `TestVerificationEventsConditionallyTotal`, `TestVerificationAssemblyRefusals` (both), deps-proof |
| D-L10-14 — sensitivity: inheritance, no second detector, no view→model | views carry tokens/outcomes/seqs only; no exposure surface built; classification enforcement deferred with the surface (none exists in v1) | — | structural (no surface to test); recorded in tasks.md M5 |
| D-L10-15 — four propositions never collapsed | COMPLETED from lattice only; PASS from evaluator only; no governance state exists; vocabulary rule in code comments/docs | R, E | `TestNoVerificationNoCompletion` (COMPLETED ⇏ reachable without PASS under this lattice), `TestRemediateDependencyFailClosesGate` |
| D-L10-16 — OPEN-2 dissolved; closed verifier capabilities; no run_command | `verify_report` executor (raw capture only, confined); invocation is code identity; params closed by registry schema; L5 process-exec recorded residual | A, E | registry loader + executor-table completeness; `TestRemediateDependencyE2E`; scope finding recorded (tasks.md M4) |
| D-L10-17 — extension discipline; L7 amendment protocol | formal amendment record (L7 archive amendments/L10-verification-seam/); archived L7+tools+state suites green; additive-only pinned | R | `TestPreAmendmentWorkflowUnchanged`, archived-suite re-runs (tasks.md M3), AMENDMENT.md |
| D-L10-18 — expected outputs subsumed; egress guarded | output expectation = the report-valid@1 gate in the slice; no expected_outputs field anywhere; egress schema untouched (existing path) | E | the slice itself (report gate = output contract); egress-schema refusal proof deferred to the egress surface (none new in v1) |
| Gate-0 condition — machinery never self-registers | all registrations authored as *.proposed.json; no write APIs (walls); owner acts pending | A | write-capability walls; PROPOSED artifacts on disk; registration steps listed in tasks.md M6 |

**Register E (live):** `TestLiveRemediateWalk` **PASS vs qwen2.5:7b**
through the unmodified production loop with the real seam — write
report → verify under the PROPOSED contract → evaluator PASS → gate →
COMPLETED → cold reconstruction consistent (14.6s). An incidental live
negative (mis-tooled walk: unknown-field denial + prose stall) failed
closed through the reviewed exhaustion path.

**Recorded residuals (tasks.md §8 + discovered):** live telemetry
grill · detect-and-report classification checker · egress/inbox typing
contract · probabilistic verification band · generic command runner ·
workflow-defined selection · reason codes at δ · **L5 process-exec
amendment for external-tool verifiers (M4 scope finding)** · **L6
audit-scope retention anchoring before any GC (M5 finding)** · L9
loader duplicate-key/version-syntax hardening (M1 review follow-up) ·
fault-point sweep over the verification store/commit window.

Reviews: M1 security review remediated in-line (H-1 AST wall, M-1
duplicate keys, L-1..6). Close reviews (security, test, architecture)
run against the final implementation — dispositions recorded in
tasks.md §7 at close.
