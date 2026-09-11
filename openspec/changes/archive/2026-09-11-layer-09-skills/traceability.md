# Layer 9 Traceability — grill decisions → mechanisms → registers → tests

Tests in `src/harness/skills` and `src/harness/orchestration`. Decisions from
`design.md` §2 (D-L9-0..17 + amendments D-L9-11a/b/c/d, R-L9-1, R-L9-2,
ADG-L9/L6-1), registers from §4 (A structural · B adversarial · C equivalence ·
D reconstruction · E live). Review record in tasks.md M1–M5.

| Decision / invariant | Mechanism | Register | Evidence |
| --- | --- | --- | --- |
| D-L9-0 — L9 is a pre-submission plane; no runtime, no second orchestrator; L7 alone executes | `skills/instantiate.go` produces an ordinary governed envelope; no walker exists in the package; `go list -deps ./orchestration` shows zero dependency on `themis/skills` | A | package API closure; M5 task "L7 remains Skill-blind" (dependency scan) |
| D-L9-1 — Skill = governed hash-pinned atomic composition; inert data; registry NOT pinned | `skills/manifest.go` fail-closed loader: closed schema, seven mandatory pins, hash-format validation; pins resolve through `confine.ResolvePath` | A | `TestManifestClosedSchema`, `TestManifestPinsMandatory`, `TestManifestPinPathRefusals`, `TestManifestSymlinkEscapeRefused`, `TestPinSymlinkEscapeRefused` |
| D-L9-2 — Procedure is a composition concept, never a second runtime entity | procedure enters as a pinned artifact reference (L1 owns instruction identity); no standalone procedure registration surface exists | A | manifest schema audit (no procedure-registration field); `TestSkillProcedureFieldsPaired` |
| D-L9-3 — author ≠ governance ≠ machinery; registration is the owning act; machinery never self-registers | catalog has no write API; `investigate-cve@1` authored as `policies/skills/catalog.proposed.json`, registration an owner act | A | `TestNoCatalogWriteCapability` (AST audit, hardened twice against rename and import-alias bypass), `TestNoCatalogVerbInRegistries` (decoded registry scan) |
| D-L9-4 — zero new interpreters; closed input-schema vocabulary; no procedure templating | `skills/schema.go` bounded structural validator; rich features unrepresentable via DisallowUnknownFields; procedure bytes immutable review→delivery | A, B | `TestSchemaVocabularyClosed`, `TestValidateClosedVocabularyBranches`, `TestSchemaShapeRefusals`, `TestCallerInputNeverTouchesProcedure` |
| D-L9-5 — skills narrow tool reachability; quotas caller-narrowable upper bounds only | grant template ⊆-checked; instantiation permits downward integer narrowing only | B | `TestQuotasNarrowOnly`, `TestWallDeadlineNarrowsOnly`, `TestGrantShapeRefusals` |
| D-L9-6 — workflow + ceiling frozen at review; one Skill = one workflow | workflow not an instantiable field; byte-identical pin to L7 | A, B | `TestInputSurfaceClosed` (substitution into workflow/contract/ceiling unrepresentable), composition-hash checks in `TestP0SkillCompositionCoheres` |
| D-L9-7 — closed four-class instantiation surface; unknown fields refused | `skills/instantiate.go` closed field list; Class-2 narrowing, Class-3 schema-validated external-untrusted inputs; `@input.<field>` deliberately NOT implemented (recorded v1 narrowing, tasks.md M2) | A, B | `TestInputSurfaceClosed`, `TestTemplatePlaceholdersRequired`, `TestP0SkillInputSchema`, `TestTraversingTaskIDRefused`, `TestRepoTraversalRefused`, `TestPinnedSHAMatchesL5Rule` |
| D-L9-8 — the model may author, never admit; four structural walls; instantiation external to the walk | catalog-only resolution; catalog root disjoint from task-writable roots; registered composition hash is the sole identity | A, B | `TestUnregisteredCopyIsNotTheSkill` (byte-identical copy ≠ identity), `TestCatalogRootDisjointness`, `TestDisjointnessRequiresEveryWritableRoot`, `TestNoCatalogWriteCapability` |
| D-L9-9 — skill output gains zero authority; provenance is evidence weight, never class promotion | attribution carries identity only; no terminal vocabulary change (no COMPLETED_UNDER_SKILL_X) | C | `TestSkillProvenancePreservedIntoAttribution`, `TestP0ProcedureClaimsNoAuthority` |
| D-L9-10 — immutable name@version→hash; append-only catalog; exact pins; TOCTOU-safe withdrawal | `skills/catalog.go`: two-way identity agreement, rebind refusal, active\|withdrawn, atomic resolve-verify-instantiate, `CheckAppendOnly` | A, B | `TestCatalogRebindRefused`, `TestCatalogAppendOnly`, `TestNoFloatingReferences`, `TestWithdrawnRefusesInstantiation`, `TestCompositionHashMismatchRefused` |
| D-L9-11 — running tasks immutable w.r.t. their Skill; resolve-once; mid-walk mismatch = invariant failure | resolve → byte-verify → fixed execution representation; no catalog arrow during the walk | B, D | `TestTamperedArtifactRefused` (resolution), `TestCompositionArtifactMismatchIsRefused` (freeze → seal → FAILED) |
| D-L9-11a (C2) — L9 sole semantic resolver; composition manifest + hash cross the seam; L7 verifies materialized identity, stays Skill-blind; execution integrity ≠ governance identity, never collapsed | compact manifest (identities only) in the ordinary envelope; L7 verifies every materialized Skill artifact against the committed identity | D (split per the amendment) | (a) REFUSED: `TestCompositionArtifactMismatchIsRefused`, `TestUnsealedOrAlteredCompositionIsRefused`; (b) DETECTABLE ONLY: `TestGovernanceIdentityIsNotEstablishedBySelfDeclaredAttribution`, `TestAttributionInconsistencyIsDetectable` |
| D-L9-11b — composition self-hash: pins sealed as one unit | self-sealed commitment; matched-but-foreign path+SHA pair fails | D | mutation-verified binding (tasks.md M5); `TestUnsealedOrAlteredCompositionIsRefused` |
| D-L9-11c — commitment binds ALL Skill-fixed material incl. input_schema; excludes registry + exec_ceiling | commitment scope fixed in design, enforced by producer | D | `TestInstantiationEmitsASealedCommitment` (producer deletion fails), per-artifact coupling mutations (M5 closure requirement 2) |
| D-L9-11d — skill attribution REQUIRES a commitment; not submitter-elective | envelope-contract coherence matrix enforced at L7 admission | C, D | `TestSkillAttributionRequiresACommitment`, `TestAnySkillAttributingKeyRequiresACommitment` (C-1 blocker remediation), `TestRefusedCompositionLeavesNoDurableRecord` |
| D-L9-12 — no skill references; recursion unrepresentable; lineage ≠ authority | manifest has no skill-reference field | A | `TestNoSkillReferenceField` (schema audit) |
| D-L9-13 — envelope is the seam; L7 skill-blind; provenance preserved never interpreted; zero trust discount | additive non-semantic envelope amendment (`skill_procedure_path/sha256` + opaque origin) | C | `TestSkillProvenanceDoesNotChangeTheWalk` / `...ASaturatingWalk` / `...ATurnSaturatingWalk` (transition-tuple equality via the Register-D replayer), `TestProvenanceSwapDoesNotRedirectExecution` (owner's closing attack), `TestSkillEnvelopeGetsNoTrustDiscount` (L9 validator bypassed, L7 refuses on its own) |
| D-L9-14 — verification: skill-required structure only; no v1 output receives derived authority | no verification mechanism in L9; v1 honesty recorded (run_command is OPEN-2) | — | structural (no mechanism exists to test); residual 10 carries the L10 dependency |
| D-L9-15 — approval unrepresentable in v1; enforced by unreachability | no approval field in the manifest; approval-vocabulary workflows already unloadable (Q-L7-9) | A | manifest closed schema; L7 loader refusals (archived L7 traceability) |
| D-L9-16 — two gates (registration ≠ ratchet evaluation); no auto-promotion; no reduced review | no harness path from evaluation results to catalog mutation (no write API at all) | A | `TestNoCatalogWriteCapability`; the ratchet side is L11's grill (residuals 8, 9) |
| D-L9-17 — five registers, investigate-cve@1, unmodified production loop | P0 bundle at `policies/skills/investigate-cve/` (two-phase lattice: ANALYZE read-only, ASSESS adds write_file) | E | `TestP0SkillBundleIsConsistent`, `TestP0SkillCompositionCoheres`, **`TestP0SkillRunsThroughProductionLoop` + `TestLiveSkillWalk` PASS vs qwen2.5:7b through the unmodified L7 loop** |
| R-L9-1 — recorded identities originate from L7-materialized bytes, not envelope claims (closed for the four L7-materialized artifacts) | L6 derives object addresses by hashing bytes itself; L7 stores the bytes it verified | D | `TestRecordedIdentitiesAreComputedNotCopied`, `TestMaterializedArtifactsAreRecoverableFromTheRecord` (six independent mutations fail) |
| R-L9-2 — grant + procedure bytes durable; single-home = semantic ownership, not physical durability (BodyHash L1 ≠ ObjectID L6) | verified bytes stored as L6 evidence-payloads inside the no-reopen window (TOCTOU constraint honored) | D | `TestProcedureBytesAreDurableAndCrossLayerConsistent` (BodyHash(bytes_A)==recorded ∧ ObjectID(bytes_B)==recorded ∧ bytes_A==bytes_B, cold), `TestGrantBytesAreDurableAndDistinctFromEffective`, `TestDurableBytesMustMatchWhatWasLoaded` (four mutations fail) |
| ADG-L9/L6-1 — option (a): existing L6 primitives, no new object class, no second durability path | chain L9 resolves → L7 materializes → L6 StoreObject → Event.Ref | D | `TestColdReconstructionOfExecutedComposition`, `TestProcedureIdentityIsVerifiedAgainstMaterializedBytes` |

Reviews: three Class-3 reviews (security, test, architecture) run at M1, M2, M3,
then **re-run in full against the final implementation** after M5 and
R-L9-1/R-L9-2 (2026-09-08, at 7c43335; the earlier verdicts were correctly
discarded): architecture CONFORMANT with 1 HIGH, security 2 HIGH + 2 MED, test
1 CRITICAL + 1 HIGH + 1 MED — every CRITICAL and HIGH remediated with a
mutation-verified regression (final remediation commits 75e2cfa, e344ce8).
Coverage: skills 84.8% · orchestration 82.4% · instructions 95.2% · state
83.7%. Live proofs re-run green after every remediation.

Methodological evidence (owner, recorded as part of the archive, not a
footnote): three review rounds each found defects that passing tests concealed;
in every round the decisive tool was mutation testing. Green tests are the
beginning of the evidence question, not the end.

Verdicts: architecture-conformant YES · test-evidenced YES ·
operationally-proven YES — **accepted by owner 2026-09-11** (in-session,
closing the pause recorded in RESUME-HERE.md).

Residuals recorded (design.md §3, never silently promoted): governed
multi-Skill pipelines (L8-vs-L9 OPEN) · L8 delegation · runtime
revocation/cancellation · approval channel · repository/external skill
activation · model-property predicate · richer input schema ·
evaluation-purpose attribution (L11/L6) · proposal lifecycle (L11) ·
deterministic verification capabilities (OPEN-2/L10) · expected-outputs
contract · C3 governance actor attestation · the event-class boundary (a third
meaning on the delivery class forces an L6 vocabulary revisit).
