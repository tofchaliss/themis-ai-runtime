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

## 3. Open questions for the grill (Q-L4-n)

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
