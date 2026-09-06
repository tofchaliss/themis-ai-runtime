# Tasks: Layer 4 — Tool Interface

Execution starts only after the grill closes Q-L4-1…9 and the owner accepts. Every milestone review states three independent verdicts: architecture-conformant · test-evidenced (coverage-verified on claimed lines) · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held 2026-09-06; Q-L4-1..9 closed and recorded in design.md §3
- [x] Design accepted by owner (gate taken as accepted 2026-09-06 with "I am waiting on it" following the proposed accept→smoke→build→proof course; explicitly recorded per arch review F8 — say the word if this reading was wrong)
- [x] Owner decision: pull qwen2.5-coder:7b (authorized 2026-09-06) — **pulled; FAILED the tool-protocol smoke** (tool JSON as content, structured tool_calls null; recorded in design §3 Q-L4-9). Live proof re-pending a new owner model decision (qwen2.5:7b / llama3.1:8b / defer)

## 1. L4-M1 — Registry + grants (Class 3) — **DONE 2026-09-06**

- [x] `ToolRegistry` artifact + fail-closed loader — **narrowed from D-L4-1** (arch review F2): ships schemas (typed params), target rules, timeout (now enforced per call), result trust class; permission-class, output-schema, and audit-behavior fields deferred until a consumer exists
- [x] `TaskGrant` artifact; grant ⊆ registry enforcement, fail closed
- [x] Registry/grant hashes into the trace
- [x] Security review done (see §3a)

## 2. L4-M2 — Authorization pipeline (Class 3) — **DONE 2026-09-06**

- [x] `Authorize(registry, grant, request)` — pure, deterministic, input-bounded by construction
- [x] Strict schema validation: unknown field ⇒ whole-call rejection, recorded (Scenario-3 regression: authority-disposition args die here)
- [x] Target validation per class (workspace-path via confinedPath; themis-id shape)
- [x] Typed denials per Q-L4-5 (reason class model-visible, full reason trace-only)
- [x] Decision vocabulary incl. `requires-approval` — reserved as a code constant (`DecisionRequiresApproval`, nothing produces it; renders not-available when the governance channel exists)
- [x] Security review done (see §3a)

## 3. L4-M3 — Executors + result evidence (Class 3) — **DONE 2026-09-06**

- [x] Read-only executors: `read_file`, `list_directory`, `search_code` (L2 confinement), Themis read stubs (typed seam)
- [x] Dispatch table (absent ⇒ impossible, not denied); no mutating/shell entries exist
- [x] Results as classified evidence (registration trust class, verbatim bytes, hash) framed via the L2 discipline; typed error results per Q-L4-8
- [x] Audit event per call/denial (D-L4-8 shape), documented for L6
- [x] Security review folded (see §3a)

## 3a. Security review record (M1-M3, 2026-09-06)

Class-3 review with PoCs. Verified holding: anti-oracle check order (availability complete before any arg byte is read; zero model detail on not-available; registry-vs-grant distinction trace-only), no JSON injection via echoes, unknown-field whole-call rejection with the literal Scenario-3 regression, grant ⊆ registry at use, single confinement implementation, trust fixed at registration (self-promotion impossible), no topology leakage, deterministic denials, fail-closed loaders. Findings remediated: **F1** unbounded list_directory evidence (repo-steerable context flooding) → capped; **F2** audit incompleteness → denial paths now record requested target + exact model detail; timing/executor-id explicitly deferred to L6; **F3** registry/table drift panicked on the authorized path → typed error + complete audit; **F4** search silently skipped oversized files (adversarial repos could hide indicators behind padding) → skip counts surfaced, empty-but-hidden refuses typed; **F5** forbidden-param stems widened (structural charset already blocks case/unicode variants); **F6** relative workspace bindings (cwd-dependent confinement identity) → absolute required at load; **F9** missing workspace binding reclassified not-available (config error, not target fault). Accepted/deferred: TOCTOU (L5 threat model), Themis prefix coarseness (documented delimiter convention), **mandatory L7 acceptance test: quota enforcement exists only if the loop increments CallState between calls**.

## 3b. Close-out review record (2026-09-06)

**Test review:** architecture-conformant PASS · test-evidenced FAIL→closed: the headline was the no-op lesson in new form — 9/14 loader fail-closed cases passed on the wrong branch (tool name "a" died at the ≥2-char name regex first), so the derived-trust hard rule and F5 stems had zero real evidence. Rewritten with per-case branch-substring assertions + a ≥2-char pin; all vocabulary branches (integer/boolean/optional/TargetNone), seam/scope/cap edges, CallState immutability, timeout enforcement, bound() rune-split, granted-but-unregistered, and the provider round-trip added. 97.4% coverage. · operationally proven PENDING (honest; see gate 0).

**Architecture review:** conformant, no boundary violations. Remediated: timeout was asserted-but-dead config → enforced per call with typed `timeout` error (goroutine-linger recorded as L5 deferral); requires-approval reservation made code; purity claim amended (input-bounded except confinement's necessary filesystem consult); grant⊆registry at-use asymmetry recorded as deliberate; M1 checkbox narrowed honestly. **PENDING OWNER: ARCHITECTURE.md gate-ordering reconciliation (design §3 Q-L4-9 note) — architecture-authority class.**

## 4. L4-M4 — Seams + proof (Class 2)

- [x] Model Interface loop: `TestProviderRoundTrip` — scripted provider emits structured tool_calls → parse → Authorize → execute → tool message → second-turn completion, through the real seam
- [ ] Capability-fetch mechanism wired to the L2 envelope (`MechanismCapabilityFetch` gets its producer); joint trace reconstruction stated
- [ ] Operational proof per Q-L4-9 owner decision
- [ ] Traceability table, coverage-verified

## 5. Deferred dependencies (NOT L4 scope)

- **OPEN-2 + L5:** `run_command`, sandboxed execution, all mutating tools (write/patch/commit/create/attach), approval-gate execution path
- **L6:** durable audit sink
- **L7:** request loop, retries, quota pacing, considers-step, epoch placement of tool results
- **Governance plane:** human approval channel for `requires-approval` decisions

## 6. Close

- [ ] Architecture + security + test review, three-state verdicts
- [ ] Code map updated; layer-status doc extended to L4; green checkpoints pushed on owner approval; archive on close
