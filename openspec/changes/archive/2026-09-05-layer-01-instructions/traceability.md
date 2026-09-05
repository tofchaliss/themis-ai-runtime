# Layer 1 Traceability — acceptance criteria and grill invariants → tests

All tests in `src/harness/instructions`. Layer-doc criteria from `docs/architecture/harness/01-instructions.md` §17; invariants from `design.md` §6.

## §17 acceptance criteria

| Criterion | Evidence |
| --- | --- |
| Root `AGENTS.md` can be loaded | **Deferred by Q-L1-1 (owner-accepted):** repository sources rejected in v1 — `TestLoadUnrecognizedSourceKinds` pins the rejection; activation contract recorded for L5 |
| Scoped instructions can be loaded | `TestLoadHappyPath`, `TestShippedContentResolvesAndRenders` |
| Harness instructions exist | `TestShippedContentResolvesAndRenders` (`instructions/global/`) |
| Themis instructions exist | `TestShippedContentResolvesAndRenders` (`instructions/themis/`) |
| Task instructions can be loaded | `TestLoadHappyPath`, `TestResolveOrderingAndTrace` |
| Skill instructions can be incorporated | **Deferred to L9** (immutable skill identity): `TestLoadUnrecognizedSourceKinds` pins the v1 rejection |
| Instruction scope is understood | `TestScopeOrderFixedForever` (order fixed forever, round-trip) |
| Instruction categories are defined | `TestLoadTrustedFailuresAbort/bad_category`, `TestRenderGolden` (fixed headings) |
| Instruction precedence is deterministic | `TestResolveOrderingAndTrace`, `TestResolveSourceOrderIndependence` |
| Conflicts are detected | `TestTaskNamespaceViolationRejectedAndRecorded`, `TestPatternRejectionRecorded`, `TestResolveConflictOrderDeterministic` |
| Invalid instructions are rejected | `TestLoadTrustedFailuresAbort`, `TestLoadUntrustedFailuresAbort`, `TestFrontmatterLineMalformations` |
| Effective instructions are generated before model invocation | `TestDeliveryProofMockProvider` (render precedes the call; L1 never calls the model) |
| Effective instructions are versioned | `TestResolveGoldenHash`, `TestResolveByteChangeSensitivity` (version = content hash, Q-L1-7-original) |
| Effective instructions are hashed | `TestCanonicalFormSpec` (independently derived canonical form) |
| Effective instruction metadata is recorded | `TestResolveOrderingAndTrace` (SourceHashes, Sources, PolicyHash), `TestResolveExemptionsAppliedOrderingAndTrace` |
| Safety instructions cannot be overridden by task instructions | `TestTaskNamespaceViolationRejectedAndRecorded` (incl. byte-identical body), `TestConflictRecordsRealBodyHash` |
| Repository content is not automatically treated as instructions | `TestLoadUnrecognizedSourceKinds` (recognition, never discovery) |
| Instructions are separated from context | Structural: the envelope does the split (design §6, Scenario 1); the only instruction inputs are registered Sources — no API accepts context |
| Instructions are separated from permissions | Structural: the package has no tool/authorization surface; `TestExemptionDoesNotAuthorize` pins exemption ≠ authorization |
| The selected model receives the effective instruction set | `TestDeliveryProofMockProvider` (byte-identical system message), `TestSystemMessage` |
| The instruction set can be reconstructed from an execution trace | `TestCanonicalFormSpec` + `SourceHashes` evidence in `TestResolveOrderingAndTrace`; `TestResolveConflictsStatusAndHash` (rejections reconstructable) |

## Grill invariants (design.md §6)

| Invariant | Evidence |
| --- | --- |
| Atomic resolution; no partial/degraded/fallback EIS (Q-L1-6) | `TestLoadMultiSourceAtomicity`, `TestResolveFailureAtomic`, zero-source case in `TestStatusOfMapping` |
| Explicit resolution statuses; conflicts ≠ ordinary success (Q-L1-6) | `TestStatusOfMapping` (per-sentinel, both directions), `TestResolveConflictsStatusAndHash` |
| Namespace ownership; content confers no ownership (Q-L1-7) | `TestNamespaceOwner`, `TestTaskNamespaceViolationRejectedAndRecorded`, `TestInlineProvenanceOverwritten` |
| Trusted-root duplicates/unknown namespaces abort (Q-L1-7) | `TestLoadDuplicateIDAborts`, `TestLoadTrustedFailuresAbort/unknown_namespace`, `TestPatternRejectedDecoyStillAbortsDuplicate` |
| Pattern policy: versioned artifact, dual corpus, fail-closed load (Q-L1-2) | `TestShippedPolicyCorpus`, `TestLoadPolicyFailsClosed`, `TestLoadRequiresValidatedPolicy` |
| Exemption ≠ authorization; one pattern × one instruction × one task (Q-L1-2) | `TestExemptionDoesNotAuthorize`, `TestExemptionDoesNotShortCircuitOtherPatterns`, `TestExemptionValidationFailsClosed`, `TestExemptionContentPinMismatch` |
| No runtime data alters resolved instruction text (Q-L1-4) | `TestRenderGolden` (verbatim bodies; no templating API exists), `TestLoadEdgeCasesPinned` (byte-exact bodies) |
| EIS immutable value; succession not mutation (Q-L1-5) | Value semantics by construction; `TestResolveByteChangeSensitivity` (stability); epoch boundaries are L7-deferred (tasks.md §7) |
| Final rendered-EIS pass: detect and refuse, never rewrite (Q-L1-3) | `TestRenderFinalValidation` (size, secret, untrusted adjacency), `TestRenderPolicyBinding` |
| What was scanned = what the model received (Q-L1-3/Q-L1-5) | `TestDeliveryProofMockProvider`, `TestSystemMessage` |
| Role: protected instruction file, renderer owns furniture only (Q-L1-8) | `instructions/global/system/role.md` + `TestShippedContentResolvesAndRenders` (role leads), `TestRenderGolden` (headings only) |
| Secret scan is a capability boundary (Q-L1-1/DAY-0 §3) | `TestLoadTrustedFailuresAbort` + `TestLoadUntrustedFailuresAbort` secret cases (both directions), `TestSecretScanMustPass` |
