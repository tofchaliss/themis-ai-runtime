# Layer 4 Traceability — grill invariants → tests

All tests in `src/harness/tools`. Invariants from `design.md` §3 (owner-locked closures).

| Invariant | Evidence |
| --- | --- |
| ExecutionGrant ⊆ Registry at use; grant cannot widen vocabulary (Q-L4-1) | `TestSeamAndScopeEdges` (granted-but-unregistered ⇒ bare not-available), `TestDispatchCompleteness` (table pruned to registry; phantom entry fails closed at startup). At-use vs at-load enforcement recorded as deliberate (registry.go) |
| Closed vocabulary: absent = does not exist; mutating verbs absent not denied (Q-L4-1) | `TestDispatchCompleteness` (run_command/write_file/apply_patch/git_commit not in table); shipped registry = 5 read-only tools |
| No inference from names/requests/task text (Q-L4-1) | Structural: `Authorize` reads only (registry, grant, request, state) — pure signature |
| Authorization precedes argument validation — anti-oracle (Q-L4-5 §3) | `TestAuthorizeDecisionTable`: unknown tool with invented authority field, not-granted with bad args — both `not-available` with zero detail; security review verified no arg byte read before availability |
| Four denial classes; unknown ∪ not-granted ∪ quota → not-available, zero detail (Q-L4-5) | Decision-table rows for all six states; `not-available` detail asserted empty |
| Scenario-3 regression: authority-disposition args die at schema | `TestAuthorizeDecisionTable/scenario-3` (`requires_human_decision` ⇒ invalid-args unknown-field, whole call); `TestRegistryFailsClosed/authority-disposition param` (same name in a schema dies at load); widened stem list (F5) |
| No self-declared derived trust (Q-L4-3) | `TestRegistryFailsClosed/self-declared derived` (branch-asserted: error must name derived) |
| Target: model supplies request, grant supplies binding, no override (Q-L4-2) | `TestAuthorizeDecisionTable` (escape ⇒ target-refused with bounded echo), `TestHandlePipeline` step 6 (symlink via tool path delivers nothing), `TestSecurityReviewRegressions` (relative binding refused at load; missing binding ⇒ not-available) |
| invalid-args(field)/target echo bounded; no topology/policy leakage (Q-L4-5) | Decision-table detail assertions; echo bounding in `TestRemainingExecutorsAndEdges`; security review verified no host path/threshold in any model-visible byte |
| Denials deterministic within a snapshot (Q-L4-5 §7) | `TestAuthorizeDecisionTable` (DeepEqual repeat) |
| Quotas in the grant; L4 stateless, CallState supplied (Q-L4-4/7) | Quota rows in the decision table; security review verified `Handle` never mutates state. **L7 obligation recorded:** the loop must increment CallState between calls |
| Registry timeout enforced per call; overrun = typed `timeout` error | `TestTimeoutEnforced` (goroutine-linger recorded as L5 deferral) |
| CallState immutability (Q-L4-7) | `TestCallStateImmutable` |
| Typed executor errors, never denial-shaped, never silent (Q-L4-8) | `TestHandlePipeline` step 4, `TestRemainingExecutorsAndEdges` (seam error); F4 regression (empty-but-hidden search refuses) |
| Results are classified evidence with capability-fetch mechanism (Q-L4-3, Q-L2 seam) | `TestHandlePipeline` steps 1–2 (trust from registration, hash, mechanism); `TestProviderRoundTrip` (full provider loop through the real seam); `TestMockProviderLoop` (ToolCallID binding) |
| Audit event every path, jointly reconstructable (D-L4-8, F2) | `TestHandlePipeline` audit assertions; `TestSecurityReviewRegressions` (denial audit carries requested target + exact model detail; drift path audits); timing/executor-id explicitly deferred to L6 |
| Fail-closed artifacts | `TestRegistryFailsClosed` (20 cases, per-branch assertions), `TestGrantFailsClosed` (7 + missing-file) |
| Operational proof (Q-L4-9) | Mock half: `TestMockProviderLoop` + `TestHandlePipeline` ✓. **Live half PENDING**: qwen2.5-coder:7b failed the tool-protocol smoke (emits tool JSON as content, structured tool_calls null — recorded evidence 2026-09-06); a tools-capable model pull awaits owner authorization; content-as-action parsing rejected on principle |
