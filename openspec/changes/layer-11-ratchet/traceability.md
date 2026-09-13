# Traceability: Layer 11 — Ratchet

Filled at M7 (pre-review state; review remediations may extend).
Realizations live in src/harness/ratchet/ (pkg) and
src/harness/cmd/themis-ratchet/ (invocation surface).

| Decision | Realized by | Proven by |
|---|---|---|
| D-L11-1 boundary/authority | whole package shape: read-only registries, pure comparators, no promotion/security/authority surface anywhere | e2e §7 governance-plane byte-identity; wall_test importer closure; architecture review (M7) |
| D-L11-2 improvement ladder | enforced by absence — no position/lifecycle state representable in any schema | candidate_test lifecycle-field refusals; compare_test package-schema wall |
| D-L11-3 Candidate | candidate.go (closed schema, content-hash, claim-not-choice, lineage-as-claim) | candidate_test refusals + no-lifecycle proof; e2e §1 |
| D-L11-4 proposition boundary | compare.go ComparisonPackage (conditioning tuple, Δ-in-package only); derive.go stateless | compare_test determinism/symmetry/schema walls; TestScalarizationRetainsDelta |
| D-L11-5 baseline authority | AdmissionObservation (door-registry-hash-grounded); refusal preconditions; ClaimMismatch never substituted | compare_test refusal cases + TestCompareClaimMismatchRecorded |
| D-L11-6 criterion registry | criterion.go closed schema + normative-token wall; registry.go (append-only, two-way identity, exact refs); comparator.go option-A table | criterion_test 20 doctored refusals; registry_test; wall_test no-write |
| D-L11-7 orderings/tradeoffs | criterion.go ordering taxonomy validation; derive.go (per-metric refuses overall; dominance incomparable; scalarization = projection) | TestDominance, TestScalarization, TestPerMetricRefusesOverallRelation, TestNoOrderingNoBetterClaim |
| D-L11-8 selection ≠ promotion | no L11 code touches the router; ratchet package unreachable from internal/service | wall_test TestRatchetImportersAreClosed |
| D-L11-9 regression evidence | package.go BuildRegressionPackage (complete-under-S mechanical); DeriveResistantUnderSet (derived, never stored); ResolveSet takes exact S@v | reconstruct_test TestRegressionPackageCompleteness (incl. regression-found-is-first-class) |
| D-L11-10 describe/consume | plan.go declarative-only closed schema; CheckPlanConformance = record-to-plan matching; no execution API exists | candidate_test TestPlanImperativeUnrepresentable, TestPlanConformance; wall_test no os/exec |
| D-L11-11 two planes/identity | store.go (evidence-payload objects only); InstanceID = content hash; no mutable package state | TestStoreRoundTrip; wall_test TestNoMutablePackageState; TestForbiddenComponentNames |
| D-L11-12 no feedback | M0 scaffold deletion (b16fc87); no feedback identifier representable | wall_test TestForbiddenComponentNames |
| D-L11-13 corpus | fixtures/goldens = pinned K parameters (config by value, hashed); no corpus directory or curation API | criterion schema (config pinned into criterion hash); registry hash-verification tests |
| D-L11-14 automation boundary | synchronous-only cmd/themis-ratchet; Compare→durable result (no-discard); no candidate constructor in machinery | wall_test (no timers/goroutines/watchers); compare_test (accepted invocation always yields package or refusal) |
| D-L11-15 consumption classes | selectorSources excludes L11 output (terminal, refusal names the wall); reconstruction/derivation = the two licensed re-reads | criterion_test "selector source is L11 output"; TestRatchetImportersAreClosed |
| D-L11-16 failure semantics | ReasonClass closed vocabulary; package/refusal/discrepancy only; no outcome enum anywhere | TestCompareRefusals (reason classes + neutrality); reconstruct_test three-result separation |
| D-L11-17 reproducibility | ComparisonPackage carries full tuple; comparator determinism (canonical serialization); reconstruct.go cold three-result | TestCompareDeterminism (50 reruns bit-identical); TestReconstructIsCold/Confirmed/MissingInputs/Discrepancy/NeverRepairs |
| D-L11-18 security boundary | normativeTokens wall at parse; no security predicate fields; regions named structurally | criterion_test normative-name refusals; TestPackageSchemaHasNoEvaluativeFields; security review (M7) |
| D-L11-19 recursion | criterion-revision + regression-set-revision families; no L11-about-L11 source representable | Families table; selector-source wall; TestRatchetImportersAreClosed |
| D-L11-20 closure | this file + tasks.md + M0 deletion; implementation = whitelist inventory only | M7 closure audit re-run against the artifact inventory |

## Live proof (M6)

TestLiveModelAuthorsCandidate — PASS vs qwen2.5:7b (2026-09-12,
5.98s): the model reads a derived comparative view as data, authors
candidate substance (family/target/relation/rationale), the
authoring surface adds hash commitments, ParseCandidate walls the
result, the full compare arc runs over it, everything durable.
Class-6 discipline held: readable ≠ authoritative; authorship
attributed ("qwen2.5:7b" via "live-proof:ollama").

## Explicit gaps / residuals (per Gate 0)

- Knowledge family: representation + egress-class storage proven
  (TestKnowledgeCandidateRepresentationOnly); HANDOFF to the Themis
  ingestion door NOT proven — door unavailable in this environment.
  Explicit residual per the Gate 0 rule; no substitute authority
  introduced.
- CheckAppendOnly is implemented and proven as a check; a LIVE
  consumption wall (the L10 seam precedent) awaits a resident
  consuming process — the CLI is one-shot, each invocation loads one
  atomic registry state. Recorded, not hidden.
- cmd/themis-ratchet is a thin flag wrapper without its own test
  binary run; behavior covered at package level. (Review may
  reclassify.)
