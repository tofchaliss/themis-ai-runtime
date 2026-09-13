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

## M7 close reviews and remediation (2026-09-13)

Three Class-3 reviews ran against b16fc87..fca10e0. Convergent core
finding (security CRITICAL C-1 = architecture H-1; security H-1 =
architecture H-2): the invocation boundary trusted its caller for
inputs the constitution assigns to L11's own mechanical
verification. Remediated:

- **C-1 admission forgery → door.go**: ObserveAdmission resolves
  the owning door's registry BYTES (closed door table:
  l9-catalog/l10-contract-registry/l11-criteria/l11-regression-sets)
  and constructs the observation; the CLI no longer accepts an
  --admission file. Reconstruct now takes door-registry bytes and
  re-verifies the observation via the SAME derivation
  (ReverifyAdmission) — hash mismatch = missing input, content
  disagreement = discrepancy. Proven: TestObserveAdmission incl.
  the forged-observation-cannot-reconstruct arc.
- **H-1 evidence self-grounding → ground.go**: GroundFacts resolves
  L6-plane refs in the store (byte equality, ObjectID syntax) and
  external-plane refs through a confined resolver; registered
  selector params are mechanically APPLIED (scalar subset-match) at
  grounding, at Compare, and at reconstruction. Params are
  scalar-only and lexically walled at registration. Proven:
  TestGroundFacts, TestCompareParamsApplied.
- **Refusal reachability (arch M-1/test H)**: acceptance boundary
  defined in the CLI — syntax errors are usage errors (exit 1,
  nothing minted); after acceptance, unregistered/withdrawn
  criterion and registry-pin mismatch mint DURABLE refusals
  (ReasonUnregisteredArtifact now reachable). Proven:
  TestCLIContract.
- **Registry binding (sec M-5)**: packages carry
  criteria_registry_sha256; --registry-sha256 pin = admission-at-
  consumption (the verdict/digest pattern).
- **Missing-inputs durability (arch M-2)**: cmdReconstruct stores
  missing-inputs facts as well as discrepancies.
- **Set hash binding (arch M-3)**: BuildRegressionPackage binds
  constituents to registered criterion BYTES (hash), not ref
  strings; cmdSet resolves members against the criteria registry
  and refuses non-member bindings.
- **Strict CLI inputs (sec M-2 / arch M-4)**: ParseEvidenceRefs
  (dup-key, unknown-field, trailing, 4MiB cap); all CLI file reads
  bounded.
- **Reconstruction parity (sec M-3)**: reverifyFacts now applies
  duplicate-selector, source-match, and params checks; run-identity
  presence, candidate-hash and registry-binding syntax re-verified.
- **Test-review mutants**: ~19 reasoned survivors killed
  (remediation_test.go: tie bands, boundaries, scalarization-equal,
  checkDeltaShape ×3, CanonicalDelta golden bytes, reconstruct
  clauses, set cross-constituent, plan-conformance clauses,
  loader-refusal expansion ×30+). Wall upgrades: exported-method
  closure, io/ioutil+syscall+dot-import, full-path importer match,
  pointer/alias writes.
- **CLI contract (test H)**: compiled-binary suite — refusal→exit 0
  stored; usage error→exit 1 nothing minted; cross-process
  reconstruction CONFIRMED (the D-L11-17 cold-binary evidence).
- ***.proposed.* handoff arc (test H)**: TestProposedHandoffArc —
  proposed registries are inert; the owner act activates;
  activation copies, never mutates. policies/ratchet/
  {criteria,regression-sets}.proposed.json created AWAITING the
  owner's Governance act (bench-score-delta@1, core-regression@1).

## Explicit gaps / residuals (per Gate 0 + reviews)

- Knowledge family: representation + egress-class storage proven;
  HANDOFF to the Themis ingestion door NOT proven — door
  unavailable. Explicit residual per the Gate 0 rule; no substitute
  authority introduced.
- Live CheckAppendOnly wall: the CLI is process-per-invocation;
  append-only continuity across invocations is enforced by the
  --registry-sha256 consumption pin (supplied by the door/operator,
  the gatePassed pattern) rather than persisted prior state — a
  persisted "last observed registry" would be L11-owned mutable
  state (D-L11-11). Full CheckAppendOnly live wall applies when a
  resident consumer exists. RECORDED DECISION, owner ratification
  at closure.
- No-discard scope (sec H-2, classified): the MACHINERY has no
  discard path — results are emitted only after durable store, and
  refusal/package paths are symmetric. Operator-level suppression
  (throwaway state roots, out-of-band re-runs) is request-level
  cherry-picking: constitutionally visible-not-prevented (D-L11-4
  §6), defended at doors via complete-under-S. The compute-to-store
  crash window mints nothing and emits nothing — indistinguishable
  from never-accepted. RECORDED CLASSIFICATION, owner ratification
  at closure.
- Sensitivity inheritance (sec M-4): packages embed evidence bytes
  verbatim; no classification field exists in v1. RESIDUAL +
  registration-review rule: criteria selecting sensitive-plane
  evidence must not be registered until inheritance machinery
  exists. Local single-user store makes this acceptable for v1.
- Advisory rationale is unbounded untrusted text; terminal emit is
  JSON-escaped; downstream renderers must treat it as data.
- Live proof is machine-local (ollama), skip-gated — recorded as
  such.
