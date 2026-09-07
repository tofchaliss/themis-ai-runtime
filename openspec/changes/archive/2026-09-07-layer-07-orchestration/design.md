# Design: Layer 7 — Orchestration

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §10 (layer doc), archived L1–L6 designs (their L7-addressed IOUs land here), `ARCHITECTURE.md`, the existing `Server.invoke`/`Router` fragments (code map: EVOLVE).
**Decision IDs** `D-L7-n`; **grill record** `Q-L7-1..12`, all CLOSED 2026-09-07. The grill record (§3) governs on conflict.

**Fold-completeness cross-check:** every original draft question maps to a closure — decision surface → Q-L7-1/2 · envelope authorship → Q-L7-10/D-L7-1 · turn/context lifecycle → Q-L7-12 (recovered orphan, the Q-L6-11 pattern) · budgets → Q-L7-4 · failure classes → Q-L7-7 · recovery/startup → Q-L7-8 · approval → Q-L7-9 · Router → Q-L7-10 · proof gate → Q-L7-11; the held completion-mechanism question discharged at Q-L7-5. No orphans remain.

## 0. Position in the flow

```
                 THEMIS GOVERNANCE
                        │
                        ▼
               Workflow Definition (immutable, hash)
                        │  ⊆ WorkflowCeiling ⊆ Registry
                        ▼
      Service adapter (transport only) ──► L7: Open · SubmitTask · ReadStatus
                        │
                        ▼
                 L7 deterministic δ
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
         L2            L4            L6
      context        capability    durable
      delivery         gate         record
          │             │             │
          └─────────────┼─────────────┘
                        ▼
                      Model — advisory reasoning; L4 is the sole effect gate
```

## 1. Hard invariants (inherited, not grillable)

- **L7 composes and paces; it never grants** — it instantiates or narrows governed ceilings and can never define one (the cross-layer principle, fourth application). L7 owns sequence, never selection.
- **Model output stays advisory through every iteration**: the loop converts model output into requests to gates; nothing accumulates trust across turns; model influence can narrow or end the walk, never extend or widen it.
- **A lower-level component never trusts the caller to have authorized** — L7 being the caller changes nothing; every gate re-checks.
- **Record-before-effect is L7's wiring duty** (D-L6-10), extended by Q-L7-12 to record-before-next-turn.
- **Deterministic security decisions never consume L6 state**; L7 reads only the StatusView vocabulary under declared, workflow-owned dependencies.
- **Fail closed**: unassemblable task never starts; unwritable record fails the task; typed outcomes only.

## 2. Locked decisions (post-grill fold, 2026-09-07)

### D-L7-1 — Governed lattice; the model's sole proposal channel already exists (Q-L7-1)
Every executable workflow originates from a Themis-governed, ⊆-bounded, per-task-immutable workflow definition (versioned, hashed, fail-closed loader; phase capability sets ⊆ WorkflowCeiling ⊆ Registry). The model has no plan/replan channel; plan-shaped prose is advisory data invisible to control. The model's only executable proposal channel is the existing L4 typed interface — bounded model proposals need no new mechanism, because building one would be a second gate (and parsing plans from content is the refused content-as-action pattern). Transitions fire on typed events and bounded counters, never model claims.

### D-L7-2 — No plan artifact: δ over (definition, cursor, typed event) (Q-L7-2)
L7 holds only {definition hash, derived position}. The next action is derived, never stored: `δ(definition-at-hash, cursor, typed event, counters)`. The cursor is a runtime projection of the durable event stream — reconstructable cold, never independent authority. Identical definitions + identical typed-event sequences ⇒ identical walks (provable, Register D). Any plan-shaped view is a re-derivable display projection excluded from the control path.

### D-L7-3 — Two-level control: constitution floors + explicit governed edges; δ is deliberately boring (Q-L7-3)
Floors (budget/deadline exhaustion, unrecordable state, corruption, invariant violations) live in the L7 constitution, outside the definition vocabulary — cannot be omitted, rerouted, or overridden. Within the definition's domain every behavior — advance, branch, stay/await, bounded retry, terminal — is an explicit edge over declared typed events and bounded counters. **Totality is verified statically at load**: full condition coverage, no overlap, exhaustion edge per counter; declared-event-at-δ-without-edge is an invariant violation, not a workflow state. Only declared events reach δ (the declaration is the filter; the declared vocabulary ⊆ the system's typed-event vocabulary). Terminals select only from the closed system termination vocabulary (→ L5 seal reasons → L6 lifecycle).

