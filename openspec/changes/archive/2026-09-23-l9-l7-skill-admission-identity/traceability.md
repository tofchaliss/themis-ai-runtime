# Skill-admission identity traceability — decisions → mechanisms → registers → tests

Tests in `src/harness/orchestration`, `src/harness/tools`,
`src/harness/execution`, `src/harness/deployment`, `src/harness/skills`,
`src/harness/internal/strictjson`. Decisions from `design.md` §2
(D-SA-1..10), drill record §6 (A-SA-1..11), whitelist §5. Review record
in `tasks.md` §6. Registers: A structural · B adversarial · C
equivalence · D reconstruction · E live.

| Decision / invariant | Mechanism | Register | Evidence |
| --- | --- | --- | --- |
| D-SA-1 — L7 identity-only correspondence is authoritative; L9 semantic check is non-authoritative (D-L9-13 amended) | `orchestration.verifyAnchoredSkill` resolves by `env.Skill` from the anchor-pinned catalog and compares identities only; L9 `Instantiate` emits, never admits | A | `TestAnchoredSkillPositiveTwin`, `TestActivateSkillSourceSingleCallSite`; L9 amendment record `archive/2026-09-11-layer-09-skills/amendments/skill-admission-identity/` |
| D-SA-2 — seven-member equality against the manifest pins; a caller's composition is never authority | per-member loop over workflow / workflow_ceiling / context_contract / grant_template / spec_template / input_schema / procedure, each its own refusal ("never its constituent hashes") | B | `TestAnchoredSkillNegativeTwins` (neg-b, b2, b3, c, c2, c3, c4 — all seven), `TestAnchoredMixedBundleRefused` (A-SA-2 tuple swap), Phase C row C17 (`evidence/harness/phasec`) |
| D-SA-3 — anchored refusal of an unattributed `skill_procedure_path` (closes F-SA-1) | `SubmitTask` anchored: `skill == "" && skill_procedure_path != ""` ⇒ `ErrAssembly` | B | `TestAnchoredUnattributedProcedureRefused` |
| D-SA-4 — effective ⊑ template: tool set / mutating / themis_scope / template_scope EQUAL; quotas, total, deadline narrow; workspace and task_id governed bindings; reference template from the manifest pin | `tools.Instantiates` (L4 vocabulary), `execution.SpecInstantiates` (L5), applied at L7 after Claim 1 with `Manifest.ResolvePin`; mirrored at L9 after `checkGrantShape` | B | `TestInstantiates` (unit table, one mutation per clause), `TestSpecInstantiates`, `TestLoadGrantTemplateScopeRules`, `TestAnchoredSkillNegativeTwins` (neg-d..m: per-tool quota, total, tool added, tool removed, template_scope, mutating, themis_scope, literal workspace, case-variant key, spec deadline, spec limit) |
| D-SA-4 as built — the relation runs on submitted bytes before assembly binds, so workspace is `@workspace` or absent on both sides; a literal path is refused at parse; assembly asserts post-load that every bound workspace is its own root (security CRITICAL-1, 2026-09-23) | `internal/strictjson.Check` (exact lowercase keys, no duplicates) before every grant/spec decode at L4 `LoadGrant`/`parseGrantShape`, L5 `parseSpec`/template, L7 `instantiateGrant`, L9 `checkGrantShape`; `instantiateGrant` entry-key allowlist + `Workspace == wsRoot` assertion | B | `TestCheck` (strictjson), `TestGrantKeyWallAtAssembly` (case variant, duplicate, unknown key, positive twin), `TestInstantiateGrantPostBindAssertion`, `TestLiteralWorkspaceRefused`, unit cases "literal workspace refused", "case-variant key refused", "duplicate key refused" |
| D-SA-5 — first-class `skill` selector; `origin` attribution-only; coherence matrix | `Envelope.Skill` with exact `name@version` syntax; `LoadEnvelope`: any skill-attributing origin key ⇒ `skill` + commitment required, `origin["skill"] == skill`; L9 emits `skill` | A, C | `TestSkillFieldCoherence`, `TestInstantiateEmitsSkillSelector`, `TestSkillAttributionRequiresACommitment`, `TestAnySkillAttributingKeyRequiresACommitment` |
| D-SA-6 — the seal has exactly two consumers (producer + L7 `verify`); no Governance consumer | `.Seal` reads confined by type-checked wall (go/types, not textual) | A | `TestSealHasNoGovernanceConsumer` |
| D-SA-7 — unanchored = a different contract, Claim 1 only; the record must not read as a verified composition | `governed["skill"]` only when the manifest resolved; unanchored records `skill_claimed` (security LOW-1) | C, D | `TestGovernanceIdentityIsNotEstablishedBySelfDeclaredAttribution`, `TestAttributionInconsistencyIsDetectable`; `TestAnchoredSkillPositiveTwin` asserts `skill` + `origin:skill` + `deployment_anchor` |
| D-SA-8 — withdrawal is forward-only on the identity; historical records stay interpretable | catalog `Resolve` refuses withdrawn; anchor pins the catalog state | B | `TestAnchoredWithdrawnSkillRefused` |
| D-SA-9 — anchor `skills[]` allowlist checked before catalog resolution (ladder order) | `deployment.Anchor.Skills` (exact refs, unique, ≤64); `SubmitTask` anchored: allowlist gate after the instruction plane, before the registry pin and the bundle loop | A, B | `TestSkillAllowlistRules`, `TestAnchoredSkillAllowlist`, `TestAnchoredSkillRequiredConfiguredCatalog` (ladder order), `TestAnchoredSkillCatalogMismatchRefused` ("skill catalog is not the anchored artifact", single load vs pin — architecture MED-1) |
| D-SA-10 — the anchor selects, never composes | no composition field on the anchor; the reference template comes from the manifest pin by member name only | A | `TestReferenceTemplateResolvedByMemberNameOnly`, anchor closed schema (`deployment` loader) |
| A-SA-11 / Q-SA-12 — the positive twin: a genuine L9-instantiated skill is ADMITTED and EXECUTED under an anchor that lists it; Claim-2 bytes retained | `materialized["skill_catalog"]`, `materialized["skill_manifest"]` (`Catalog.Raw`, `Manifest.Raw`) | D, E | `TestAnchoredSkillPositiveTwin` (COMPLETED; object ids = `sha256:` of the governed files), **`TestLiveAnchoredSkillWalk` PASS 2026-09-22 and 2026-09-23 (qwen2.5:7b, COMPLETED/VERIFIED, test-harness anchor — the first anchored skill execution)** |

Reviews: three Class-3 reviews (architecture, security, test) run in
isolated worktrees against `dc8034c` (2026-09-22/23). Security found one
CRITICAL (case-variant key past an exact-key guard) — remediated with the
strictjson wall and re-verified by the full suite and the live anchored
walk; every HIGH/MED remediated or dispositioned in `tasks.md` §6.
Checkpoints: `dc8034c` (SA-M1..M5), `7be7116` (M6 remediation).

Verdicts: architecture-conformant YES · test-evidenced YES ·
operationally-proven **test harness only** — the live proof ran under a
test-harness anchor with the real catalog; a production run under
`rsys@4` awaits the owner's Governance act.

Residuals (never silently promoted): `rsys@4` act listing `skills[]` ·
production `themis-run` · scripts printing `skills[]` · Claim-2
reconstruction VIEW (L10) · template-bytes retention (LOW-4, owner) ·
fault sweep at the new store points · exact-case key wall on the other
loaders (registry, ceiling, anchor, envelope, catalog/manifest — no
exact-key guard upstream, no known bypass) · L9 mirror has no reachable
negative (D-L9-5 refuses widening first) · live-test opt-in gating ·
F-SA-2 → L11 · C3 governance actor attestation.
