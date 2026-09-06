# Design: Layer 4 — Tool Interface

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §7 (layer doc), the archived L1–L3 designs (their invariants bind — especially L1 Scenario 3, Q-L1-2 amendment 1, Q-L2-1's three verbs, Q-L2-2), `ARCHITECTURE.md`, the Model Interface (`runtime/model` ToolCall/ToolDef seam, already proven).
**Decision IDs** `D-L4-n`; **open questions** `Q-L4-n`. DEC-05: nothing provider-specific; the layer doc's DeepSeek framing is illustrative.

## 0. Position in the flow

```
Model ──ToolCall (advisory data)──► L7 considers (future: loop/sequencing)
                                        │
                                        ▼
                              ┌───────────────────┐
                              │ L4 AUTHORIZE      │
                              │ schema validation │
                              │ grant ⊆ registry  │
                              │ target validation │
                              └─────────┬─────────┘
                              denied ◄──┤──► authorized
                              (typed,   │        │
                               recorded,│        ▼
                               data)    │  registered executor (v1: read-only,
                                        │  confined; L5-era: sandboxed)
                                        │        │
                                        ▼        ▼
                                     audit event + result-as-evidence
                                     (trust class from registration)
```

## 1. Hard invariants (inherited, not grillable)

- **L4 independence:** authorization is computed from capability/policy state independent of model output and task-level instructions (Q-L1-2, owner-locked). Model confidence, phrasing, persistence, or claimed approval never move the decision.
- **Closed vocabulary:** the registry embodies the authority boundary. No verb mutates Themis security state; schemas carry the operation's subject, never its authority disposition; no caller-supplied field determines the governance treatment of its own request (L1 Scenario 3).
- **Unknown ⇒ hard denial, recorded** (unknown tool, unknown field, unknown target class). Fail closed on any control failure (3.9); never fall back to model judgment.
- **Three verbs:** L7 considers, L4 authorizes, execution fulfills — L4 never sequences, retries, or judges usefulness; it answers exactly "is this capability authorized for this execution?"
- **Results are data:** tool output re-enters context under the L2 discipline (registered trust class, verbatim bytes, framing, hash, trace) — a tool result containing imperative text is evidence, not instructions.
- **Least privilege (3.4):** grants are per-task, minimal, explicit; nothing is ambiently available.
- **Uniform gate:** every acquisition path (planned context, model-requested expansion, any future connector) passes this same authorization — no privileged bypass.
- Determinism: same registry + grants + request ⇒ same decision; every decision auditable and reconstructable.

## 2. Draft decisions (grill targets)

### D-L4-1 — Capability registry is a governed artifact

`ToolRegistry` (versioned file, hash in trace): per tool — name, description, input schema (strict), output schema, permission class, target rules, timeout, audit behavior, **result trust class** (the L2/Q-L2-10 classification fixed at registration: workspace reads ⇒ `external-untrusted`; Themis reads ⇒ `governed-record`/`governed-external`; registered analyzers ⇒ `derived`). Registration is Class-3 review; output vocabulary and jurisdiction are judged there (Q-L2-6 discipline applied to tools).

### D-L4-2 — Grants: per-task allowlist ⊆ registry

`TaskGrant` artifact: the tools (and target scopes) this execution may use. Grant ⊄ registry ⇒ fail closed. Mirrors Plan ⊆ Contract. Ownership question → Q-L4-1.

### D-L4-3 — The authorization function

`Authorize(registry, grant, request) → Decision{allow|deny, reason, audit}` — pure, deterministic, no I/O, no clock beyond injected inputs. Inputs are exactly (registered tool def, grant entry, request args, target). Nothing else is readable by construction (the L3 type-boundary trick applied to authorization).

### D-L4-4 — Strict schema validation

JSON schema per tool, `DisallowUnknownFields` posture: unknown field ⇒ whole-call rejection, recorded (never silent stripping — the L1 Scenario 3 rule verbatim). Args are size-capped. Validation happens before authorization reads anything from args.

### D-L4-5 — Target validation is per-class deterministic rules

Filesystem targets: L2's `confinedPath` (symlink-escape refusal, workspace root from the task). Themis targets: id-shape validation against the typed seam. Target rules live in the registry entry; v1 classes: `workspace-path`, `themis-id`, `none`.

### D-L4-6 — Executors are registered, dispatch is dumb

`Executor` interface keyed by tool name; v1 ships read-only executors (filesystem trio via L2 confinement; Themis read stubs via `ThemisReader`-style seam). Dispatch = table lookup after authorization; no reflection, no dynamic anything. Mutating/shell executors are absent from the table, not merely denied.

### D-L4-7 — Results and denials both become typed evidence

Success: `ToolResult{tool, args-hash, evidence bytes, trust class, producer, hash}` → framed into the conversation as an L2-disciplined item (tool-execution trace entry with the Q-L2-1 amendment-4 completeness: request, decision, source, class, content hash). Denial: `ToolDenial{tool, reason-class}` → model-visible as typed data (minimum-disclosure question → Q-L4-5), trace-visible in full.

### D-L4-8 — Audit event per call, shape defined for L6

`{request(tool, args hash), decision + reason, registry hash, grant hash, target, executor id, result hash, timing}` — joins the delivery trace so model-visible bytes remain jointly reconstructable.

### D-L4-9 — Deferred by name

`run_command` (OPEN-2 + L5), all mutating tools (L5 worktrees + approval gates), sandboxing (L5), loop/retry (L7), durable audit (L6), approval-gated human-in-the-loop tools (governance-plane era).

## 3. Grill record (2026-09-06)

### Q-L4-1 — Grant ownership (CLOSED, locked)

Owner-locked invariants:

1. **Themis owns the workflow authorization ceiling.**
2. **L7 may narrow but never expand that ceiling.**
3. **`ExecutionGrant ⊆ WorkflowCapabilityCeiling ⊆ CapabilityRegistry`** — architectural invariant, not implementation convention; violation fails closed.
4. **No per-task operator authorization override exists in L4 v1.** Operator-driven capability change happens outside the runtime authorization path, through governed Themis configuration producing a new reviewed contract/grant (the L3 budget-override precedent). The override channel's prerequisites (operator authn, scope, expiry, approval semantics, emergency access, audit, revocation, separation of duties) are governance-era.

Supporting constraints (owner-locked):

- **No inference:** L4 must not infer grants from tool names, model requests, task text, or L7 intent — it evaluates an already-authorized capability identity against registry + execution grant, nothing else.
- **Closed vocabulary, no dead entries:** absent from the registry/dispatch table = does not exist as an executable capability. A mutating operation is never made "safe" via `allowed=false` — if it is not a v1 capability it is not in the executable vocabulary at all.

Ownership chain: Themis decides the ceiling → L7 selects/narrows execution scope → L4 authorizes the invocation → capability executes → result returns as L2-classified evidence.

### Q-L4-5 — Denial disclosure (CLOSED, locked with amendments)

Governing sentence (owner-locked): **L4 exposes actionable capability state, not authorization-policy state.** `not-available` means "this execution cannot invoke it," never "it does not exist."

The locked invariants:

1. **Four model-visible denial classes:** `not-available` · `invalid-args(field)` · `target-refused(target)` · `error`.
2. **Unknown ∪ not-granted → `not-available`** (registry-enumeration oracle closed; no contradiction with L2 typed absence — evidence-plane honesty ≠ capability-plane state: L2 answers "what is the state of the security world," L4 answers "what can this execution do").
3. **Authorization precedes argument validation** — the decisive anti-oracle rule: an unavailable capability yields `not-available` and NEVER exposes argument-validation information (probing `unknown_tool{invented_field}` or ungranted tools with bad args reveals nothing about schemas or existence). The full decision table:

| Capability state | Args | Model-visible |
| --- | --- | --- |
| Unknown | anything | `not-available` |
| Known, not granted | anything | `not-available` |
| Known + granted | invalid | `invalid-args(field)` |
| Known + granted | valid | execute |
| Known + granted | target rejected | `target-refused(target)` |
| Known + granted | executor fails | `error` |

4. **`invalid-args(field)` applies only after availability is established** and may identify only schema information already exposed to the model in the authorized capability's ToolDef.
5. **`target-refused(target)`** echoes only the canonicalized, bounded target reference necessary to identify the rejected request — never the rule, topology, policy threshold, or confinement boundary; ordinary error-size and sensitive-value bounds apply to the echo.
6. **Full denial mechanics are trace-only:** exact failed predicate, registry/grant/ceiling hashes, capability identity, authorization state, policy versions, full target, timing/decision metadata.
7. **Denials are deterministic within a fixed execution authorization/policy snapshot** (same canonical request ⇒ same model-visible denial; legitimate cross-execution grant changes may legitimately change outcomes).
8. **No authorization thresholds or governance state are model-visible** — ever, in any class.
9. **`requires-approval` is an internal/reserved governance state rendering as `not-available` in v1** — exposing pending-approval would invite the model into the governance process ("ask the human to hurry the approval") rather than keeping it inside its capability boundary. Trace records the true state and failed predicate.
10. **No intentional timing distinctions** based on hidden authorization-policy state; ordinary system timing variation is operational noise — no artificial constant-time machinery in v1.

### Q-L4-2 — Registry vs grant granularity (CLOSED)

Registry = legal capability shapes (target *classes* + validation rules); grant = per-execution instance bindings (workspace root, Themis id scope). **`EffectiveAuthorization = RegistryCapability ∩ ExecutionGrant ∩ TargetBinding`** — neither side may widen the other. Worked examples locked: registered capability + out-of-scope target ⇒ `target-refused`; in-scope workspace + unregistered capability ⇒ `not-available`. **The model supplies the requested target; the grant supplies the authoritative scope binding; the model cannot supply or override the binding.**

### Q-L4-3 — Result trust classes (CLOSED with the hard rule)

Registration table: workspace reads ⇒ `external-untrusted` · Themis reads ⇒ `governed-record`/`governed-external` per endpoint · registered computations (full Q-L2-6 provenance, Class-3 reviewed) ⇒ potentially `derived` · unregistered claimed derivation ⇒ `external-untrusted`. **A tool cannot promote its own output to `derived`** — any authority-class field in tool output is untrusted metadata, ignored. v1 ships zero derived tools; the seam waits for L5-era analyzers.

### Q-L4-4 — Quotas (CLOSED)

Hard per-tool/total call caps live in the execution grant; exhaustion ⇒ `not-available` (model-visible), `quota-exhausted` + counts trace-only. L4 answers "within the cap?"; L7 owns retry/wait/backoff/terminate. **The quota belongs to the grant: L7 may decline to spend remaining quota but cannot alter the ceiling.**

### Q-L4-6 — Approval-gated tools (CLOSED as corollary of Q-L4-5 §9)

Decision vocabulary reserves `requires-approval`; it renders `not-available` in v1; the approval channel is governance-era.

### Q-L4-7 — Idempotency/replay (CLOSED)

`Authorize(registry, grant, request, call_count) → Decision` — pure, no internal mutable counters; execution state (call counts, loop history, dedup, idempotency policy) is supplied by and owned by L7. The L3 input-bounded discipline preserved for testability and reproducibility.

### Q-L4-8 — Executor errors (CLOSED)

Authorization failure ⇒ denial; authorized operation + world failure ⇒ **typed error evidence** (`error(class)` — file-unreadable, seam-unavailable, timeout; deliberately non-topological; full detail trace-only). L4 reports the deterministic outcome; L7 decides retry/continue/fail. Never rendered as `not-available`, never silent.

### Q-L4-9 — Operational proof (CLOSED WITH ACTION, owner-authorized)

Owner decision: **pull `qwen2.5-coder:7b`** as test equipment for the seam — explicitly not a model standardization (DEC-05; the model is replaceable behind the Model Interface). Rationale: mock fixtures prove the deterministic pipeline but not that a real provider produces the tool-call representation the Model Interface expects — an empirical, already-observed integration risk (WhiteRabbitNeo's tool rejection). Live proof must exercise: authorized call · `not-available` · `invalid-args(field)` · `target-refused(target)` · `error` · result re-entering L2 as classified evidence. Deterministic negative cases may be driven via controlled prompts/fixtures — the acceptance criterion is the protocol seam, not model cleverness. Gate: mock proof ✓ required; live cases pending the download + implementation.

### Grill board — ALL CLOSED 2026-09-06

Q1 ✅ locked · Q2 ✅ · Q3 ✅ hard rule · Q4 ✅ · Q5 ✅ locked w/ amendments · Q6 ✅ corollary · Q7 ✅ · Q8 ✅ · Q9 ✅ with authorized action. Governing sentences: **L4 exposes actionable capability state, not authorization-policy state** · `ExecutionGrant ⊆ WorkflowCeiling ⊆ Registry` · authorization precedes argument validation · absent from the vocabulary = does not exist.

## 4. Draft open questions (SUPERSEDED — resolved by the grill record above)

1. **Q-L4-1 — Grant ownership:** who authors the per-task grant — Themis workflow definition (ceiling) + L7 instantiation (⊆), mirroring Q-L2-1's split? Can L7 narrow but never widen? Is there a per-task operator override channel (exemption posture) or none in v1?
2. **Q-L4-2 — Registry vs grant granularity:** are target scopes (e.g. which paths, which Themis ids) part of the grant, the registry, or both (registry = legal shapes, grant = task instances)?
3. **Q-L4-3 — Result trust classes:** confirm the registration table — workspace reads external-untrusted, Themis reads governed, analyzers derived. Is a tool ever allowed to *declare* derived without the full Q-L2-6 computational provenance?
4. **Q-L4-4 — Quotas:** per-task call-count/time budgets — L4 (part of the grant) or L7 (loop discipline)? Proposal: hard per-task caps in the grant (fail closed), pacing in L7.
5. **Q-L4-5 — Denial disclosure:** what does the model learn from a denial? Proposal: reason *class* only (`not-granted`, `invalid-args`, `target-refused`) — enough to adapt, no policy internals; full reasons trace-only (the framing-refusal precedent).
6. **Q-L4-6 — Approval-gated tools:** is there a v1 seam for "authorized but requires human approval before execution" (async approval state), or is that wholly governance/L7-era? Proposal: define the decision vocabulary now (`allow | deny | requires-approval`), implement only allow/deny paths.
7. **Q-L4-7 — Idempotency/replay:** does L4 dedupe identical tool calls, or is replay an L7 concern? Proposal: L4 is stateless per call (purity); replay control is L7's loop.
8. **Q-L4-8 — Error surface:** executor runtime failure (file unreadable, seam down) — typed tool-error evidence to the model (data) vs run-level failure? Proposal: typed error result (source-status/delivery-status discipline), never a silent empty.
9. **Q-L4-9 — Operational proof:** requires a tools-capable local model — the recorded Phase-0 spike showed WhiteRabbitNeo has no tool support. **Owner decision needed: pull a tools-capable model (download) or accept mock-provider proof for v1.** Deterministic proof regardless: full pipeline against the scripted mock providers (the Model Interface's proven tool-call fixtures).

## 4. Interfaces to neighbors

- **← Model Interface:** consumes `model.ToolCall` (id, name, raw args) and produces `model.Message{Role: tool}` results — the seam proven at the Model Interface milestone.
- **← L2/L3:** results framed as classified evidence; capability-fetch mechanism (`MechanismCapabilityFetch`, reserved since L2) finally gets its producer.
- **→ L5:** executor sandbox boundary; mutating tools arrive there behind this same authorization.
- **→ L6:** audit event shape. **→ L7:** the considers/authorizes split; loop, retries, quota pacing.

## 5. Test plan (three-state discipline)

Registry/grant fail-closed loaders (artifact posture); authorization table tests (ungranted, unregistered, unknown-field whole-call rejection, authority-disposition field rejection — the literal Scenario-3 regression: `requires_human_decision` in args dies at schema); target confinement (symlink escapes via tool path); grant ⊄ registry; denial typing + minimum disclosure; result classification + framing through L2; audit completeness (every call, every denial); determinism; end-to-end through the mock tool-calling providers; live proof per Q-L4-9 decision.