### D-L7-4 — Budgets: independent gates, monotonic narrowing, static boundedness (Q-L7-4)
Every L7 budget is a load-time monotonic narrowing of an upstream ceiling or a workflow-native counter ⊆ WorkflowCeiling. Narrowing is enforced by **independent gates, never budget-passing** — each owner counts at its own gate; first binding ceiling wins; "L7 raises a lower ceiling" is inexpressible, not forbidden. Budget values are load-time constants; runtime arithmetic only decrements; no replenish/reset. **Counter-free cycles refuse at load** ⇒ every valid workflow has a statically proven finite worst-case walk ("retry until deadline" is unwritable; the deadline is a floor, not a license). The one caller-supplied counter (CallState, the recorded L4 obligation) is monotonic by construction AND recounted from the L6 record by the verifier — prevention plus post-hoc detection. Pacing is L7's (when, incl. explicit stay) and is never free (wall floors drain) and never creates time.

### D-L7-5 — Model interaction: structural turn facts + gate outcomes; nothing else reaches δ (Q-L7-5)
δ accepts exactly two model-adjacent event families: harness-computed structural turn facts (closed: tool-calls-emitted, no-action, provider-error — facts about the turn's shape, authored by observation) and typed L4 gate outcomes, including registered workflow-control capabilities. Model content and claims never reach δ in any encoding; MODEL_REQUESTED_TERMINATION as a parsed event is structurally unproducible. What a no-action turn means (stay / advance / fail-after-counter) is a governed per-phase edge, never a universal rule.

### D-L7-6 — Control vocabulary: constitution-owned, verb-per-signal, zero semantic arguments (Q-L7-6)
Control verbs form a closed constitution vocabulary, declared in the registry under the `control` classification (a review flag, never an authorization branch — no `if control` anywhere in Authorize), gated by L4 identically to every capability, granted per phase by the definition. **Distinct signals are distinct verbs** — no reason/enum/free-text arguments (argument-borne signals would be content-adjacent-to-control, caller-owned sub-vocabulary, and grant-granularity destruction). Verb → fixed 1:1 typed signal (constitution); the definition's edges give the signal meaning; the verb means nothing, the edge means everything. Two-way ⊆ at load: control-flagged registry entries must exist in the constitution vocabulary and vice versa (dispatch completeness). **v1 ships exactly one verb: `declare_done()` → `PHASE_COMPLETION_REQUESTED`.**

### D-L7-7 — Failure taxonomy: outcomes, execution failures, invariant failures (Q-L7-7)
Three natures: (1) governed workflow outcomes and (2) declared typed execution failures belong to the definition's transition vocabulary; (3) orchestration-invariant failures belong to the constitution alone — **structurally undeclarable** (loader refuses definitions naming them; the L6 primitive-only-class mirror), unreachable by any edge, handled on one fixed path: typed invariant event → seal(fatal-breach) → unconditional teardown → FAILED. No new lifecycle vocabulary — differentiation by typed event and seal reason (the verdict/lifecycle orthogonality, one layer up). A workflow can never decide what happens when its enforcer is broken. Undeclared execution failures never reach δ — they flow to record + conversation (the model's problem, within budgets); a declared event may both fire an edge and remain model-visible.

### D-L7-8 — Startup closes the past before opening the future; resume is unproducible (Q-L7-8)
L7 startup reconstructs and classifies, never resumes: every discovered non-terminal record is driven to a typed terminal through L6 recovery **before any new work is accepted** — after startup, no non-terminal task exists, terminal streams refuse appends, identities are single-use: resume has no object, by reachability. New execution always = new task_id (+ `retry_of` lineage). CORRUPT records are surfaced typed and preserved untouched; L7 gains no cleanup/reconciliation/reopen/delete authority. **Deployment invariant: one orchestrator per state root** (the recorded cross-process residual, now with its operational rule).

### D-L7-9 — Approval is condition discharge, never authorization (Q-L7-9)
Approval is an external authority input represented solely as a Themis-governed, primitive-owned, durably recorded typed event, **bound to {task, position, requested-action identity}** — never a bearer token. L7 consumes it only to select an already-defined edge; approval can never widen a ceiling or create a capability; denial is a governed edge, not an error; the ingress is structurally non-model-reachable (the model may one day *request* approval via a control verb; it can never *produce* one). **v1: vocabulary reserved in the constitution; a definition declaring an approval gate refuses at load** (a gate without a channel is an eternal await). Recorded pressure point: no-resume + human approval latency ⇒ the approval channel's grill must confront resumable-await explicitly.

