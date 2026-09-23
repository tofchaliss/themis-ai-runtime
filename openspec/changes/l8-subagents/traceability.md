# Layer 8 Traceability — decisions → mechanisms → registers → tests

Tests in `src/harness/subagents/delegation`, `subagents/delegation/seam`,
`orchestration`, `tools`, `context`, `instructions`, `state`,
`deployment`, `runtime/model`, `verification/seam`. Decisions from
`design.md` §2 (D-L8-1..21) and §6 (C-L8-1..21); registers from §5.5
(A admission/authority · B positive path first · C record · D static
boundedness · E live). Gate 1 classifications and gaps in
`RESUME-HERE.md`; milestone evidence in `tasks.md`.

| Decision / invariant | Mechanism | Register | Evidence |
| --- | --- | --- | --- |
| D-L8-1 — a delegated reasoning execution, not a delegated authority; result = untrusted advisory | result re-enters framed under the `delegate` capability's registered `external-untrusted` trust (`FrameToolResult`), or as a closed typed error; no fact kind for model-authored bytes | B, C | `TestDelegationPositivePath` (frame header `authority: external-untrusted`, capture never shown), `TestClassDerivationIsTotalAndFloorBound`, `TestConversationProjectionAfterDelegation` |
| D-L8-2 — one isolated, tool-less, single-call execution; no workflow/phase/L4/L5/stream/task of its own | seam issues exactly one `Model.Execute` with no tools; the delegation package holds no state machine or transition type; identity = (task_id, l8 seq) | A | `TestSeamIsNotASecondL7` (one Execute, not in a loop, `Model:` = request), `TestDelegationPackageWall`, `TestEventBodyReproducible` (no delegation id) |
| D-L8-3 — model-requested through the L4 gate; injected one-way seam; paired re-entry | `delegate` registry-v5 capability; `Config.Delegator` (Registered / Instantiate / Delegate); loop post-hook appends exactly one tool-role message paired by ToolCallID and never steps δ | A, B | `TestAuthorizeDelegate`, `TestDelegatorInterfaceCarriesNoConversation`, `TestDelegationBranchNeverSteps`, `TestL8LeavesWorkflowVocabularyUnchanged` |
| D-L8-4 — selection authority only: `template` + `evidence` + `brief`; closed world; harness composes | L4 closed params (unknown key = `unknown-field`), exact `name@version` target, evidence shape; seam composes via L1 Resolve + L2 Gather/Compose | A, B | `TestAuthorizeDelegate` ("model requests a scope", "instructions"), `TestDelegationOutsideScopeIsAnL4Denial`, `TestDelegationPositivePath` (composition object = exact input) |
| D-L8-5/6 — template = distinct governed family; disjointness; grant `template_scope` is the only link | `delegation.LoadRegistry`/`LoadTemplate` (closed schema, disjoint keys refused by name, carry filter, pins, brief-slot cross-checks); no write API | A | `TestRegisteredTemplateLoads`, `TestTemplateRefusals` (33 cases), `TestRegistryRefusals`, `TestParseRefExactOnly`, `TestResolveRefusals`, `TestAppendOnly`, `TestNoRegistryWriteCapability` (+ self-check), `TestInstantiationSurfaceHasNoTemplateScope` |
| Q-L8-7 — anchor pins the template registry | `Anchor.DelegationTemplateRegistry` (sha or `absent`); `verifyAnchoredInstructionPlane` ties it to `Config.DelegationRegistryPath`; `themis-run` builds the seam over the same path | A, B | `TestAnchoredDelegationRefusals` (pin/no path, absent/path, bytes ≠ pin, withdrawn in scope), `TestAnchoredDelegationPositivePath` |
| D-L8-8/17/19 — dedicated `l8-delegation` witness; template bytes durable; G2 row `l8_delegation_record` | `state.EvL8Delegation`; `delegation.Event` closure + canonical encoding; seam stores manifest/contract/instruction + composition + output objects and references them | C | `TestL8DelegationEventClass`, `TestEventClosure`, `TestDelegationPositivePath` (refs, ordering, window purity), `TestReconstructionConfirmedFromRecordAlone` |
| D-L8-15 — stage-indexed failure: A denial · B refusal in the executor · C outcome · D invariant · E orphan | Authorize (A); `execDelegate` → `delegation-refused:<reason>` in `l4-audit{error}` (B); seam outcomes provider-error / output-over-bound / model-identity-mismatch (C); seam error → `ErrInvariant` (D); fault points (E) | B, C | `TestDelegationOutsideScopeIsAnL4Denial`, `TestDelegationStageBRefusals` (7), `TestDelegationPostInstanceOutcomes` (3), `TestDelegationFaultPoints` (3), `TestDelegateRefusesDoctoredCapture` |
| D-L8-16 — no budget authority; every resource charged to the parent; output captured whole, never truncated | quota consumed at proposal; ctx = min(turn timeout, remaining wall); output stored whole then bounded for re-entry | B, D | `TestDelegateQuotaCountsEveryAttempt`, `TestDelegationPostInstanceOutcomes/output over bound` (stored whole), `TestStaticExecutionBoundWithDelegate`, `TestParentTurnDeadlineIsMinOfTurnAndWall` |
| D-L8-18 — never verifier-eligible; observed, never evaluated | registry-v5 `delegate` not verifier-eligible; L10 `HistoryView.Delegations` read-only | A | `TestRegistryV5DeclaresDelegate`, `TestL10HistoryObservesDelegation` |
| D-L8-20 — no hierarchy; no delegation id | delegated call has no capability interface (no tools offered); identity = (task_id, seq) | A, C | `TestSeamIsNotASecondL7`, `TestEventBodyReproducible` |
| C-L8-4 — EIS carry: roots unconditional, optional scopes only by filter, template instruction the one added source | seam source filter; `instructions.ActivateDelegationSource` (`skill.delegation`) | B | `TestDelegatedEISCarryFilter` (×3), `TestTemplateRefusals` (root/task/unknown/duplicate carry) |
| C-L8-5 — evidence = `<seq>:<id>` re-established from the parent's stream; class from the witnessing event | `ParseEvidenceRefs` shape at L4; seam: event exists, Refs carry the object, `deriveClass` (l4-audit → trust under matching registry hash; model-turn/l8 → floor) | A, B, C | `TestEvidenceRefParse`, `TestDelegationStageBRefusals` (unreachable ×4, duplicate), `TestClassDerivationIsTotalAndFloorBound` (+ AST: no store read, no `Refs`) |
| C-L8-6 — evidence is a set; canonical by seq; no dedup; permutation invariant | `delegation.SortEvidence` (duplicates refused); L2 orders by (Kind, Hash) across sources | C | `TestEvidencePermutationIsInvariant` (equal-hash items; ArgsHash differs, composition identical) |
| C-L8-7 — bounded by constants L8 does not own; lazy L6-object source | `context.KindRecordObject` (fetch + address verify in `collect()`), `MaxEvidenceRefs`, P-L8-1 ≤ `MaxItemBytes` | A | `TestRecordObjectSourcesFillOneSlot`, `TestProviderCeilingWithinItemCap`, `TestResponseCeiling` |
| C-L8-8 — `evidence.seq < parent_call_seq < l8 seq`; empty window | executor runs before the audit commits; `Event.Validate`; reconstruction asserts the window | C | `TestDelegationPositivePath`, `TestEventClosure`, `TestReconstructionTypedFailures/event inside the window` |
| C-L8-9 — reconstruction from the L6 closure; live registry admits, record reconstructs | `seam.ReconstructDelegation` (CONFIRMED / UNREPRODUCIBLE / DISCREPANCY, discrepancy outranks); `ComposeWithSystem`, `ParseTemplate`, `ParseContract` | C | `TestReconstructionConfirmedFromRecordAlone` (after withdrawal, deletion, root change), `TestReconstructionTypedFailures` (missing object, doctored payload/template hashes) |
| C-L8-10 — governed vs execution model identity; `Reported` | `Identity.Reported` (P-L8-2); `model_identity{governed, execution}`; stage C `model-identity-mismatch` | B, C | `TestResponseCeiling` (reported ≠ requested observable), `TestDelegationPostInstanceOutcomes/model identity mismatch`, `TestModelTurnRecordsExecutionIdentity` |
| C-L8-11 — re-entry framed under registry trust / closed typed errors, unframed | `walk.delegate`; `tools.KnownErrorClass`; `DelegationRefused` cannot mint an open class | B | `TestDelegationErrorVocabularyClosed`, `TestDelegateExecutorIsInstantiation`, `TestConversationProjectionAfterDelegation` |
| C-L8-12/13 — instantiation in the executor; capture vs re-derivation; refusals reconstructable | `execDelegate` → capture; seam `Delegate` re-derives and refuses a disagreeing capture; F-L8-3 `PreResolve` for the L10 seam | B, C | `TestDelegateRefusesDoctoredCapture`, `TestDelegationPositivePath` (capture = witness), `TestVerificationRefusalIsNotAnOutcome`, `TestPreResolveMirrorsStageOne` |
| C-L8-14 — registration establishes admissibility only; review checklist; brief slot rules | `policies/delegation/README.md`; loader cross-checks; `themis-preflight` template verification | A | `TestTemplateRefusals` (brief slot missing/withheld/class), preflight PASS on the bundle |
| C-L8-15/16 — exact `name@version` scope at L4; `template_scope` fixed-by-skill, digested | `Authorize` exact equality; `grantAuthorityDigest` includes `TemplateScope`; assembly validates entries resolve | A | `TestAuthorizeDelegate` (`@2 ∉ scope`, prefix, floating), `TestGrantAuthorityDigestCoversEveryAuthorityField` (scope gained/swapped/emptied), `TestDelegationAssemblyRefusals` |
| C-L8-17 — nothing enters by existing; the seam never sees the conversation | interface carries no `model.Message`/`ExecutionResponse` (reflection wall); brief inline as `external-untrusted` | A, B | `TestDelegatorInterfaceCarriesNoConversation`, `TestDelegationPositivePath` (brief in its slot), `TestDelegatedEISCarryFilter` |
| C-L8-18 — laundering impossible: class = f(witnessing event) | `deriveClass` total; every L7/L8-written class maps to the floor or a registry-derived value | C | `TestClassDerivationIsTotalAndFloorBound` |
| C-L8-19 — attempts consume quota; deadline = min rule; static bound | L4 counters; seam ctx; `executionBound` recorded as `l8_execution_bound` | D | `TestDelegateQuotaCountsEveryAttempt`, `TestStaticExecutionBoundWithDelegate` |
| C-L8-20 — identity minted by L6 at commit; no id field | `Event` has no id; identity = (task_id, seq) | C | `TestEventBodyReproducible` |
| D-L8-21 §3 — not a second L7 | walls: imports, one Execute, one AppendEvent literal, no goroutines, no model literal, no δ step, vocabularies unchanged, `eventClasses` +1 | A | `TestSeamIsNotASecondL7`, `TestDelegationPackageWall`, `TestDelegationBranchNeverSteps`, `TestL8LeavesWorkflowVocabularyUnchanged`, `TestL8DelegationEventClass` |
| P-L8-1 / P-L8-2 / F-L8-2 / F-L8-3 / F-L8-4 (prerequisites) | response ceiling, reported identity, model-turn identity, PreResolve audit annotation, min deadline | — | `runtime/model/ceiling_test.go`, `TestModelTurnRecordsExecutionIdentity`, `TestVerificationRefusalIsNotAnOutcome`, `TestParentTurnDeadlineIsMinOfTurnAndWall` |
| Register E — live | `TestLiveDelegationWalk` (anchored @2, qwen2.5:7b): admitted, typed terminal; the model did not delegate | E | recorded 2026-09-23 as model behaviour; delegation-through-a-live-model not evidenced |

Reviews: (pending — three Class-3 reviews in isolated worktrees against
71a7188; findings and dispositions to be recorded here and in
`tasks.md` §7).

Verdicts: (pending owner acceptance) architecture-conformant · test-
evidenced · operationally-proven — the last is TEST HARNESS ONLY
(anchored positive path under a test-harness anchor with the real
governed artifacts; `rsys@4` and a production `themis-run` are the
owner's acts).

Residuals: tool-capable L8 · δ-declarable delegation event · parallel
fan-out · per-target quotas · caller-narrowed `template_scope` ·
phase-level capability parameters · parent-loop handling of reported
model mismatch · provider version/digest governance · approval channel
· per-item `derived_sensitivity` (floor rank) · registry-less
reconstruction (class from the witness, registry reported missing) ·
`remediate-dependency@1` cannot instantiate (reserved H2 headings) ·
live model did not exercise delegation · the record-ref furniture
shows tool results only (model-turn objects are not referenceable by
a model today).