### D-L7-10 — The service boundary is transport, structurally (Q-L7-10)
L7's public seam is exactly `Open() · SubmitTask(envelope) · ReadStatus(task_id)` — API-closure-proven. The adapter decodes, authenticates where applicable, invokes the seam, serializes typed results — and **defaults nothing** (a defaulting adapter is a hidden envelope author). Prohibitions enforced by API closure + an import lint (adapter imports only the L7 seam — never tools/execution/state/context/instructions). The envelope is authored on the governance side, referencing governed artifacts; **assembly re-validates every containment fail-closed** (grant ⊆ ceiling, plan ⊆ contract, spec ⊆ execution ceiling, definition ⊆ ceilings, root disjointness at its designed call site, hashes into the record) — trusting the submitter for nothing. The legacy model-routing Router survives as an envelope-construction-time advisor outside the boundary (model selection is a fixed governed envelope input; per-turn routing does not exist). Duplicate submission is structurally safe (single-use ids). Transport serves StatusView only; audit is a separate governance-side path.

### D-L7-11 — Turn/context lifecycle: append-only phases, fresh composition at boundaries, record-before-next-turn (Q-L7-12)
Within a phase the conversation is append-only and immutable: one L2 composition per phase entry (durably recorded before model delivery), then model turns and tool results appended — no re-composition, editing, or silent trimming; **L7 never performs context re-management** (context pressure is handled only by the next declared phase's fresh L2 gather / L3 manage / L2 compose — new management, never re-management, honoring the L3 lock). Only what the next phase's governed context plan selects carries forward; earlier tool evidence is selectable by its durable classified identity. **Every model-turn output is durably recorded as an evidence-payload object referenced by a typed model-turn event** — exact bytes, model provenance, classification preserved — making the complete model-visible conversation at every turn cold-reconstructable. The recorded model output is durable history, never security truth, governed finding, workflow authority, or a transition condition. Ordering: compose → commit → deliver; model output → commit → next turn; L4 audit → tool execution → typed result → commit → model delivery. Nothing needed to reconstruct a later model-visible state exists only in loop memory.

### D-L7-12 — Retry is a new identity (unchanged from draft)
Re-running = new task_id + `retry_of`, new environment identity, fresh assembly. L7 never resumes, reconciles, or reuses.

### Constitution amendments required at implementation (deliberate, Class-3)
- **L6 event vocabulary** gains `workflow-transition` (recording `{from, to, edge-id, causing-event seq}` — the cause-carrying form that makes Register D the single-authority prover) and `model-turn` (structural facts + Ref to the output object); both hash-changing constitution amendments, not quiet additions.
- **L7 constitution** (code, hashed like L6's): control-verb vocabulary + signals, structural-turn-fact vocabulary, termination mapping, invariant-failure path, reserved approval vocabulary.

### Deliberate residuals (recorded, never silently promoted)
1. Approval channel: reserved fail-closed; its grill owns ingress authentication and the resumable-await question.
2. One orchestrator per state root (v1 deployment invariant); cross-process exclusion is the future mechanism.
3. Future budget dimensions (tokens/cost) enter via the declared-dimension vocabulary pattern.
4. L8 differently-trusted principals unchanged; L2 durable-record source kind unchanged.

## 3. Grill record (2026-09-07) — Q-L7-1..12, all CLOSED

1. **Q-L7-1 — Who defines the workflow (CLOSED):** governance defines the lattice; the model walks it; L7 enforces the walk; sole proposal channel = L4. → D-L7-1.
2. **Q-L7-2 — What is the plan (CLOSED):** not an artifact; δ + cursor-as-projection; deterministic-walk property. L7 is a workflow-state evaluator, not a planner. → D-L7-2.
3. **Q-L7-3 — Totality of δ (CLOSED):** owner's correction adopted — stay is an explicit governed edge, never an implicit fallback; floors outside the vocabulary; static totality at load. → D-L7-3.
4. **Q-L7-4 — Budgets and pacing (CLOSED):** independent enforcement, not trust propagation; static boundedness (counter-free cycles unloadable); CallState = structural monotonicity + record recount. → D-L7-4.
5. **Q-L7-5 — Model interaction (CLOSED):** structural turn facts + gate outcomes only; completion via registered control verb (the held opening question, discharged here); no-action semantics are per-phase governed edges. → D-L7-5.
6. **Q-L7-6 — Control-verb ownership (CLOSED):** constitution vocabulary; verb-per-signal, zero semantic arguments; `control` is classification, not authorization; v1 = `declare_done` only. → D-L7-6.
7. **Q-L7-7 — Failure taxonomy (CLOSED):** three natures; invariant failures structurally undeclarable; the on-error-edge laundering pattern rejected by unreachability. → D-L7-7.
8. **Q-L7-8 — Startup and recovery (CLOSED):** reconstruct-and-classify only; startup closes every non-terminal before accepting work; the case table derived, accepted. → D-L7-8.
9. **Q-L7-9 — Approval seam (CLOSED):** condition discharge, scope-bound, non-model-reachable ingress; v1 reserved fail-closed. → D-L7-9.
10. **Q-L7-10 — Router/service boundary (CLOSED):** three-verb seam, import lint, no defaulting; routing as pre-submission advisor. → D-L7-10.
11. **Q-L7-11 — Proof gate (CLOSED):** five registers + three-verdict acceptance + the single-authority property, made provable by cause-carrying transition events. → §4.
12. **Q-L7-12 — Turn/context lifecycle (CLOSED; the recovered draft orphan):** append-only phases, fresh composition at declared boundaries, model outputs as recorded first-class payloads — durable history, never authority. → D-L7-11.

**Locked layer principle:** *L7 is a deterministic executor of a governed workflow lattice — not a planner, not a policy engine, not a model controller, and not a second security-governance system. It owns sequence and pacing; every meaning lives in a governed artifact; every effect passes one gate; and for every transition in a completed record there exists exactly one governing edge and exactly one typed causing event.*

## 4. Test plan — five proof registers (Q-L7-11)

**Register A — structural:** public seam is exactly Open/SubmitTask/ReadStatus (API-closure audit); adapter import lint; no cursor-mutation/reopen/resume API; definitions immutable + hash-pinned; no plan/replan channel; constitution-closed control vocabulary, zero-argument verbs, two-way registry ⊆; invariant vocabulary undeclarable; definition ⊆ ceilings; no implicit transition; counter-free cycles unloadable; exhaustion edges statically required; no budget reset mechanism; δ consumes no content (typed-input signatures); declared-events-only filter; single-use identities; startup-before-work ordering.

**Register B — adversarial:** model prose/JSON → transition/termination/plan (all unproducible); control verb with invented arguments (whole-call invalid-args); undeclared event at δ (invariant); counter reset/overflow; retry-exhaustion bypass; budget inflation; workflow values exceeding grant/L5/L4 ceilings (load refusals, each pinned); illegal cycles; invariant-failure → workflow edge (unloadable); duplicate SubmitTask; task_id reuse; terminal continuation; **incomplete envelope through the adapter → typed assembly refusal naming the absence** (the no-defaulting proof, per missing reference class).

**Register C — crash/restart:** deterministic fault injection at every loop boundary — pre-transition-commit, post-commit/pre-action, post-action/pre-next-event, mid-SubmitTask assembly — plus the L6 commit-path points; universal assertions: no acknowledged transition disappears; no effect is represented without its preceding durable record; recovery never originates. Fault-point/source sync meta-test extended to the orchestration package. Real-kill: live task → SIGKILL → cold Open() → recovery → terminal; **no continuation**, proven.

**Register D — deterministic walk (the distinctive L7 proof):** same definition hash + same ordered typed-event sequence + same initial state ⇒ identical walk. The replayer re-derives every cursor position, transition, counter value, and terminal from (definition-at-hash, recorded events), checks each recorded `workflow-transition` event's `{edge-id, causing-event}` against its own derivation, and checks single-edge uniqueness. Divergence = CORRUPT/invariant, never best-effort. **Includes the Q-L7-12 completion criterion: the replayer reconstructs what the model actually saw at every turn** (phase composition + recorded model-turn objects + recorded results), byte-exact.

**Register E — live operational proof:** SubmitTask → Open/active L7 → model turns → L4 capabilities → typed results → δ → phase transitions → bounded retry/branch → `declare_done` → governed edge → terminal → L6 record → ReadStatus — against a real local model, through the production loop, not a test harness. Negative proofs live: prose cannot move the workflow; ungranted `declare_done` is not-available; `declare_done` cannot bypass the governed edge; budgets cannot be exceeded; invariant failure cannot enter the workflow; restart cannot resume; the adapter cannot bypass L7.

**Single-authority property (capstone, proven in D+B):** for every transition in a completed record there is exactly one governing definition edge and exactly one typed causing event; no transition is caused by model content, a transport request, a direct L6 mutation, or an implicit rule.

## 5. Review record (2026-09-07) — three Class-3 reviews, remediated

**Security review** — no CRITICAL/HIGH; six MED, all remediated with regressions: MED-1 cross-artifact task_id binding (spec/grant must bind to the envelope's task); MED-2 control-verb results appended to the conversation (protocol conformance); MED-3 runtime-producible events (`turns-exhausted`, `turn-no-action`, `turn-provider-error`) load-mandatory + control-verb signal declaration required; MED-4 `turns-exhausted` cannot target `@stay`; MED-5 startup sweep fails closed on ReadDir and discriminates CORRUPT from transient IO; MED-6 honest event source (`l7`, `eis+raw-payload-v1`). LOW: 64KiB payload cap.

**Test review** — one CRITICAL (real-kill test could silently skip its assertion path) remediated: child-error file, skip→fatal, retry_of asserted, ceiling raised so the child provably runs. HIGHs remediated, including the predicted phase-capability-narrowing hole: `phaseGrant()` now narrows the grant to phase capabilities at the gate, pinned by `TestPhaseCapabilityNarrowing`. ~13 regressions in `review_test.go`; suite 80.8% coverage, all green including live registers.

**Architecture review** — MEDs remediated: 1.2 governed `turn_timeout_sec` (no L7-authored constant); 1.3 `SubmitTask(path)` loads the envelope internally (no stale-hash execution); 2d field-scoped placeholder-only grant instantiation (literal workspace refused); 2e no-action prose appended to the conversation; 4.3 wall-clock budget floor (seal env-deadline → FAILED, never a workflow edge).

**Architecture HIGH 2a — CLOSED (owner decision 2026-09-07: implement, not narrow):** phase composition now runs the full L2 pipeline. The envelope gains the governed `context_contract_path` reference (required, no defaulting); assembly loads the contract fail-closed and binds it to the workflow (`contract.Workflow == wf.Name`, cross-workflow contracts refuse); `composePhase` runs Gather→Compose — Plan ⊆ Contract, content-derived fencing, provenance labels (`authority: external-untrusted`), typed absence — so the untrusted payload never reaches the conversation raw. The composed user message's exact bytes are the stored delivery object (byte-exact D-L7-11 reconstruction); the delivery event records `l2-composed-v1` + contract/render/payload hashes; `context_contract` joins the manifest attribution. Governed artifact: `policies/context/task-contract-v1.json`. Pinned by `TestComposedDeliveryThroughL2` + `TestContractWorkflowBindingRefused`; live proof re-run green with fenced delivery (qwen2.5:7b, 25s).

**Acceptance gate:** architecture-conformant + test-evidenced + operationally proven — independently challenged, all three.

## 6. Post-archive conformance audit (owner, 2026-09-07) — Register-D strengthening

Owner code-vs-architecture audit found the loop CONFORM but the Register-D proof weaker than the locked single-authority property: (1) the transition event's `edge` field carried the event name, leaving edge identity implicit in the loader's duplicate-On refusal, and leaving the countered edge's normal-vs-exhaustion branch derivable but unrecorded; (2) the replayer checked `cause_seq` only for precedence and compared from/to sequences — sequence correspondence, not causal correspondence — and never re-derived `turns-exhausted` or `tool-error` walks. Remediation (owner-directed, all four): `step()` records `{from, to, edge_id: "<from>/<on>", exhausted, cause_seq}`; the replayer derives full tuples — mirroring the loop's lastSeq discipline — and asserts `recorded.cause_seq == derived.cause_seq` (a valid-but-wrong earlier seq fails); replay covers every transition-producing class including turns-exhausted (re-derived by per-visit turn counting) and tool-error (exhaustion-branch identity asserted); Register A pins the loader refusal as the edge-identity invariant (`TestEdgeIdentityUnique`, refusal + positive uniqueness halves). All registers re-run green, live proof re-passed (qwen2.5:7b), coverage 80.9%.
